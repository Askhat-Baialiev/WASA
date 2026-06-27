<script>
import axios from '../services/axios.js'

export default {
  props: {
    conversation: { type: Object, required: true },
  },
  emits: ['close', 'updated', 'left'],
  data() {
    return {
      name: this.conversation.name || '',
      photoData: '',
      members: [],
      query: '',
      results: [],
      busy: false,
    }
  },
  computed: {
    convId() { return this.conversation.conversationId },
  },
  mounted() {
    this.loadMembers()
  },
  watch: {
    query(val) {
      if (val.length >= 1) this.search()
      else this.results = []
    },
  },
  methods: {
    async loadMembers() {
      try {
        const res = await axios.get(`/conversations/${this.convId}/members`)
        this.members = res.data || []
      } catch (e) {
        this.members = []
      }
    },
    onFileChange(e) {
      const file = e.target.files && e.target.files[0]
      if (!file) return
      if (!file.type.startsWith('image/')) { alert('Please choose an image file'); return }
      if (file.size > 1.5 * 1024 * 1024) { alert('Image too large (max 1.5 MB)'); e.target.value = ''; return }
      const reader = new FileReader()
      reader.onload = () => { this.photoData = reader.result }
      reader.readAsDataURL(file)
    },
    async saveName() {
      if (!this.name.trim() || this.busy) return
      this.busy = true
      try {
        await axios.put(`/conversations/${this.convId}/name`, { name: this.name.trim() })
        this.$emit('updated')
      } catch (e) {
        alert('Could not rename group')
      } finally {
        this.busy = false
      }
    },
    async savePhoto() {
      if (!this.photoData || this.busy) return
      this.busy = true
      try {
        await axios.put(`/conversations/${this.convId}/photo`, { photo: this.photoData })
        this.photoData = ''
        this.$emit('updated')
      } catch (e) {
        alert('Could not update group picture')
      } finally {
        this.busy = false
      }
    },
    async search() {
      try {
        const res = await axios.get(`/users?search=${encodeURIComponent(this.query)}`)
        const ids = new Set(this.members.map(m => m.identifier))
        this.results = (res.data || []).filter(u => !ids.has(u.identifier))
      } catch (e) {
        this.results = []
      }
    },
    async addMember(u) {
      try {
        await axios.post(`/conversations/${this.convId}/members`, { userId: u.identifier })
        this.query = ''
        this.results = []
        await this.loadMembers()
        this.$emit('updated')
      } catch (e) {
        alert('Could not add member')
      }
    },
    async leave() {
      if (!confirm('Leave this group?')) return
      try {
        await axios.delete(`/conversations/${this.convId}/members/me`)
        this.$emit('left')
        this.$emit('close')
      } catch (e) {
        alert('Could not leave group')
      }
    },
  },
}
</script>

<template>
  <div class="modal-backdrop-custom" @click.self="$emit('close')">
    <div class="modal-card">
      <div class="d-flex justify-content-between align-items-center mb-3">
        <h6 class="m-0">Group info</h6>
        <button type="button" class="btn-close" @click="$emit('close')"></button>
      </div>

      <!-- Name -->
      <label class="form-label">Group name</label>
      <div class="d-flex gap-2 mb-3">
        <input v-model="name" type="text" class="form-control" maxlength="64" />
        <button class="btn btn-outline-primary" :disabled="busy" @click="saveName">Save</button>
      </div>

      <!-- Photo -->
      <label class="form-label">Group picture</label>
      <div class="d-flex align-items-center gap-2 mb-3">
        <img v-if="photoData || conversation.photo" :src="photoData || conversation.photo" class="gphoto" alt="group" />
        <input type="file" accept="image/*" class="form-control" @change="onFileChange" />
        <button class="btn btn-outline-primary" :disabled="!photoData || busy" @click="savePhoto">Save</button>
      </div>

      <!-- Members -->
      <label class="form-label">Members ({{ members.length }})</label>
      <div class="list-group mb-3" style="max-height: 140px; overflow-y: auto;">
        <div v-for="m in members" :key="m.identifier" class="list-group-item py-1">{{ m.name }}</div>
      </div>

      <!-- Add member -->
      <label class="form-label">Add member</label>
      <input v-model="query" type="text" class="form-control mb-2" placeholder="Type a name…" />
      <div v-if="results.length" class="list-group mb-3" style="max-height: 140px; overflow-y: auto;">
        <button
          v-for="u in results"
          :key="u.identifier"
          type="button"
          class="list-group-item list-group-item-action"
          @click="addMember(u)"
        >+ {{ u.name }}</button>
      </div>

      <button type="button" class="btn btn-outline-danger w-100" @click="leave">Leave group</button>
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
  width: 90%; max-width: 420px; max-height: 85vh; overflow-y: auto;
}
.gphoto { width: 48px; height: 48px; border-radius: 50%; object-fit: cover; border: 1px solid #ddd; }
</style>
