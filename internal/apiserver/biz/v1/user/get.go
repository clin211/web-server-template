package user

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Get 实现 UserBiz 接口中的 Get 方法.
func (b *userBiz) Get(ctx context.Context, rq *v1.GetUserRequest) (*v1.GetUserResponse, error) {
	userM, err := b.store.User().Get(ctx, where.F("user_id", rq.GetUserID()))
	if err != nil {
		// 资源不存在应表现为业务错误（404 语义），而非 500 系统错误。
		return nil, errno.FromStore(err)
	}

	return &v1.GetUserResponse{User: conversion.UserModelToUserV1(userM)}, nil
}
