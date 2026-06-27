<script>
import axios from '../services/axios.js'

export default {
  emits: ['close', 'created'],
  data() {
    return {
      name: '',
      query: '',
      results: [],
      selected: [],   // chosen user objects
      creating: false,
    }
  },
  watch: {
    query(val) {
      if (val.length >= 1) this.search()
      else this.results = []
    },
  },
  methods: {
    async search() {
      try {
        const res = await axios.get(`/users?search=${encodeURIComponent(this.query)}`)
        const myId = localStorage.getItem('token')
        this.results = (res.data || []).filter(u => u.identifier !== myId)
      } catch (e) {
        this.results = []
      }
    },
    isSelected(u) {
      return this.selected.some(s => s.identifier === u.identifier)
    },
    toggle(u) {
      if (this.isSelected(u)) {
        this.selected = this.selected.filter(s => s.identifier !== u.identifier)
      } else {
        this.selected.push(u)
      }
    },
    async create() {
      if (!this.name.trim() || this.selected.length === 0 || this.creating) return
      this.creating = true
      try {
        const res = await axios.post('/groups', {
          name: this.name.trim(),
          members: this.selected.map(u => u.identifier),
        })
        this.$emit('created', res.data.conversationId)
        this.$emit('close')
      } catch (e) {
        alert('Could not create group')
      } finally {
        this.creating = false
      }
    },
  },
}
</script>

<template>
  <div class="modal-backdrop-custom" @click.self="$emit('close')">
    <div class="modal-card">
      <div class="d-flex justify-content-between align-items-center mb-3">
        <h6 class="m-0">New group</h6>
        <button type="button" class="btn-close" @click="$emit('close')"></button>
      </div>

      <input v-model="name" type="text" class="form-control mb-2" placeholder="Group name" maxlength="64" />

      <div v-if="selected.length" class="mb-2 d-flex flex-wrap gap-1">
        <span v-for="u in selected" :key="u.identifier" class="badge bg-primary">
          {{ u.name }}
          <span role="button" @click="toggle(u)">×</span>
        </span>
      </div>

      <input v-model="query" type="text" class="form-control mb-2" placeholder="Add members — type a name…" />
      <div class="list-group mb-3" style="max-height: 200px; overflow-y: auto;">
        <button
          v-for="u in results"
          :key="u.identifier"
          type="button"
          class="list-group-item list-group-item-action d-flex justify-content-between align-items-center"
          :class="{ active: isSelected(u) }"
          @click="toggle(u)"
        >
          {{ u.name }}
          <span v-if="isSelected(u)">✓</span>
        </button>
        <div v-if="query && results.length === 0" class="text-muted small p-2">No users found</div>
      </div>

      <button type="button" class="btn btn-primary w-100" :disabled="!name.trim() || selected.length === 0 || creating" @click="create">
        {{ creating ? 'Creating…' : `Create group (${selected.length})` }}
      </button>
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
</style>
