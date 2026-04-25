import client from './client'
import type { PlayRecordResponse, BatchCreatePlayRecordRequest, PlayRecord, AllChartsResponse, Difficulty } from './types'

export interface RecordFilterParams {
  minLevel?: number | null
  maxLevel?: number | null
  difficulties?: Difficulty[]
}

export const getRecords = (
  username: string,
  scope: string = 'b50',
  pageSize: number = 50,
  pageIndex: number = 1,
  sortBy: string = 'rating',
  order: string = 'desc',
  filter?: RecordFilterParams,
  underflow?: number,
) => {
  const params: Record<string, unknown> = {
    scope,
    page_size: pageSize,
    page_index: pageIndex,
    sort_by: sortBy,
    order,
  }
  if (underflow != null) params.underflow = underflow
  if (filter?.minLevel != null) params.min_level = filter.minLevel
  if (filter?.maxLevel != null) params.max_level = filter.maxLevel
  if (filter?.difficulties && filter.difficulties.length > 0) {
    params.difficulty = filter.difficulties
  }
  return client.get<PlayRecordResponse>(`/records/${username}`, {
    params,
    paramsSerializer: {
      indexes: null,
    },
  })
}

export const uploadRecords = (username: string, data: BatchCreatePlayRecordRequest) => {
  return client.post<PlayRecord[]>(`/records/${username}`, data)
}

export const getAllChartsWithScores = (username: string) => {
  return client.get<AllChartsResponse>(`/records/${username}`, { params: { scope: 'all-charts' } })
}

export const getSongRecords = (username: string, songAddr: string, scope: string = 'best') => {
  return client.get<PlayRecordResponse>(`/records/${username}/song/${songAddr}`, {
    params: { scope },
  })
}
