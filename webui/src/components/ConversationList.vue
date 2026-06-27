<script>
import axios from '../services/axios.js'
import UserSearch from './UserSearch.vue'
import GroupCreate from './GroupCreate.vue'

export default {
  components: { UserSearch, GroupCreate },
  props: {
    activeId: { type: String, default: '' },
  },
  data() {
    return {
      conversations: [],
      showSearch: false,
      showGroup: false,
      timer: null,
    }
  },
  mounted() {
    this.refresh()
    // Poll every 3s — cheap way to see new chats / preview changes / group
    // renames & pictures without WebSockets (auto-refresh).
    this.timer = setInterval(this.refresh, 3000)
  },
  beforeUnmount() {
    if (this.timer) clearInterval(this.timer)
  },
  methods: {
    async refresh() {
      try {
        const res = await axios.get('/conversations')
        this.conversations = res.data || []
      } catch (e) {
        if (e.response?.status === 401) {
          localStorage.clear()
          this.$router.push('/login')
        }
      }
    },
    openChat(id) {
      this.$router.push(`/chat/${id}`)
    },
    async onUserPicked(user) {
      this.showSearch = false
      try {
        const res = await axios.post('/conversations', { userId: user.identifier })
        await this.refresh()
        this.openChat(res.data.conversationId)
      } catch (e) {
        alert('Could not start a chat')
      }
    },
    async onGroupCreated(conversationId) {
      this.showGroup = false
      await this.refresh()
      this.openChat(conversationId)
    },
    initials(name) {
      return (name || '?').trim().charAt(0).toUpperCase()
    },
    formatTime(iso) {
      if (!iso) return ''
      const d = new Date(iso)
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    },
  },
}
</script>

<template>
  <aside class="d-flex flex-column bg-white border-end" style="width: 320px;">
    <div class="p-3 border-bottom d-flex justify-content-between align-items-center">
      <h6 class="m-0 fw-bold">Conversations</h6>
      <div class="d-flex gap-1">
        <button class="btn btn-sm btn-primary" @click="showSearch = true">+ Chat</button>
        <button class="btn btn-sm btn-outline-primary" @click="showGroup = true">+ Group</button>
      </div>
    </div>

    <div class="flex-grow-1 overflow-auto">
      <div v-if="conversations.length === 0" class="text-center text-muted small p-4">
        No conversations yet.<br />Start a chat or a group.
      </div>
      <div
        v-for="c in conversations"
        :key="c.conversationId"
        :class="['p-3 border-bottom chat-item d-flex gap-2', { active: c.conversationId === activeId }]"
        @click="openChat(c.conversationId)"
      >
        <img v-if="c.photo" :src="c.photo" class="avatar" alt="" />
        <div v-else class="avatar avatar-fallback">{{ initials(c.name) }}</div>
        <div class="flex-grow-1 overflow-hidden">
          <div class="d-flex justify-content-between align-items-start">
            <strong class="text-truncate">
              <span v-if="c.isGroup" class="me-1">👥</span>{{ c.name || 'Unnamed' }}
            </strong>
            <small class="text-muted ms-2 flex-shrink-0">{{ formatTime(c.lastMessageTime) }}</small>
          </div>
          <div class="small text-muted text-truncate">{{ c.lastMessage || 'No messages yet' }}</div>
        </div>
      </div>
    </div>

    <UserSearch v-if="showSearch" @close="showSearch = false" @pick="onUserPicked" />
    <GroupCreate v-if="showGroup" @close="showGroup = false" @created="onGroupCreated" />
  </aside>
</template>

<style scoped>
.chat-item { cursor: pointer; }
.chat-item:hover { background-color: #f0f0f0; }
.chat-item.active { background-color: #e7f1ff; }
.avatar {
  width: 40px; height: 40px; border-radius: 50%;
  object-fit: cover; flex-shrink: 0;
}
.avatar-fallback {
  display: flex; align-items: center; justify-content: center;
  background: #6c757d; color: white; font-weight: bold;
}
</style>
