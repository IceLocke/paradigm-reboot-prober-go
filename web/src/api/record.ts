import client from './client'
import i18n from '@/i18n'
import type { PlayRecordResponse, PlayRecordInfo, BatchCreatePlayRecordRequest, PlayRecord, AllChartsResponse, Difficulty } from './types'

export interface RecordFilterParams {
  minLevel?: number | null
  maxLevel?: number | null
  difficulties?: Difficulty[]
  b15?: boolean | null
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
  if (filter?.b15 != null) {
    params.b15 = filter.b15
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

/** Fetch the server-ranked global top 50, respecting a smaller server page-size cap. */
export const getGlobalBest50Records = async (username: string, sortBy = 'rating'): Promise<PlayRecordResponse> => {
  const records: PlayRecordInfo[] = []
  let pageSize = 50
  let pageIndex = 1
  let data: PlayRecordResponse

  do {
    data = (await getRecords(username, 'best', pageSize, pageIndex, sortBy)).data
    records.push(...data.records)
    if (data.records.length === 0 && records.length < Math.min(50, data.total)) {
      throw new Error(i18n.global.t('message.incomplete_best_records'))
    }
    if (pageIndex === 1 && data.records.length > 0) pageSize = data.records.length
    pageIndex++
  } while (records.length < Math.min(50, data.total))

  return { ...data, records: records.slice(0, 50) }
}

export const getSongRecords = (username: string, songAddr: string, scope: string = 'best') => {
  return client.get<PlayRecordResponse>(`/records/${username}/song/${songAddr}`, {
    params: { scope },
  })
}
