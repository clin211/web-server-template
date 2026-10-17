package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Update 实现 UserBiz 接口中的 Update 方法.
func (b *userBiz) Update(ctx context.Context, rq *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error) {
	userM, err := b.store.User().Get(ctx, where.F("user_id", rq.GetUserID()))
	if err != nil {
		return nil, err
	}

	if err := b.applyUniqueUserFields(ctx, rq, userM); err != nil {
		return nil, err
	}

	if rq.Nickname != nil {
		userM.Nickname = rq.GetNickname()
	}

	updates := selectedUserUpdates(rq, userM)
	if len(updates) == 0 {
		return &v1.UpdateUserResponse{}, nil
	}
	if err := b.store.TX(ctx, func(txCtx context.Context) error {
		if err := b.store.User().Update(txCtx, userM.UserID, updates); err != nil {
			return fmt.Errorf("update user: %w", err)
		}
		if err := b.store.AuditLog().Create(txCtx, &model.AuditLogM{UserID: contextx.UserID(ctx), Action: "user.update", Resource: &userM.UserID, CreatedAt: time.Now().UTC()}); err != nil {
			return fmt.Errorf("create user update audit log: %w", err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("update user transaction: %w", err)
	}

	return &v1.UpdateUserResponse{}, nil
}

func selectedUserUpdates(rq *v1.UpdateUserRequest, userM *model.UserM) map[string]any {
	updates := make(map[string]any)
	if rq.Username != nil {
		updates["username"] = userM.Username
	}
	if rq.Nickname != nil {
		updates["nickname"] = userM.Nickname
	}
	if rq.Email != nil && rq.GetEmail() != "" {
		updates["email"] = rq.GetEmail()
	}
	if rq.Phone != nil && rq.GetPhone() != "" {
		updates["phone"] = rq.GetPhone()
	}
	return updates
}

func (b *userBiz) applyUniqueUserFields(ctx context.Context, rq *v1.UpdateUserRequest, user *model.UserM) error {
	if rq.Username != nil && rq.GetUsername() != user.Username {
		if err := checkUserUniqueness(ctx, b.store, "username", rq.GetUsername(), user.UserID); err != nil {
			return err
		}
		user.Username = rq.GetUsername()
	}
	if err := updateUniqueContact(ctx, b.store, user.UserID, "email", rq.Email, &user.Email); err != nil {
		return err
	}
	return updateUniqueContact(ctx, b.store, user.UserID, "phone", rq.Phone, &user.Phone)
}

func updateUniqueContact(ctx context.Context, core store.IStore, userID, name string, requested *string, current **string) error {
	// 空联系方式沿用现有 PATCH 语义，表示不更新，不清空原字段。
	if requested == nil || *requested == "" || (*current != nil && *requested == **current) {
		return nil
	}
	if err := checkUserUniqueness(ctx, core, name, *requested, userID); err != nil {
		return err
	}
	*current = requested
	return nil
}

func checkUserUniqueness(ctx context.Context, core store.IStore, name, value, userID string) error {
	existing, err := core.User().Get(ctx, where.F(name, value).L(1))
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("check %s uniqueness: %w", name, err)
	}
	if existing != nil && existing.UserID != userID {
		slog.WarnContext(ctx, "User field already exists", "field", name, "value", value)
		return errno.ErrUserAlreadyExists
	}
	return nil
}
