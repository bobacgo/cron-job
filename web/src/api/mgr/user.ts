import { request } from '@/utils/request';
import { PageResp, PageReq } from '../model';

export interface LoginResp {
    token: string;
}

export interface UserInfoResp {
  name: string;
  roles: string[];
}

export interface User {
    id: number;
    created_at: number;
    updated_at: number;
    account: string;
    phone: string;
    email: string;
    status: number;
    register_at: number;
    register_ip: string;
    login_at: number;
    login_ip: string;
    role_ids: string;
    operator?: string;
}

export interface UserListReq extends PageReq {
  keyword?: string;
  status?: number;
}

export interface UserAddReq {
  account: string;
  password: string;
  email?: string;
  phone?: string;
  status: number;
  role_ids?: string;
  operator?: string;
}

export interface UserUpdateReq {
  id: number;
  email?: string;
  phone?: string;
}

export interface UserStatusUpdateReq {
  id: number;
  status: number; // 1:启用 2:禁用
}

export interface UserRoleUpdateReq {
  id: number;
  role_ids: string; // 逗号分隔
}

export interface UserPasswordUpdateReq {
  id: number;
  old_password?: string;
  new_password: string;
}

// 用户登录相关接口
// 登录
export function PostLogin(req: Record<string, unknown>) { return request.post<LoginResp>({ url: '/user/login', data: req}) }
// 获取用户信息
export function GetUserInfo() { return request.get<UserInfoResp>({ url: '/user-info' }); }
// 退出登录
export function PostLogout() { return request.post({ url: '/logout' }); }

// 用户管理相关接口
// 获取用户列表
export function getUserList(params: UserListReq) { return request.get<PageResp<User>>({ url: '/user/list', params }); }
// 添加用户
export function addUser(data: UserAddReq) { return request.post<User>({ url: '/user', data }); }
// 更新用户
export function updateUser(data: UserUpdateReq) { return request.put<User>({ url: '/user', data }); }
// 更新状态
export function updateUserStatus(data: UserStatusUpdateReq) { return request.put({ url: '/user/status', data }); }
// 更新角色
export function updateUserRole(data: UserRoleUpdateReq) { return request.put({ url: '/user/role', data }); }
// 更新密码
export function updateUserPassword(data: UserPasswordUpdateReq) { return request.put({ url: '/user/password', data }); }
// 删除用户
export function deleteUser(ids: number[]) { return request.delete({ url: '/user', params: { ids: ids.join(',') } }); }
