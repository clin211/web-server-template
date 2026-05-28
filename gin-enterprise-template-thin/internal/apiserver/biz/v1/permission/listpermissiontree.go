package permission

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// ListPermissionTree 获取权限树.
func (b *permissionBiz) ListPermissionTree(ctx context.Context, rq *v1.ListPermissionTreeRequest) (*v1.ListPermissionTreeResponse, error) {
	permissions, err := b.store.Permission().ListTree(ctx, buildListPermissionTreeOptions(rq))
	if err != nil {
		return nil, err
	}

	tree := buildPermissionTree(permissions, rq.GetLevel())

	return &v1.ListPermissionTreeResponse{Permissions: tree}, nil
}

// buildListPermissionTreeOptions 构建权限树查询选项.
func buildListPermissionTreeOptions(rq *v1.ListPermissionTreeRequest) *where.Options {
	opts := &where.Options{}

	if rq.GetResourceType() != "" {
		opts.F("resource_type", rq.GetResourceType())
	}

	if rq.GetStatus() != 0 {
		opts.F("status", rq.GetStatus())
	}

	return opts
}

// buildPermissionTree 构建权限树结构.
func buildPermissionTree(permissions []*conversion.PermissionModel, level int32) []*v1.PermissionTreeNode {
	// 这里简化处理，实际可能需要递归构建
	var nodes []*v1.PermissionTreeNode
	for _, p := range permissions {
		node := &v1.PermissionTreeNode{
			Permission: conversion.PermissionModelToPermissionV1(p),
		}
		nodes = append(nodes, node)
	}
	return nodes
}
