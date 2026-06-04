/**
 * 同步前端路由到后端菜单数据库
 * 运行: pnpm sync:route
 *
 * 核心逻辑：以数据库为准，匹配本地菜单 code 进行更新或新建
 * - 已有菜单：更新字段，保留数据库中的 parentID
 * - 新建菜单：基于路由树结构自动设置 parentID
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
  menuID?: string;
}

// 菜单完整信息（从数据库获取）
interface ExistingMenu {
  menuID: string;
  parentID: string;
  menuCode: string;
  [key: string]: unknown;
}

// API 配置
const API_BASE_URL = process.env.VITE_SERVICE_BASE_URL || 'http://localhost:5558';
const token =
  process.env.API_TOKEN ||
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODA1NDYyNjQsImlhdCI6MTc4MDUzOTA2NCwibmJmIjoxNzgwNTM5MDY0LCJ0b2tlbl90eXBlIjoiYWNjZXNzIiwieC11c2VyLWlkIjoiNmIwOTQ1MGQtMjVkOC00NGQ5LWE1Y2EtZDFkZjU3YjBiN2Q1In0.UWHn6spnooN1BtEyOc2n-dxTn62mDXPbxIpNAylISvo';

// 内置常量路由（不同步到数据库）
const CONSTANT_ROUTES = new Set(['403', '404', '500', 'login', 'home', 'iframe-page']);

// 递归收集所有菜单（扁平化）
function collectAllMenus(nodes: RouteNode[], result: RouteNode[] = []): RouteNode[] {
  for (const node of nodes) {
    if (CONSTANT_ROUTES.has(node.name)) continue;
    result.push(node);
    if (node.children && node.children.length > 0) {
      collectAllMenus(node.children, result);
    }
  }
  return result;
}

// 获取已有菜单映射 (menuCode -> menu)，包括嵌套 children
async function fetchExistingMenus(): Promise<Map<string, ExistingMenu>> {
  const response = await axios.get(`${API_BASE_URL}/v1/menus`, {
    params: { page_size: 1000 },
    headers: { Authorization: `Bearer ${token}` }
  });

  const map = new Map<string, ExistingMenu>();

  function extractMenus(menus: any[]): void {
    for (const menu of menus) {
      map.set(menu.menuCode, {
        menuID: menu.menuID,
        parentID: menu.parentID,
        menuCode: menu.menuCode,
        ...menu
      });
      if (menu.children && menu.children.length > 0) {
        extractMenus(menu.children);
      }
    }
  }

  if (response.data.code === 0) {
    extractMenus(response.data.data.menus || []);
  }

  return map;
}

// 创建或更新菜单
async function upsertMenu(
  menuData: MenuPayload,
  existingMenu: ExistingMenu | undefined,
  parentMenuID: string | undefined
): Promise<{ menuID: string; created: boolean }> {
  const isCreate = !existingMenu;

  const url = isCreate
    ? `${API_BASE_URL}/v1/menus`
    : `${API_BASE_URL}/v1/menus/${existingMenu.menuID}`;

  // 新建菜单：使用传入的 parentMenuID
  // 更新菜单：使用传入的 parentMenuID 覆盖数据库中的 parentID，确保树形结构正确
  // 注意：如果 parentMenuID 是空或 "0"，表示顶级菜单，不应该传给后端
  const payload: MenuPayload = {
    ...menuData,
    parentID:
      parentMenuID && parentMenuID !== '0'
        ? parentMenuID
        : undefined
  };

  if (!isCreate && existingMenu) {
    payload.menuID = existingMenu.menuID;
  }

  const response = await axios({
    method: isCreate ? 'post' : 'patch',
    url,
    data: payload,
    headers: { Authorization: `Bearer ${token}` }
  }).catch((err) => {
    const message = err.response?.data?.message || err.message;
    console.error(`  API 错误 [${menuData.menuCode}]:`, message);
    throw new Error(message);
  });

  if (response.data.code !== 0) {
    console.error(`  业务错误 [${menuData.menuCode}]:`, response.data.message);
    throw new Error(response.data.message || '操作失败');
  }

  const menuID = isCreate ? response.data.data.menuID : existingMenu.menuID;
  return { menuID, created: isCreate };
}

// 递归同步树形结构
async function syncRouteTree(
  nodes: RouteNode[],
  parentMenuID: string | undefined,
  existingMenus: Map<string, ExistingMenu>,
  menuIDMap: Map<string, string>,
  stats: { created: number; updated: number }
): Promise<void> {
  for (const node of nodes) {
    if (CONSTANT_ROUTES.has(node.name)) continue;

    const existingMenu = existingMenus.get(node.name);

    const menuData: MenuPayload = {
      menuCode: node.name,
      menuName: node.meta?.title || node.name,
      menuType: node.children && node.children.length > 0 ? 'menu' : 'page',
      i18nKey: node.meta?.i18nKey || `route.${node.name}`,
      icon: node.meta?.icon,
      path: node.path,
      component: node.component,
      sortOrder: node.meta?.order || 0,
      visible: node.meta?.hideInMenu ? 0 : 1,
      status: 0
    };

    try {
      // 传递 parentMenuID，确保父子关系与路由树一致
      const result = await upsertMenu(menuData, existingMenu, parentMenuID);
      menuIDMap.set(node.name, result.menuID);

      if (result.created) {
        stats.created++;
        console.log(`  + 新建: ${node.name}`);
      } else {
        stats.updated++;
        console.log(`  ~ 更新: ${node.name}`);
      }

      // 递归同步子路由，传入当前节点的 menuID 作为子节点的 parentMenuID
      if (node.children && node.children.length > 0) {
        await syncRouteTree(node.children, result.menuID, existingMenus, menuIDMap, stats);
      }
    } catch (err: any) {
      console.log(`  x 失败: ${node.name} - ${err.message}`);
    }
  }
}

// 主函数
async function main() {
  console.log('='.repeat(50));
  console.log('开始同步前端路由到后端菜单数据库');
  console.log(`API 地址: ${API_BASE_URL}`);
  console.log('='.repeat(50));

  // 1. 收集已有菜单
  console.log('\n[1/3] 收集已有菜单...');
  const existingMenus = await fetchExistingMenus();
  console.log(`  已存在 ${existingMenus.size} 个菜单`);

  // 2. 统计路由节点
  console.log('\n[2/3] 解析路由树结构...');
  const allMenus = collectAllMenus(generatedRoutes as RouteNode[]);
  console.log(`  解析 ${allMenus.length} 个路由节点`);

  // 3. 树形同步
  console.log('\n[3/3] 同步菜单（保持树形结构）...');
  const menuIDMap = new Map<string, string>();
  const stats = { created: 0, updated: 0 };

  await syncRouteTree(
    generatedRoutes as RouteNode[],
    undefined,
    existingMenus,
    menuIDMap,
    stats
  );

  // 总结
  console.log('\n' + '='.repeat(50));
  console.log('同步完成');
  console.log(`  新建: ${stats.created}`);
  console.log(`  更新: ${stats.updated}`);
  console.log('='.repeat(50));
}

main().catch((err) => {
  console.error(`\n同步失败: ${err.message}`);
  process.exit(1);
});
