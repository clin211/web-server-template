package user

import (
	"context"
	"log/slog"

	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Update 实现 UserBiz 接口中的 Update 方法.
func (b *userBiz) Update(ctx context.Context, rq *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error) {
	userM, err := b.store.User().Get(ctx, where.T(ctx))
	if err != nil {
		return nil, err
	}

	// 检查用户名是否已被其他用户占用
	if rq.Username != nil && rq.GetUsername() != userM.Username {
		if existingUser, err := b.store.User().Get(ctx, where.F("username", rq.GetUsername()).L(1)); err == nil && existingUser != nil && existingUser.UserID != userM.UserID {
			slog.WarnContext(ctx, "Username already exists", "username", rq.GetUsername())
			return nil, errno.ErrUserAlreadyExists
		}
		userM.Username = rq.GetUsername()
	}

	// 检查邮箱是否已被其他用户占用
	if rq.Email != nil && rq.GetEmail() != "" && (userM.Email == nil || rq.GetEmail() != *userM.Email) {
		if existingUser, err := b.store.User().Get(ctx, where.F("email", rq.GetEmail()).L(1)); err == nil && existingUser != nil && existingUser.UserID != userM.UserID {
			slog.WarnContext(ctx, "Email already exists", "email", rq.GetEmail())
			return nil, errno.ErrUserAlreadyExists
		}
		email := rq.GetEmail()
		userM.Email = &email
	}

	// 检查手机号是否已被其他用户占用
	if rq.Phone != nil && rq.GetPhone() != "" && (userM.Phone == nil || rq.GetPhone() != *userM.Phone) {
		if existingUser, err := b.store.User().Get(ctx, where.F("phone", rq.GetPhone()).L(1)); err == nil && existingUser != nil && existingUser.UserID != userM.UserID {
			slog.WarnContext(ctx, "Phone already exists", "phone", rq.GetPhone())
			return nil, errno.ErrUserAlreadyExists
		}
		phone := rq.GetPhone()
		userM.Phone = &phone
	}

	if rq.Nickname != nil {
		userM.Nickname = rq.GetNickname()
	}

	if err := b.store.User().Update(ctx, userM); err != nil {
		return nil, err
	}

	return &v1.UpdateUserResponse{}, nil
}
