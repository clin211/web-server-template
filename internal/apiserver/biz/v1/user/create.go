package user

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jinzhu/copier"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/pkg/rid"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/authn"
)

// Create 实现 UserBiz 接口中的 Create 方法.
func (b *userBiz) Create(ctx context.Context, rq *v1.CreateUserRequest) (*v1.CreateUserResponse, error) {
	var userM model.UserM
	if err := copier.Copy(&userM, rq); err != nil {
		slog.ErrorContext(ctx, "Failed to copy request to model", "error", err)
		return nil, fmt.Errorf("failed to copy request: %w", err)
	}

	// 预检查提供友好的业务错误；最终一致性仍由数据库唯一约束保证。
	if err := b.checkCreateUniqueness(ctx, &userM); err != nil {
		return nil, err
	}

	userM.UserID = rid.UserID.NewContext(ctx)
	password, err := authn.Encrypt(rq.GetPassword())
	if err != nil {
		return nil, fmt.Errorf("encrypt user password: %w", err)
	}
	userM.Password = password

	if err := b.store.User().Create(ctx, &userM); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &v1.CreateUserResponse{UserID: userM.UserID}, nil
}

func (b *userBiz) checkCreateUniqueness(ctx context.Context, user *model.UserM) error {
	fields := map[string]string{"username": user.Username}
	if user.Email != nil && *user.Email != "" {
		fields["email"] = *user.Email
	}
	if user.Phone != nil && *user.Phone != "" {
		fields["phone"] = *user.Phone
	}
	// 固定检查顺序，确保多个字段冲突时返回的错误保持一致。
	for _, name := range []string{"username", "email", "phone"} {
		value, exists := fields[name]
		if !exists {
			continue
		}
		if err := checkUserUniqueness(ctx, b.store, name, value, ""); err != nil {
			return err
		}
	}
	return nil
}
