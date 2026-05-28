package role

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// Get 获取角色.
func (b *roleBiz) Get(ctx context.Context, rq *v1.GetRoleRequest) (*v1.GetRoleResponse, error) {
	roleM, err := b.store.Role().Get(ctx, where.F("role_id", rq.GetRoleID()).L(1))
	if err != nil {
		return nil, errno.ErrRoleNotFound
	}

	return &v1.GetRoleResponse{Role: conversion.RoleModelToRoleV1(roleM)}, nil
}
