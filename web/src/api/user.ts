import client from './client'
import type {
  Token, User, UploadToken, Response,
  CreateUserRequest, UpdateUserRequest,
  ChangePasswordRequest, ResetPasswordRequest,
  RefreshTokenRequest,
} from './types'

export const login = (username: string, password: string) => {
  const params = new URLSearchParams()
  params.append('username', username)
  params.append('password', password)
  return client.post<Token>('/user/login', params, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
  })
}

export const refreshToken = (refresh_token: string) => {
  return client.post<Token>('/user/refresh', { refresh_token } satisfies RefreshTokenRequest)
}

export const register = (data: CreateUserRequest) => {
  return client.post<User>('/user/register', data)
}

export const getMyInfo = () => {
  return client.get<User>('/user/me')
}

export const updateMyInfo = (data: UpdateUserRequest) => {
  return client.put<User>('/user/me', data)
}

export const changePassword = (data: ChangePasswordRequest) => {
  return client.put<Response>('/user/me/password', data)
}

export const refreshUploadToken = () => {
  return client.post<UploadToken>('/user/me/upload-token')
}

export const resetPassword = (data: ResetPasswordRequest) => {
  return client.post<Response>('/user/reset-password', data)
}
