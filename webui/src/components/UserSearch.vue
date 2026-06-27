<script>
import axios from '../services/axios.js'

export default {
  emits: ['close', 'pick'],
  data() {
    return {
      query: '',
      results: [],
      loading: false,
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
      this.loading = true
      try {
        const res = await axios.get(`/users?search=${encodeURIComponent(this.query)}`)
        const myId = localStorage.getItem('token')
        // Hide myself from search results.
        this.results = (res.data || []).filter(u => u.identifier !== myId)
      } catch (e) {
        this.results = []
      } finally {
        this.loading = false
      }
    },
  },
}
</script>

<template>
  <div class="modal-backdrop-custom" @click.self="$emit('close')">
    <div class="modal-card">
      <div class="d-flex justify-content-between align-items-center mb-3">
        <h6 class="m-0">Find a user</h6>
        <button type="button" class="btn-close" @click="$emit('close')"></button>
      </div>
      <input
        v-model="query"
        type="text"
        class="form-control mb-3"
        placeholder="Type a name…"
        autofocus
      />
      <div v-if="loading" class="text-muted small">Searching…</div>
      <div v-else-if="results.length === 0 && query" class="text-muted small">No users found</div>
      <div v-else class="list-group">
        <button
          v-for="u in results"
          :key="u.identifier"
          type="button"
          class="list-group-item list-group-item-action"
          @click="$emit('pick', u)"
        >
          {{ u.name }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop-custom {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1050;
}
.modal-card {
  background: white;
  border-radius: 8px;
  padding: 1.5rem;
  width: 90%;
  max-width: 400px;
  max-height: 70vh;
  overflow-y: auto;
}
</style>
