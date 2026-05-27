package permission

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/pagination"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// List 获取权限列表.
func (b *permissionBiz) List(ctx context.Context, rq *v1.ListPermissionRequest) (*v1.ListPermissionResponse, error) {
	pageSize := pagination.NormalizePageSize(rq.GetPageSize())
	total, perms, err := b.store.Permission().List(ctx, buildListPermissionOptions(rq))
	if err != nil {
		return nil, err
	}

	nextPageToken := pagination.NextPageToken(len(perms), pageSize, func() int64 {
		return perms[len(perms)-1].ID
	})

	return &v1.ListPermissionResponse{
		TotalCount:  total,
		Permissions: conversion.PermissionModelListToPermissionV1List(perms),
		PageToken:   nextPageToken,
	}, nil
}

// buildListPermissionOptions 构建权限列表查询选项.
func buildListPermissionOptions(rq *v1.ListPermissionRequest) *where.Options {
	pageSize := pagination.NormalizePageSize(rq.GetPageSize())

	opts := where.NewWhere(where.WithLimit(int64(pageSize)))

	// 解析 page_token 获取游标
	pageToken := rq.GetPageToken()
	if pageToken != "" {
		decodedCursor, err := pagination.DecodeCursor(pageToken)
		if err == nil {
			if id, ok := decodedCursor.GetInt64("id"); ok {
				opts.Cursor = &id
			}
		}
	}

	if rq.GetResourceType() != "" {
		opts.F("resource_type", rq.GetResourceType())
	}

	if rq.GetStatus() != 0 {
		opts.F("status", rq.GetStatus())
	}

	if rq.GetParentID() != "" {
		opts.F("parent_id", rq.GetParentID())
	}

	return opts
}
