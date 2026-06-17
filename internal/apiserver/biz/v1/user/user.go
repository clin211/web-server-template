package user

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// UserBiz 定义处理用户请求所需的方法.
type UserBiz interface {
	Create(ctx context.Context, rq *v1.CreateUserRequest) (*v1.CreateUserResponse, error)
	Update(ctx context.Context, rq *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error)
	Delete(ctx context.Context, rq *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error)
	Get(ctx context.Context, rq *v1.GetUserRequest) (*v1.GetUserResponse, error)
	List(ctx context.Context, rq *v1.ListUserRequest) (*v1.ListUserResponse, error)

	UserExpansion
}

// UserExpansion 定义用户操作的扩展方法.
type UserExpansion interface {
	Login(ctx context.Context, rq *v1.LoginRequest) (*v1.LoginResponse, error)
	RefreshToken(ctx context.Context, rq *v1.RefreshTokenRequest) (*v1.LoginResponse, error)
	ChangePassword(ctx context.Context, rq *v1.ChangePasswordRequest) (*v1.ChangePasswordResponse, error)
}

// userBiz 是 UserBiz 接口的实现.
type userBiz struct {
	store store.IStore
}

// 确保 userBiz 实现了 UserBiz 接口.
var _ UserBiz = (*userBiz)(nil)

func New(store store.IStore, _ any) *userBiz {
	return &userBiz{store: store}
}
