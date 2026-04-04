import client from './client'
import type { ChartInfo, Song, CreateSongRequest, UpdateSongRequest } from './types'

export const getAllSongLevels = () => {
  return client.get<ChartInfo[]>('/songs')
}

export const getSingleSongInfo = (songId: number, src: string = 'prp') => {
  return client.get<Song>(`/songs/${songId}`, { params: { src } })
}

export const createSong = (data: CreateSongRequest) => {
  return client.post<ChartInfo[]>('/songs', data)
}

export const updateSong = (data: UpdateSongRequest) => {
  return client.put<ChartInfo[]>('/songs', data)
}
