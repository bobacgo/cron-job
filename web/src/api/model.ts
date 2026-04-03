// 公共模型

export interface IdsReq {
  ids: number[];
}

export interface PageReq {
  page?: number;
  page_size?: number;
}

export interface PageResp<T> {
  list: T[];
  total: number;
}

export interface ArrayResp<T> {
  list: T[];
}
