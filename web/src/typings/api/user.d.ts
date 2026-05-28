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

    interface RefreshTokenResponse {
      token: string;
      expireAt: number;
      refreshToken: string;
      refreshExpireAt: number;
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
      pageToken: string;
    }

    interface Role {
      id: string;
      code: string;
      name: string;
    }

    interface GetUserRolesResponse {
      roles: Role[];
      permissionCodes: string[];
    }

    interface AssignRolesRequest {
      roleIDs: string[];
    }

    interface MenuTreeNode {
      id: string;
      name: string;
      parentId: string;
      path: string;
      icon?: string;
      order: number;
      menuType: number;
      children?: MenuTreeNode[];
    }

    interface GetMenuTreeResponse {
      menus: MenuTreeNode[];
    }
  }
}