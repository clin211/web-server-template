package menu_role

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// GetMenuRoles 获取菜单允许的角色列表.
func (b *menuRoleBiz) GetMenuRoles(ctx context.Context, rq *v1.GetMenuRolesRequest) (*v1.GetMenuRolesResponse, error) {
	menuID := rq.GetMenuID()

	if err := b.validateMenuExists(ctx, menuID); err != nil {
		return nil, fmt.Errorf("get menu roles: %w", err)
	}

	roleIDs, roleCodes, err := b.store.MenuRole().GetMenuRoles(ctx, menuID)
	if err != nil {
		return nil, fmt.Errorf("get menu roles: %w", err)
	}

	return &v1.GetMenuRolesResponse{
		RoleIds:   roleIDs,
		RoleCodes: conversion.UniqueStrings(roleCodes),
	}, nil
}