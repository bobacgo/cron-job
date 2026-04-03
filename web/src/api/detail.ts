import { request } from '@/utils/request';

export interface PurchaseListResult {
  list: Array<PurchaseInfo>;
}
export interface PurchaseInfo {
  adminName: string;
  index: string;
  pdName: string;
  pdNum: string;
  pdType: string;
  purchaseNum: number;
  updateTime: Date;
}

export interface ProjectListResult {
  list: Array<ProjectInfo>;
}
export interface ProjectInfo {
  adminName: string;
  adminPhone: string;
  index: number;
  name: string;
  updateTime: Date;
}

const Api = {
  PurchaseList: '/get-purchase-list',
  ProjectList: '/get-project-list',
};

export function getPurchaseList() {
  return request.get<PurchaseListResult>({
    url: Api.PurchaseList,
  });
}

export function getProjectList() {
  return request.get<ProjectListResult>({
    url: Api.ProjectList,
  });
}
