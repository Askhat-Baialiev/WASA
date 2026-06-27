<script>
import axios from '../services/axios.js'

export default {
  emits: ['close', 'updated'],
  data() {
    return {
      name: localStorage.getItem('name') || '',
      photoData: '',
      error: '',
      saving: false,
    }
  },
  methods: {
    onFileChange(e) {
      const file = e.target.files && e.target.files[0]
      if (!file) return
      if (!file.type.startsWith('image/')) { alert('Please choose an image file'); return }
      if (file.size > 1.5 * 1024 * 1024) { alert('Image too large (max 1.5 MB)'); e.target.value = ''; return }
      const reader = new FileReader()
      reader.onload = () => { this.photoData = reader.result }
      reader.readAsDataURL(file)
    },
    async save() {
      this.error = ''
      this.saving = true
      try {
        // Update the name (may fail with 409 if taken).
        if (this.name !== localStorage.getItem('name')) {
          const res = await axios.put('/users/me/username', { name: this.name })
          localStorage.setItem('name', res.data.name)
        }
        // Update the photo if a new one was chosen.
        if (this.photoData) {
          await axios.put('/users/me/photo', { photo: this.photoData })
        }
        this.$emit('updated', localStorage.getItem('name'))
        this.$emit('close')
      } catch (e) {
        this.error = e.response?.data?.error || ('Could not save (' + (e.response?.status || e.message) + ')')
      } finally {
        this.saving = false
      }
    },
  },
}
</script>

<template>
  <div class="modal-backdrop-custom" @click.self="$emit('close')">
    <div class="modal-card">
      <div class="d-flex justify-content-between align-items-center mb-3">
        <h6 class="m-0">Profile settings</h6>
        <button type="button" class="btn-close" @click="$emit('close')"></button>
      </div>

      <form @submit.prevent="save">
        <div class="mb-3">
          <label class="form-label">Display name</label>
          <input v-model="name" type="text" class="form-control" minlength="3" maxlength="16" required />
          <div class="form-text">3 to 16 characters, must be unique.</div>
        </div>

        <div class="mb-3">
          <label class="form-label">Profile picture</label>
          <input type="file" accept="image/*" class="form-control" @change="onFileChange" />
          <img v-if="photoData" :src="photoData" class="preview mt-2" alt="preview" />
        </div>

        <div v-if="error" class="alert alert-danger py-2">{{ error }}</div>

        <button type="submit" class="btn btn-primary w-100" :disabled="saving">
          {{ saving ? 'Saving…' : 'Save' }}
        </button>
      </form>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop-custom {
  position: fixed; inset: 0; background: rgba(0,0,0,0.5);
  display: flex; align-items: center; justify-content: center; z-index: 1050;
}
.modal-card {
  background: white; border-radius: 8px; padding: 1.5rem;
  width: 90%; max-width: 400px; max-height: 80vh; overflow-y: auto;
}
.preview { max-width: 96px; max-height: 96px; border-radius: 50%; border: 1px solid #ddd; object-fit: cover; }
</style>
