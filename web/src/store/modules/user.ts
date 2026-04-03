import { defineStore } from 'pinia';

import { GetUserInfo, PostLogin, PostLogout, type UserInfoResp } from '@/api/mgr/user';
import { usePermissionStore } from '@/store';
import type { UserInfo } from '@/types/interface';

const InitUserInfo: UserInfo = {
  name: '', // 用户名，用于展示在页面右上角头像处
  roles: [], // 前端权限模型使用 如果使用请配置modules/permission-fe.ts使用
};

export const useUserStore = defineStore('user', {
  state: () => ({
    token: '',
    userInfo: { ...InitUserInfo },
  }),
  getters: {
    roles: (state) => {
      return state.userInfo?.roles;
    },
  },
  actions: {
    async login(userInfo: Record<string, unknown>) {
      const res = await PostLogin(userInfo);
      this.token = res.token;
    },
    async getUserInfo() {
      const res = await GetUserInfo();
      this.userInfo = res as UserInfoResp;
    },
    async logout() {
      if (this.token) {
        try {
          await PostLogout();
        } catch (_err) {
          // 忽略登出接口异常，仍然清理本地态。
        }
      }
      this.token = '';
      this.userInfo = { ...InitUserInfo };
    },
  },
  persist: {
    afterRestore: () => {
      const permissionStore = usePermissionStore();
      permissionStore.initRoutes();
    },
    key: 'user',
    paths: ['token'],
  },
});
