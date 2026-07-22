import { defineStore } from 'pinia'
import { api, setCsrf } from '../lib/api'

export const useAuthStore = defineStore('auth', {
  state: () => ({ user: null, ready: false }),
  actions: {
    async restore() {
      try { const { data } = await api.get('/auth/me'); this.user = data.user; setCsrf(data.csrfToken) }
      catch { this.user = null; setCsrf(null) }
      finally { this.ready = true }
    },
    async login(username, password) {
      const { data } = await api.post('/auth/login', { username, password })
      this.user = data.user; setCsrf(data.csrfToken)
    },
    async logout() { await api.post('/auth/logout'); this.user = null; setCsrf(null) }
  }
})
