package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jinzhu/copier"
	"gorm.io/gorm"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/clin211/gin-enterprise-template/internal/pkg/rid"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/authn"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// Create 实现 UserBiz 接口中的 Create 方法.
func (b *userBiz) Create(ctx context.Context, rq *v1.CreateUserRequest) (*v1.CreateUserResponse, error) {
	var userM model.UserM
	if err := copier.Copy(&userM, rq); err != nil {
		slog.ErrorContext(ctx, "Failed to copy request to model", "error", err)
		return nil, fmt.Errorf("failed to copy request: %w", err)
	}

	// 唯一性预检查用于尽早返回友好业务错误，不能替代数据库唯一约束；并发请求仍可能
	// 同时通过检查，最终一致性必须由数据库约束兜底。
	// 检查用户名是否已存在
	if existingUser, err := b.store.User().Get(ctx, where.F("username", userM.Username).L(1)); err == nil && existingUser != nil {
		slog.WarnContext(ctx, "Username already exists", "username", userM.Username)
		return nil, errno.ErrUserAlreadyExists
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check existing username: %w", err)
	}

	// 检查邮箱是否已存在（如果提供了邮箱）
	if userM.Email != nil && *userM.Email != "" {
		if existingUser, err := b.store.User().Get(ctx, where.F("email", *userM.Email).L(1)); err == nil && existingUser != nil {
			slog.WarnContext(ctx, "Email already exists", "email", *userM.Email)
			return nil, errno.ErrUserAlreadyExists
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("check existing email: %w", err)
		}
	}

	// 检查手机号是否已存在（如果提供了手机号）
	if userM.Phone != nil && *userM.Phone != "" {
		if existingUser, err := b.store.User().Get(ctx, where.F("phone", *userM.Phone).L(1)); err == nil && existingUser != nil {
			slog.WarnContext(ctx, "Phone already exists", "phone", *userM.Phone)
			return nil, errno.ErrUserAlreadyExists
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("check existing phone: %w", err)
		}
	}

	userM.UserID = rid.UserID.MustNew()
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
