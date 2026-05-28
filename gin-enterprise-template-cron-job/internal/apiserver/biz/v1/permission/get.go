package permission

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// Get 获取权限.
func (b *permissionBiz) Get(ctx context.Context, rq *v1.GetPermissionRequest) (*v1.GetPermissionResponse, error) {
	permM, err := b.store.Permission().Get(ctx, where.F("permission_id", rq.GetPermissionID()).L(1))
	if err != nil {
		return nil, errno.ErrPermissionNotFound
	}

	return &v1.GetPermissionResponse{Permission: conversion.PermissionModelToPermissionV1(permM)}, nil
}
