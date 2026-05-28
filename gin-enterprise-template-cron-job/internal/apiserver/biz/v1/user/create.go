package user

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jinzhu/copier"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/clin211/gin-enterprise-template/internal/pkg/known"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// Create 实现 UserBiz 接口中的 Create 方法.
func (b *userBiz) Create(ctx context.Context, rq *v1.CreateUserRequest) (*v1.CreateUserResponse, error) {
	var userM model.UserM
	if err := copier.Copy(&userM, rq); err != nil {
		slog.ErrorContext(ctx, "Failed to copy request to model", "error", err)
		return nil, fmt.Errorf("failed to copy request: %w", err)
	}

	// 检查用户名是否已存在
	if existingUser, err := b.store.User().Get(ctx, where.F("username", userM.Username).L(1)); err == nil && existingUser != nil {
		slog.WarnContext(ctx, "Username already exists", "username", userM.Username)
		return nil, errno.ErrUserAlreadyExists
	}

	// 检查邮箱是否已存在（如果提供了邮箱）
	if userM.Email != nil && *userM.Email != "" {
		if existingUser, err := b.store.User().Get(ctx, where.F("email", *userM.Email).L(1)); err == nil && existingUser != nil {
			slog.WarnContext(ctx, "Email already exists", "email", *userM.Email)
			return nil, errno.ErrUserAlreadyExists
		}
	}

	// 检查手机号是否已存在（如果提供了手机号）
	if userM.Phone != nil && *userM.Phone != "" {
		if existingUser, err := b.store.User().Get(ctx, where.F("phone", *userM.Phone).L(1)); err == nil && existingUser != nil {
			slog.WarnContext(ctx, "Phone already exists", "phone", *userM.Phone)
			return nil, errno.ErrUserAlreadyExists
		}
	}

	if err := b.store.User().Create(ctx, &userM); err != nil {
		return nil, err
	}

	if _, err := b.authz.AddGroupingPolicy(userM.UserID, known.RoleUser); err != nil {
		slog.ErrorContext(ctx, "Failed to add grouping policy for user", "user", userM.UserID, "role", known.RoleUser, "error", err)
		return nil, errno.ErrAddRole.WithMessage(err.Error())
	}

	return &v1.CreateUserResponse{UserID: userM.UserID}, nil
}
