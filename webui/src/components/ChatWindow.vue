<script>
import axios from '../services/axios.js'
import GroupInfo from './GroupInfo.vue'

const EMOJIS = ['👍', '❤️', '😂', '😮', '😢', '🙏']

export default {
  components: { GroupInfo },
  props: {
    chatId: { type: String, required: true },
  },
  data() {
    return {
      conversation: null,
      messages: [],
      draft: '',
      imageData: '',     // base64 data URI of an attached image (uploaded, not a URL)
      sending: false,
      timer: null,
      myId: localStorage.getItem('token') || '',
      emojis: EMOJIS,
      reactionPickerFor: '', // messageId whose emoji picker is open
      forwardFor: null,      // message being forwarded
      forwardTargets: [],    // conversations to forward into
      showGroupInfo: false,  // group management modal
      replyTo: null,         // message currently being replied to (null = none)
    }
  },
  computed: {
    // Lookup by messageId so a reply can render the quoted message.
    messageById() {
      const m = {}
      for (const x of this.messages) m[x.messageId] = x
      return m
    },
  },
  mounted() {
    this.refresh()
    // Poll every 2s — auto-refreshes incoming messages, reactions and checkmarks
    // without any manual page reload.
    this.timer = setInterval(this.refresh, 2000)
  },
  beforeUnmount() {
    if (this.timer) clearInterval(this.timer)
  },
  methods: {
    async refresh() {
      try {
        const res = await axios.get(`/conversations/${this.chatId}`)
        const wasAtBottom = this.isAtBottom()
        this.conversation = res.data.conversation
        this.messages = res.data.messages || []
        if (wasAtBottom) this.$nextTick(this.scrollToBottom)
      } catch (e) {
        if (e.response?.status === 404) this.$router.push('/home')
      }
    },
    isAtBottom() {
      const el = this.$refs.scroll
      if (!el) return true
      return el.scrollHeight - el.scrollTop - el.clientHeight < 80
    },
    scrollToBottom() {
      const el = this.$refs.scroll
      if (el) el.scrollTop = el.scrollHeight
    },
    onFileChange(e) {
      const file = e.target.files && e.target.files[0]
      if (!file) return
      if (!file.type.startsWith('image/')) {
        alert('Please choose an image file')
        return
      }
      if (file.size > 1.5 * 1024 * 1024) {
        alert('Image is too large (max 1.5 MB)')
        e.target.value = ''
        return
      }
      const reader = new FileReader()
      reader.onload = () => { this.imageData = reader.result }
      reader.readAsDataURL(file)
    },
    clearImage() {
      this.imageData = ''
      if (this.$refs.file) this.$refs.file.value = ''
    },
    onGroupUpdated() {
      this.refresh()
    },
    onGroupLeft() {
      this.$router.push('/home')
    },
    async send() {
      const text = this.draft.trim()
      // A message may be text-only, image-only, or both.
      if ((!text && !this.imageData) || this.sending) return
      this.sending = true
      try {
        await axios.post(`/conversations/${this.chatId}/messages`, {
          text,
          photo: this.imageData,
          replyToMessageId: this.replyTo ? this.replyTo.messageId : '',
        })
        this.draft = ''
        this.clearImage()
        this.replyTo = null
        await this.refresh()
        this.$nextTick(this.scrollToBottom)
      } catch (e) {
        alert('Could not send message')
      } finally {
        this.sending = false
      }
    },
    // --- Reply -------------------------------------------------------------
    startReply(m) {
      this.replyTo = m
      this.$nextTick(() => this.$refs.input && this.$refs.input.focus())
    },
    cancelReply() {
      this.replyTo = null
    },
    quotePreview(m) {
      if (!m) return ''
      if (m.text) return m.text.length > 80 ? m.text.slice(0, 80) + '…' : m.text
      if (m.photo) return '📷 Photo'
      return '…'
    },
    scrollToMessage(id) {
      const el = document.getElementById('msg-' + id)
      if (!el) return
      el.scrollIntoView({ behavior: 'smooth', block: 'center' })
      el.classList.add('flash')
      setTimeout(() => el.classList.remove('flash'), 1200)
    },
    async deleteMsg(msgId) {
      if (!confirm('Delete this message?')) return
      try {
        await axios.delete(`/conversations/${this.chatId}/messages/${msgId}`)
        await this.refresh()
      } catch (e) {
        alert('Could not delete')
      }
    },
    isMine(senderID) {
      return senderID === this.myId
    },
    // --- Reactions ---------------------------------------------------------
    myReaction(m) {
      return (m.comments || []).find(c => c.userId === this.myId)
    },
    togglePicker(msgId) {
      this.reactionPickerFor = this.reactionPickerFor === msgId ? '' : msgId
    },
    async react(m, emoji) {
      this.reactionPickerFor = ''
      const mine = this.myReaction(m)
      try {
        // Clicking your current emoji again removes it (toggle off).
        if (mine && mine.emoticon === emoji) {
          await axios.delete(`/conversations/${this.chatId}/messages/${m.messageId}/reaction`)
        } else {
          await axios.put(`/conversations/${this.chatId}/messages/${m.messageId}/reaction`, { emoticon: emoji })
        }
        await this.refresh()
      } catch (e) {
        alert('Could not update reaction')
      }
    },
    reactionTitle(m) {
      return (m.comments || []).map(c => `${c.emoticon} ${c.userName}`).join('\n')
    },
    // --- Forwarding --------------------------------------------------------
    async openForward(m) {
      this.forwardFor = m
      try {
        const res = await axios.get('/conversations')
        // Can only forward into an existing chat other than this one.
        this.forwardTargets = (res.data || []).filter(c => c.conversationId !== this.chatId)
      } catch (e) {
        this.forwardTargets = []
      }
    },
    async doForward(targetId) {
      const m = this.forwardFor
      this.forwardFor = null
      try {
        await axios.post(`/conversations/${this.chatId}/messages/${m.messageId}/forward`, {
          targetConversationId: targetId,
        })
      } catch (e) {
        alert('Could not forward message')
      }
    },
    statusIcon(status) {
      if (status === 'sent') return '✓'
      return '✓✓'
    },
    statusColor(status) {
      return status === 'read' ? '#34b7f1' : '#888'
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
  <div class="d-flex flex-column h-100">
    <div class="px-4 py-3 border-bottom bg-white d-flex align-items-center gap-2">
      <img v-if="conversation?.photo" :src="conversation.photo" class="head-avatar" alt="" />
      <strong class="flex-grow-1">
        <span v-if="conversation?.isGroup" class="me-1">👥</span>{{ conversation?.name || 'Chat' }}
      </strong>
      <button
        v-if="conversation?.isGroup"
        class="btn btn-sm btn-outline-secondary"
        @click="showGroupInfo = true"
      >Group info</button>
    </div>

    <div ref="scroll" class="flex-grow-1 overflow-auto p-4" style="background-color: #efeae2;">
      <div v-if="messages.length === 0" class="text-center text-muted small mt-5">
        No messages yet. Say hi!
      </div>
      <div
        v-for="m in messages"
        :key="m.messageId"
        class="d-flex mb-2"
        :class="isMine(m.senderId) ? 'justify-content-end' : 'justify-content-start'"
      >
        <div class="msg-wrap" :class="isMine(m.senderId) ? 'align-items-end' : 'align-items-start'">
          <div :id="'msg-' + m.messageId" :class="['bubble', isMine(m.senderId) ? 'mine' : 'theirs']">
            <div v-if="!isMine(m.senderId)" class="small fw-bold text-primary mb-1">{{ m.sender }}</div>

            <!-- Quoted message (reply) -->
            <div
              v-if="m.replyToId && messageById[m.replyToId]"
              class="quote"
              @click="scrollToMessage(m.replyToId)"
              :title="'Jump to ' + messageById[m.replyToId].sender + '\'s message'"
            >
              <div class="quote-author">{{ messageById[m.replyToId].sender }}</div>
              <div class="quote-text">{{ quotePreview(messageById[m.replyToId]) }}</div>
            </div>
            <div v-else-if="m.replyToId" class="quote quote-missing">
              <em>Original message unavailable</em>
            </div>

            <!-- Forwarded marker -->
            <div v-if="m.forwarded" class="forwarded small mb-1">
              <em>↪ Forwarded</em>
            </div>

            <!-- Uploaded image -->
            <img v-if="m.photo" :src="m.photo" class="msg-img mb-1" alt="image" />

            <div v-if="m.text" class="msg-text">{{ m.text }}</div>

            <div class="small text-end mt-1" style="opacity: 0.6;">
              {{ formatTime(m.timestamp) }}
              <span v-if="isMine(m.senderId)" :style="{ color: statusColor(m.status) }">
                {{ statusIcon(m.status) }}
              </span>
            </div>

            <!-- Reactions on this message -->
            <div v-if="m.comments && m.comments.length" class="reactions" :title="reactionTitle(m)">
              <span v-for="c in m.comments" :key="c.commentId" class="reaction-chip">
                {{ c.emoticon }}<span class="reaction-name">{{ c.userName }}</span>
              </span>
            </div>
          </div>

          <!-- Hover actions -->
          <div class="actions">
            <button class="act" title="React" @click="togglePicker(m.messageId)">🙂</button>
            <button class="act" title="Reply" @click="startReply(m)">↩</button>
            <button class="act" title="Forward" @click="openForward(m)">↪</button>
            <button v-if="isMine(m.senderId)" class="act" title="Delete" @click="deleteMsg(m.messageId)">🗑</button>

            <!-- Emoji picker -->
            <div v-if="reactionPickerFor === m.messageId" class="picker">
              <button
                v-for="e in emojis"
                :key="e"
                class="emoji"
                :class="{ chosen: myReaction(m) && myReaction(m).emoticon === e }"
                @click="react(m, e)"
              >{{ e }}</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <form @submit.prevent="send" class="p-3 border-top bg-white">
      <!-- Replying to … -->
      <div v-if="replyTo" class="reply-bar mb-2">
        <div class="reply-bar-bar"></div>
        <div class="flex-grow-1 overflow-hidden">
          <div class="small fw-bold text-primary">Replying to {{ replyTo.sender }}</div>
          <div class="small text-muted text-truncate">{{ quotePreview(replyTo) }}</div>
        </div>
        <button type="button" class="btn-close btn-sm" @click="cancelReply" aria-label="Cancel reply"></button>
      </div>

      <div v-if="imageData" class="mb-2 d-flex align-items-center gap-2">
        <img :src="imageData" class="preview" alt="preview" />
        <button type="button" class="btn btn-sm btn-outline-secondary" @click="clearImage">Remove image</button>
      </div>
      <div class="d-flex gap-2">
        <input ref="file" type="file" accept="image/*" class="d-none" @change="onFileChange" />
        <button type="button" class="btn btn-outline-secondary" title="Attach image" @click="$refs.file.click()">📎</button>
        <input
          ref="input"
          v-model="draft"
          type="text"
          class="form-control"
          placeholder="Type a message…"
          :disabled="sending"
        />
        <button type="submit" class="btn btn-primary" :disabled="(!draft.trim() && !imageData) || sending">
          Send
        </button>
      </div>
    </form>

    <!-- Forward target picker -->
    <div v-if="forwardFor" class="modal-backdrop-custom" @click.self="forwardFor = null">
      <div class="modal-card">
        <div class="d-flex justify-content-between align-items-center mb-3">
          <h6 class="m-0">Forward to…</h6>
          <button type="button" class="btn-close" @click="forwardFor = null"></button>
        </div>
        <div v-if="forwardTargets.length === 0" class="text-muted small">
          No other chats available. You can only forward to an existing chat.
        </div>
        <div v-else class="list-group">
          <button
            v-for="c in forwardTargets"
            :key="c.conversationId"
            type="button"
            class="list-group-item list-group-item-action"
            @click="doForward(c.conversationId)"
          >
            {{ c.name || 'Unnamed' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Group management -->
    <GroupInfo
      v-if="showGroupInfo && conversation"
      :conversation="conversation"
      @close="showGroupInfo = false"
      @updated="onGroupUpdated"
      @left="onGroupLeft"
    />
  </div>
</template>

<style scoped>
.head-avatar { width: 36px; height: 36px; border-radius: 50%; object-fit: cover; }
.msg-wrap {
  display: flex;
  flex-direction: column;
  max-width: 70%;
}
.bubble {
  padding: 0.5rem 0.75rem;
  border-radius: 8px;
  word-wrap: break-word;
}
.bubble.mine { background-color: #d9fdd3; }
.bubble.theirs { background-color: white; }
.msg-text { white-space: pre-wrap; }
.msg-img {
  max-width: 240px;
  max-height: 240px;
  border-radius: 6px;
  display: block;
}
.forwarded { color: #667781; }
.quote {
  border-left: 3px solid #34b7f1;
  background: rgba(0, 0, 0, 0.04);
  padding: 4px 8px;
  border-radius: 4px;
  margin-bottom: 4px;
  cursor: pointer;
  font-size: 0.85rem;
  max-width: 100%;
  overflow: hidden;
}
.quote-author { color: #34b7f1; font-weight: 600; font-size: 0.78rem; }
.quote-text {
  color: #555;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.quote-missing { color: #999; font-size: 0.8rem; }
.reply-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #f0f0f0;
  border-radius: 6px;
  padding: 6px 10px;
}
.reply-bar-bar {
  width: 3px;
  align-self: stretch;
  background: #34b7f1;
  border-radius: 2px;
}
@keyframes flash-anim {
  0%   { box-shadow: 0 0 0 3px rgba(52, 183, 241, 0.7); }
  100% { box-shadow: 0 0 0 3px rgba(52, 183, 241, 0); }
}
.bubble.flash { animation: flash-anim 1.2s ease-out; }
.preview {
  max-width: 80px;
  max-height: 80px;
  border-radius: 6px;
  border: 1px solid #ddd;
}
.reactions {
  margin-top: 4px;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.reaction-chip {
  background: #f0f0f0;
  border-radius: 10px;
  padding: 0 6px;
  font-size: 0.8rem;
  display: inline-flex;
  align-items: center;
  gap: 2px;
}
.reaction-name { color: #667781; font-size: 0.7rem; }
.actions {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-top: 2px;
  opacity: 0;
  transition: opacity 0.15s;
  position: relative;
}
.msg-wrap:hover .actions { opacity: 1; }
.act {
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 0.9rem;
  padding: 0 2px;
}
.picker {
  position: absolute;
  bottom: 100%;
  background: white;
  border: 1px solid #ddd;
  border-radius: 20px;
  padding: 2px 6px;
  display: flex;
  gap: 2px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
  z-index: 10;
}
.emoji {
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 1.1rem;
  padding: 2px;
  border-radius: 50%;
}
.emoji:hover { background: #f0f0f0; }
.emoji.chosen { background: #d9fdd3; }
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
