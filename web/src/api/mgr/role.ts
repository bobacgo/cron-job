import { request } from '@/utils/request';
import type { PageResp, PageReq } from '../model';

export interface Role {
  id: number;
  role_name: string;
  description?: string;
  user_count?: number;
  status?: number;
  created_at?: number;
  updated_at?: number;
  operator?: string;
}

export interface RoleListReq extends PageReq {
  role_name?: string;
  status?: string; // comma separated
}

export interface RoleCreateReq {
  role_name: string;
  description?: string;
  status?: number;
  operator?: string;
}

export interface RoleUpdateReq {
  id: number;
  role_name?: string;
  description?: string;
  status?: number;
  operator?: string;
}

export interface RolePermissionsResp { 
  menu_ids: number[];
}


// 角色管理相关接口
// 获取角色列表
export function getRoleList(params: RoleListReq) { return request.get<PageResp<Role>>({ url: '/role/list', params }); }
// 获取单个角色
export function getRole(id: number) { return request.get<Role>({ url: '/role/one', params: { id } }); }
// 创建角色
export function addRole(data: RoleCreateReq) { return request.post<Role>({ url: '/role', data }); }
// 更新角色
export function updateRole(data: RoleUpdateReq) { return request.put<Role>({ url: '/role', data }); }
// 删除角色
export function deleteRole(ids: number[]) { return request.delete({ url: '/role', params: { ids: ids.join(',') } }); }
// 角色权限相关
export function saveRolePermissions(roleId: number, menuIds: number[]) { return request.post({ url: '/role/permissions', data: { role_id: roleId, menu_ids: menuIds } }); }

// 获取角色权限
export function getRolePermissions(roleId: number) { return request.get<RolePermissionsResp>({ url: '/role/permissions', params: { role_id: roleId } }); }
