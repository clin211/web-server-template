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

	// 以下查询只为提前返回可读错误；并发更新的唯一性仍由数据库约束保证。
	// 检查用户名是否已被其他用户占用
	if rq.Username != nil && rq.GetUsername() != userM.Username {
		if existingUser, err := b.store.User().Get(ctx, where.F("username", rq.GetUsername()).L(1)); err == nil && existingUser != nil && existingUser.UserID != userM.UserID {
			slog.WarnContext(ctx, "Username already exists", "username", rq.GetUsername())
			return nil, errno.ErrUserAlreadyExists
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("check username uniqueness: %w", err)
		}
		userM.Username = rq.GetUsername()
	}

	// 检查邮箱是否已被其他用户占用
	if err := checkUniqueField(ctx, b.store, userM, rq.Email, "email", rq.GetEmail); err != nil {
		return nil, err
	}

	// 检查手机号是否已被其他用户占用
	if err := checkUniqueField(ctx, b.store, userM, rq.Phone, "phone", rq.GetPhone); err != nil {
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

// checkUniqueField 检查唯一字段（如 email、phone）是否已被其他用户占用.
func checkUniqueField[T any](
	ctx context.Context,
	store store.IStore,
	userM *model.UserM,
	fieldPtr *T,
	fieldName string,
	getValue func() string,
) error {
	// 空字符串沿用当前 PATCH 语义：表示“不更新”，不是清空字段。若需要支持清空，
	// API 必须引入明确的 presence/clear 契约，不能复用这个分支。
	value := getValue()
	if value == "" {
		return nil
	}

	// 获取当前值
	var currentValue *string
	switch fieldName {
	case "email":
		currentValue = userM.Email
	case "phone":
		currentValue = userM.Phone
	}

	// 如果值没有变化，跳过检查
	if currentValue != nil && value == *currentValue {
		return nil
	}

	// 检查是否已被其他用户占用
	existingUser, err := store.User().Get(ctx, where.F(fieldName, value).L(1))
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("check %s uniqueness: %w", fieldName, err)
	}
	if existingUser != nil && existingUser.UserID != userM.UserID {
		slog.WarnContext(ctx, fmt.Sprintf("%s already exists", fieldName), fieldName, value)
		return errno.ErrUserAlreadyExists
	}

	// 更新字段值
	switch fieldName {
	case "email":
		email := value
		userM.Email = &email
	case "phone":
		phone := value
		userM.Phone = &phone
	}

	return nil
}
