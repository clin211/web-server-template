package user

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Get 实现 UserBiz 接口中的 Get 方法.
func (b *userBiz) Get(ctx context.Context, rq *v1.GetUserRequest) (*v1.GetUserResponse, error) {
	userM, err := b.store.User().Get(ctx, where.T(ctx))
	if err != nil {
		return nil, err
	}

	return &v1.GetUserResponse{User: conversion.UserModelToUserV1(userM)}, nil
}
