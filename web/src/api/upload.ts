import client, { API_BASE } from './client'
import type { UploadFileResponse } from './types'

export const uploadCsvUrl = API_BASE + '/upload/csv'

export const uploadCsv = (formData: FormData) => {
  return client.post<UploadFileResponse>('/upload/csv', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

export const uploadImg = (formData: FormData) => {
  return client.post<UploadFileResponse>('/upload/img', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}
