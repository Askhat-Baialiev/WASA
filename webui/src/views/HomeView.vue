<script>
import ConversationList from '../components/ConversationList.vue'
import ChatWindow from '../components/ChatWindow.vue'
import ProfileSettings from '../components/ProfileSettings.vue'

export default {
  components: { ConversationList, ChatWindow, ProfileSettings },
  data() {
    return {
      myName: localStorage.getItem('name') || 'User',
      showProfile: false,
    }
  },
  computed: {
    activeChatId() {
      return this.$route.params.id || ''
    },
  },
  methods: {
    onProfileUpdated(name) {
      this.myName = name
    },
    logout() {
      localStorage.clear()
      this.$router.push('/login')
    },
  },
}
</script>

<template>
  <div class="d-flex flex-column vh-100">
    <nav class="navbar navbar-dark bg-dark px-3 py-2">
      <span class="navbar-brand mb-0 h5">WASA Messenger</span>
      <div class="d-flex align-items-center gap-3">
        <button class="btn btn-sm btn-outline-light" @click="showProfile = true">{{ myName }}</button>
        <button class="btn btn-sm btn-outline-light" @click="logout">Sign out</button>
      </div>
    </nav>

    <div class="d-flex flex-grow-1 overflow-hidden">
      <ConversationList :active-id="activeChatId" />
      <div class="flex-grow-1 d-flex flex-column bg-light">
        <ChatWindow v-if="activeChatId" :key="activeChatId" :chat-id="activeChatId" />
        <div v-else class="d-flex justify-content-center align-items-center h-100 text-muted">
          Select a conversation or start a new one
        </div>
      </div>
    </div>

    <ProfileSettings v-if="showProfile" @close="showProfile = false" @updated="onProfileUpdated" />
  </div>
</template>
