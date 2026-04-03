import { request } from '@/utils/request';
import type { PageResp, PageReq } from '../model';

export interface I18nItem {
  id: number;
  class?: string;
  lang: string;
  key: string;
  value: string;
  created_at?: number;
  updated_at?: number;
  operator?: string;
}

export interface I18nListReq extends PageReq {
  class?: string;
  lang?: string;
  key?: string;
}

export interface I18nCreateReq {
  class?: string;
  lang: string;
  key: string;
  value: string;
  operator?: string;
}

export interface I18nUpdateReq {
  id: number;
  class?: string;
  lang?: string;
  value?: string;
  operator?: string;
}

const Api = {
  I18nList: '/i18n/list',
  I18nOne: '/i18n/one',
  I18nCreate: '/i18n',
  I18nUpdate: '/i18n',
  I18nDelete: '/i18n',
};

export function getI18nList(params: I18nListReq) {
  return request.get<PageResp<I18nItem>>({
    url: Api.I18nList,
    params,
  });
}

export function getI18n(id: number) {
  return request.get<I18nItem>({
    url: Api.I18nOne,
    params: { id },
  });
}

export function addI18n(data: I18nCreateReq) {
  return request.post<I18nItem>({
    url: Api.I18nCreate,
    data,
  });
}

export function updateI18n(data: I18nUpdateReq) {
  return request.put<I18nItem>({
    url: Api.I18nUpdate,
    data,
  });
}

export function deleteI18n(ids: number[]) {
  const params = { ids: ids.join(',') };
  return request.delete({
    url: Api.I18nDelete,
    params,
  });
}
