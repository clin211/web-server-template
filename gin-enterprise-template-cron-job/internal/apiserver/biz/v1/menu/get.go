package menu

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// Get 获取菜单.
func (b *menuBiz) Get(ctx context.Context, rq *v1.GetMenuRequest) (*v1.GetMenuResponse, error) {
	menuM, err := b.store.Menu().Get(ctx, where.F("menu_id", rq.GetMenuID()).L(1))
	if err != nil {
		return nil, errno.ErrMenuNotFound
	}

	return &v1.GetMenuResponse{Menu: conversion.MenuModelToMenuV1(menuM)}, nil
}
