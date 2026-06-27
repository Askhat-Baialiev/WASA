import axios from 'axios'

// __API_URL__ is injected by vite.config.js — points to the backend.
const instance = axios.create({
  baseURL: __API_URL__,
  timeout: 5000,
})

instance.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export default instance
