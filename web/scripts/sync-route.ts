/**
 * 同步前端路由到后端菜单数据库
 * 运行: pnpm sync:route
 *
 * 幂等性: 通过 menuCode 作为唯一标识，已存在的菜单会更新，不存在则创建
 */

import axios from 'axios';
import { generatedRoutes } from '../src/router/elegant/routes';

// 路由节点类型
interface RouteNode {
  name: string;
  path: string;
  component?: string;
  props?: unknown;
  meta?: {
    title?: string;
    i18nKey?: string;
    icon?: string;
    order?: number;
    constant?: boolean;
    hideInMenu?: boolean;
    keepAlive?: boolean;
  };
  children?: RouteNode[];
}

// 菜单数据结构
interface MenuPayload {
  menuCode: string;
  menuName: string;
  menuType: 'menu' | 'page';
  i18nKey?: string;
  icon?: string;
  path?: string;
  component?: string;
  sortOrder: number;
  visible: 0 | 1;
  status: 0;
  parentID?: string;
}

// 扩展菜单数据（用于关联父子关系）
interface MenuWithParent extends MenuPayload {
  _parentCode?: string;
}

// API 配置
const API_BASE_URL = process.env.VITE_SERVICE_BASE_URL || 'http://localhost:5558';
const token = process.env.API_TOKEN || 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODAyODQ2MTQsImlhdCI6MTc4MDI3NzQxNCwibmJmIjoxNzgwMjc3NDE0LCJ0b2tlbl90eXBlIjoiYWNjZXNzIiwieC11c2VyLWlkIjoiYjA3MzgyZDItZWNlNy00MDI2LTliNDYtMmRhMjY4ZGUxMDA0In0.94u5CjjX9Aw8VRFNHmu6hDKy6KbEkVxyukfpXOrheNc';

// 内置常量路由（不同步到数据库）
const CONSTANT_ROUTES = new Set(['403', '404', '500', 'login', 'home', 'iframe-page']);

// 扁平化路由为菜单数组
function flattenRoutes(routes: RouteNode[]): MenuWithParent[] {
  const menus: MenuWithParent[] = [];

  for (const route of routes) {
    if (CONSTANT_ROUTES.has(route.name)) continue;

    const i18nKey = route.meta?.i18nKey || `route.${route.name}`;

    const menu: MenuWithParent = {
      menuCode: route.name,
      menuName: route.meta?.title || route.name,
      menuType: route.children && route.children.length > 0 ? 'menu' : 'page',
      i18nKey,
      icon: route.meta?.icon,
      path: route.path,
      component: route.component,
      sortOrder: route.meta?.order || 0,
      visible: route.meta?.hideInMenu ? 0 : 1,
      status: 0
    };

    menus.push(menu);

    if (route.children && route.children.length > 0) {
      // 递归处理子路由
      for (const child of route.children) {
        if (CONSTANT_ROUTES.has(child.name)) continue;

        const childI18nKey = child.meta?.i18nKey || `route.${child.name}`;

        const childMenu: MenuWithParent = {
          menuCode: child.name,
          menuName: child.meta?.title || child.name,
          menuType: 'page',
          i18nKey: childI18nKey,
          icon: child.meta?.icon,
          path: child.path,
          component: child.component,
          sortOrder: child.meta?.order || 0,
          visible: child.meta?.hideInMenu ? 0 : 1,
          status: 0,
          _parentCode: route.name
        };
        menus.push(childMenu);
      }
    }
  }

  return menus;
}

// 获取已有菜单映射 (menuCode -> menu)
async function fetchExistingMenus(): Promise<Map<string, any>> {
  const response = await axios.get(`${API_BASE_URL}/v1/menus`, {
    params: { page_size: 1000 },
    headers: { Authorization: `Bearer ${token}` },
  });

  const map = new Map<string, any>();
  if (response.data.code === 0) {
    for (const menu of response.data.data.menus || []) {
      map.set(menu.menuCode, menu);
    }
  }
  return map;
}

// 幂等 upsert：存在则更新，不存在则创建
async function upsertMenu(
  menu: MenuPayload,
  existingMenus: Map<string, any>
): Promise<{ created: boolean; menuID: string }> {
  const existing = existingMenus.get(menu.menuCode);

  let data: any;
  let method: 'post' | 'put';
  let url: string;

  if (existing) {
    // 更新已有菜单（保留 parentID）
    data = { ...menu };
    if (existing.parentID) {
      data.parentID = existing.parentID;
    }
    method = 'put';
    url = `${API_BASE_URL}/v1/menus/${existing.menuID}`;
  } else {
    // 创建新菜单
    data = menu;
    method = 'post';
    url = `${API_BASE_URL}/v1/menus`;
  }

  const response = await axios({
    method,
    url,
    data,
    headers: { Authorization: `Bearer ${token}` },
  }).catch((err) => {
    const message = err.response?.data?.message || err.message;
    console.error(`  API 错误 [${menu.menuCode}]:`, message);
    throw new Error(message);
  });

  if (response.data.code !== 0) {
    console.error(`  业务错误 [${menu.menuCode}]:`, response.data.message);
    throw new Error(response.data.message || '操作失败');
  }

  const menuID = existing?.menuID || response.data.data.menuID;
  return { created: !existing, menuID };
}

// 更新子菜单的父级关系
async function updateParentRelation(
  menuCode: string,
  parentMenuCode: string,
  menuIDMap: Map<string, string>,
  existingMenus: Map<string, any>
): Promise<void> {
  const parentMenu = existingMenus.get(parentMenuCode);
  if (!parentMenu) return;

  const menuID = menuIDMap.get(menuCode);
  if (!menuID) return;

  if (parentMenu.menuID !== parentMenu.parentID) {
    await axios.put(
      `${API_BASE_URL}/v1/menus/${menuID}`,
      { parentID: parentMenu.menuID },
      { headers: { Authorization: `Bearer ${token}` } }
    );
  }
}

// 主函数
async function main() {
  console.log('='.repeat(50));
  console.log('开始同步前端路由到后端菜单数据库');
  console.log(`API 地址: ${API_BASE_URL}`);
  console.log('='.repeat(50));

  // 1. 收集已有菜单
  console.log('\n[1/4] 收集已有菜单...');
  const existingMenus = await fetchExistingMenus();
  console.log(`  已存在 ${existingMenus.size} 个菜单`);

  // 2. 扁平化路由
  console.log('\n[2/4] 解析路由结构...');
  const menus = flattenRoutes(generatedRoutes as RouteNode[]);
  console.log(`  生成 ${menus.length} 个菜单项`);

  // 3. 批量 upsert
  console.log('\n[3/4] 同步菜单...');
  const menuIDMap = new Map<string, string>();
  let created = 0, updated = 0;

  for (const menu of menus) {
    try {
      const result = await upsertMenu(menu, existingMenus);
      menuIDMap.set(menu.menuCode, result.menuID);
      if (result.created) {
        created++;
        console.log(`  ✓ 新建: ${menu.menuCode}`);
      } else {
        updated++;
        console.log(`  ↻ 更新: ${menu.menuCode}`);
      }
    } catch (err: any) {
      console.log(`  ✗ 失败: ${menu.menuCode} - ${err.message}`);
    }
  }

  // 4. 更新父子关系
  console.log('\n[4/4] 关联父子菜单...');
  // 重新获取菜单来获取最新的 menuID
  const updatedMenus = await fetchExistingMenus();
  for (const menu of menus) {
    if (menu._parentCode) {
      const parentMenu = updatedMenus.get(menu._parentCode);
      const childMenu = updatedMenus.get(menu.menuCode);
      if (parentMenu && childMenu && parentMenu.menuID !== childMenu.parentID) {
        await axios.put(
          `${API_BASE_URL}/v1/menus/${childMenu.menuID}`,
          { parentID: parentMenu.menuID },
          { headers: { Authorization: `Bearer ${token}` } }
        );
        console.log(`  → 关联: ${menu.menuCode} -> ${menu._parentCode}`);
      }
    }
  }

  // 总结
  console.log('\n' + '='.repeat(50));
  console.log('同步完成');
  console.log(`  新建: ${created}`);
  console.log(`  更新: ${updated}`);
  console.log('='.repeat(50));
}

main().catch((err) => {
  console.error(`\n同步失败: ${err.message}`);
  process.exit(1);
});
