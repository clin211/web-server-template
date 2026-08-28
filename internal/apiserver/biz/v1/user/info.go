package user

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// Info 返回当前访问令牌对应的脱敏用户信息。
func (b *userBiz) Info(ctx context.Context, _ *v1.InfoRequest) (*v1.InfoResponse, error) {
	userID := contextx.UserID(ctx)
	if userID == "" {
		return nil, errno.ErrUnauthenticated
	}
	userM, err := b.store.User().Get(ctx, where.F("user_id", userID))
	if err != nil {
		// 认证后用户被删除等场景：按未认证处理，而非 500 系统错误。
		return nil, errno.FromStore(err)
	}
	return &v1.InfoResponse{User: conversion.UserModelToUserV1(userM)}, nil
}
