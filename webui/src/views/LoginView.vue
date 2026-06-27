<script>
import axios from '../services/axios.js'

export default {
  data() {
    return { name: '', error: '', loading: false }
  },
  methods: {
    async login() {
      if (this.name.length < 3 || this.name.length > 16) {
        this.error = 'Name must be 3-16 characters'
        return
      }
      this.loading = true
      this.error = ''
      try {
        const res = await axios.post('/session', { name: this.name })
        localStorage.setItem('token', res.data.identifier)
        localStorage.setItem('name', res.data.name)
        this.$router.push('/home')
      } catch (e) {
        this.error = 'Login failed: ' + (e.response?.status || e.message)
      } finally {
        this.loading = false
      }
    },
  },
}
</script>

<template>
  <div class="container py-5">
    <div class="row justify-content-center">
      <div class="col-md-5">
        <div class="card shadow-sm">
          <div class="card-body p-4">
            <h2 class="card-title text-center mb-4">WASA Messenger</h2>
            <form @submit.prevent="login">
              <div class="mb-3">
                <label class="form-label">Your name</label>
                <input
                  v-model="name"
                  type="text"
                  class="form-control"
                  placeholder="e.g. alice"
                  required
                  minlength="3"
                  maxlength="16"
                />
                <div class="form-text">3 to 16 characters</div>
              </div>
              <div v-if="error" class="alert alert-danger py-2">{{ error }}</div>
              <button type="submit" class="btn btn-primary w-100" :disabled="loading">
                {{ loading ? 'Signing in…' : 'Sign in' }}
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>