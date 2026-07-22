import axios from 'axios'

export const api = axios.create({ baseURL: '/api/v1', withCredentials: true, timeout: 15000 })

export function setCsrf(token) {
  if (token) api.defaults.headers.common['X-CSRF-Token'] = token
  else delete api.defaults.headers.common['X-CSRF-Token']
}
