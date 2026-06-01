package user

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
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

	// 检查用户名是否已被其他用户占用
	if rq.Username != nil && rq.GetUsername() != userM.Username {
		if existingUser, err := b.store.User().Get(ctx, where.F("username", rq.GetUsername()).L(1)); err == nil && existingUser != nil && existingUser.UserID != userM.UserID {
			slog.WarnContext(ctx, "Username already exists", "username", rq.GetUsername())
			return nil, errno.ErrUserAlreadyExists
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

	if err := b.store.User().Update(ctx, userM); err != nil {
		return nil, err
	}

	return &v1.UpdateUserResponse{}, nil
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
	if err != nil {
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