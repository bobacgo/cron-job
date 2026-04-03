import { request } from '@/utils/request';
import type { PageResp } from '../model';

export interface MenuMeta {
  title?: Record<string, string>;
  [key: string]: any;
}

export interface MenuItem {
  id: number;
  parent_id?: number;
  menu_type?: number;
  path: string;
  name: string;
  component?: string;
  redirect?: string;
  meta?: MenuMeta;
  icon?: string;
  sort?: number;
  children?: MenuItem[];
  created_at?: number;
  updated_at?: number;
  operator?: string;
}

export interface MenuListReq {
  page?: number;
  page_size?: number;
  name?: string;
}

export interface MenuCreateReq {
  parent_id?: number;
  name: string;
  path: string;
  component?: string;
  icon?: string;
  sort?: number;
  meta?: string;
  operator?: string;
}

export interface MenuUpdateReq {
  id: number;
  parent_id?: number;
  name?: string;
  path?: string;
  component?: string;
  icon?: string;
  sort?: number;
  meta?: string;
  operator?: string;
}

const Api = {
  MenuList: '/menu/list',
  MenuTree: '/menu/tree',
  MenuCreate: '/menu',
  MenuUpdate: '/menu',
  MenuDelete: '/menu',
};

class MenuApi {
  list(params?: MenuListReq) {
    return request.get<PageResp<MenuItem>>({
      url: Api.MenuList,
      params,
    });
  }

  tree() {
    return request.get<MenuItem[]>({
      url: Api.MenuTree,
    });
  }

  create(data: MenuCreateReq) {
    return request.post<MenuItem>({
      url: Api.MenuCreate,
      data,
    });
  }

  update(data: MenuUpdateReq) {
    return request.put<MenuItem>({
      url: Api.MenuUpdate,
      data,
    });
  }

  delete(ids: number[]) {
    const params = { ids: ids.join(',') };
    return request.delete({
      url: Api.MenuDelete,
      params,
    });
  }
}

export const menuApi = new MenuApi();

// 简化导出函数
export const getMenuTree = () => menuApi.tree();
export const getMenuList = (params?: MenuListReq) => menuApi.list(params);
export const addMenu = (data: MenuCreateReq) => menuApi.create(data);
export const updateMenu = (data: MenuUpdateReq) => menuApi.update(data);
export const deleteMenu = (ids: number[]) => menuApi.delete(ids);

