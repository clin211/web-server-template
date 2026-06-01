package menu

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// GetMenuRoles 获取菜单允许的角色列表.
func (b *menuBiz) GetMenuRoles(ctx context.Context, rq *v1.GetMenuRolesRequest) (*v1.GetMenuRolesResponse, error) {
	// 先验证菜单是否存在
	_, err := b.store.Menu().Get(ctx, where.F("menu_id", rq.GetMenuID()).L(1))
	if err != nil {
		return nil, fmt.Errorf("get menu for get roles: %w", err)
	}

	// 一次性获取角色ID列表和角色代码列表
	roleIDs, roleCodes, err := b.store.MenuRole().GetMenuRoles(ctx, rq.GetMenuID())
	if err != nil {
		return nil, fmt.Errorf("get menu roles: %w", err)
	}

	return &v1.GetMenuRolesResponse{
		RoleIds:   roleIDs,
		RoleCodes: conversion.UniqueStrings(roleCodes),
	}, nil
}