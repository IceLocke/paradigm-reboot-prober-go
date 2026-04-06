import client from './client'
import type { PlayRecordResponse, BatchCreatePlayRecordRequest, PlayRecord, AllChartsResponse } from './types'

export const getRecords = (
  username: string,
  scope: string = 'b50',
  pageSize: number = 50,
  pageIndex: number = 1,
  sortBy: string = 'rating',
  order: string = 'desc'
) => {
  return client.get<PlayRecordResponse>(`/records/${username}`, {
    params: {
      scope,
      page_size: pageSize,
      page_index: pageIndex,
      sort_by: sortBy,
      order,
    },
  })
}

export const uploadRecords = (username: string, data: BatchCreatePlayRecordRequest) => {
  return client.post<PlayRecord[]>(`/records/${username}`, data)
}

export const getAllChartsWithScores = (username: string) => {
  return client.get<AllChartsResponse>(`/records/${username}`, {
    params: { scope: 'all-charts' },
  })
}
