declare namespace Api {
  /**
   * namespace User
   *
   * backend api module: "user" (apiserver/v1)
   */
  namespace User {
    interface User {
      userID: string;
      username: string;
      nickname: string;
      email: string;
      phone: string;
      postCount: number;
      createdAt: number;
      updatedAt: number;
      status: number;
      gender: number;
      avatar: string;
      description: string;
      lastLoginAt: number;
    }

    interface LoginRequest {
      username: string;
      password: string;
    }

    interface LoginResponse {
      accessToken: string;
      refreshToken: string;
      expireAt: string;
    }

    interface CreateUserRequest {
      username: string;
      password: string;
      nickname?: string;
      email?: string;
      phone?: string;
    }

    interface CreateUserResponse {
      userID: string;
    }

    interface GetUserResponse {
      user: User;
    }

    interface UpdateUserRequest {
      username?: string;
      nickname?: string;
      email?: string;
      phone?: string;
    }

    interface ListUserRequest {
      pageToken?: string;
      pageSize?: number;
    }

    interface ListUserResponse {
      totalCount: number;
      users: User[];
      nextPageToken: string;
    }

    type Role = Api.Role.Role;

    interface GetUserRolesResponse {
      roles: Role[];
      permissionCodes: string[];
    }

    interface AssignRolesRequest {
      roleIDs: string[];
    }

    interface Menu {
      menuID: string;
      parentID: string;
      menuName: string;
      menuCode: string;
      menuType: string;
      icon: string;
      path: string;
      component: string;
      permissionID: string;
      sortOrder: number;
      visible: number;
      status: number;
      createdAt: number;
      updatedAt: number;
    }

    interface MenuTreeNode {
      menu: Menu;
      children: MenuTreeNode[];
    }

    interface GetMenuTreeResponse {
      menus: MenuTreeNode[];
    }
  }
}
