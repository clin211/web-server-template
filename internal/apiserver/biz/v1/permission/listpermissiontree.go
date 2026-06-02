package permission

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// ListPermissionTree 获取权限树.
// 支持按资源类型、状态过滤，可指定层级深度.
func (b *permissionBiz) ListPermissionTree(ctx context.Context, rq *v1.ListPermissionTreeRequest) (*v1.ListPermissionTreeResponse, error) {
	// 查询所有权限数据
	permissions, err := b.store.Permission().ListTree(ctx, buildListPermissionTreeOptions(rq))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list permission tree", "error", err)
		return nil, fmt.Errorf("list permission tree: %w", err)
	}

	// 构建树形结构
	tree := buildPermissionTree(permissions)

	// 按层级过滤
	if rq.GetLevel() > 0 {
		tree = filterPermissionTreeByLevel(tree, int(rq.GetLevel()))
	}

	return &v1.ListPermissionTreeResponse{Permissions: tree}, nil
}

// buildListPermissionTreeOptions 构建权限树查询选项.
func buildListPermissionTreeOptions(rq *v1.ListPermissionTreeRequest) *where.Options {
	opts := where.NewWhere()

	if rq.GetResourceType() != "" {
		opts.F("resource_type", rq.GetResourceType())
	}

	if rq.GetStatus() != 0 {
		opts.F("status", rq.GetStatus())
	}

	return opts
}

// buildPermissionTree 将扁平权限列表构建为树形结构.
// 采用 O(n) 复杂度的一次遍历算法，避免 N+1 查询问题.
func buildPermissionTree(permissions []*conversion.PermissionModel) []*v1.PermissionTreeNode {
	if len(permissions) == 0 {
		return nil
	}

	treeMap := make(map[string]*v1.PermissionTreeNode, len(permissions))
	roots := make([]*v1.PermissionTreeNode, 0, len(permissions))

	// 第一遍：创建所有节点
	for _, perm := range permissions {
		node := &v1.PermissionTreeNode{
			Permission: conversion.PermissionModelToPermissionV1(perm),
			Children:   make([]*v1.PermissionTreeNode, 0),
		}
		treeMap[perm.PermissionID] = node
	}

	// 第二遍：建立父子关系
	for _, perm := range permissions {
		node := treeMap[perm.PermissionID]
		if perm.ParentID == nil || *perm.ParentID == "" {
			// 根节点
			roots = append(roots, node)
		} else if parent, ok := treeMap[*perm.ParentID]; ok {
			// 添加到父节点
			parent.Children = append(parent.Children, node)
		} else {
			// 孤儿节点（父节点不存在），作为根节点处理
			roots = append(roots, node)
		}
	}

	return roots
}

// filterPermissionTreeByLevel 按层级深度过滤权限树.
// level=1 返回根节点，level=2 返回根节点及子节点，以此类推.
// level=0 或负数返回完整树.
func filterPermissionTreeByLevel(nodes []*v1.PermissionTreeNode, level int) []*v1.PermissionTreeNode {
	if level <= 0 {
		return nodes
	}

	if level == 1 {
		// 只返回根节点，清空子节点
		for _, node := range nodes {
			node.Children = nil
		}
		return nodes
	}

	// 递归处理子节点
	var filter func(nodes []*v1.PermissionTreeNode, currentLevel int)
	filter = func(nodes []*v1.PermissionTreeNode, currentLevel int) {
		if currentLevel >= level {
			// 到达指定层级，清空子节点
			for _, node := range nodes {
				node.Children = nil
			}
			return
		}
		// 继续向下遍历
		for _, node := range nodes {
			filter(node.Children, currentLevel+1)
		}
	}

	filter(nodes, 1)
	return nodes
}
