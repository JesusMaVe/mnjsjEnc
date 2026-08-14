<template>
  <div class="animate-fade-in bg-white rounded-2xl shadow-card overflow-hidden">
    <!-- Connection Status Banner -->
    <Transition name="slide-down">
      <div v-if="connectionStatus !== 'connected'" class="px-4 py-2 text-center text-sm font-medium"
        :class="{
          'bg-yellow-100 text-yellow-800': connectionStatus === 'connecting' || connectionStatus === 'reconnecting',
          'bg-red-100 text-red-800': connectionStatus === 'disconnected',
          'bg-red-200 text-red-900': connectionStatus === 'failed'
        }">
        <span v-if="connectionStatus === 'connecting'">Conectando...</span>
        <span v-else-if="connectionStatus === 'reconnecting'">Reconectando...</span>
        <span v-else-if="connectionStatus === 'disconnected'">Desconectado. Intentando reconectar...</span>
        <span v-else>No se pudo conectar. Recarga la página.</span>
      </div>
    </Transition>

    <!-- Header -->
    <div class="bg-gradient-to-r from-primary to-primary-hover px-5 py-4">
      <div class="flex justify-between items-center">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 bg-white/20 rounded-xl flex items-center justify-center">
            <PhUsers class="w-5 h-5 text-white" :weight="'duotone'" />
          </div>
          <div>
            <h2 class="font-display font-bold text-white text-lg">Sala: {{ roomId }}</h2>
            <p class="text-white/70 text-sm">{{ username }}</p>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <div class="flex items-center gap-2 bg-white/10 px-3 py-1.5 rounded-full">
            <span class="w-2 h-2 rounded-full" :class="connected ? 'bg-green-400' : 'bg-red-400'"></span>
            <span class="text-white text-sm font-medium">{{ connected ? 'Conectado' : 'Desconectado' }}</span>
          </div>
          <button @click="handleDisconnect" class="text-white/70 hover:text-white hover:bg-white/10 p-2 rounded-lg transition-all duration-200" title="Salir">
            <PhSignOut class="w-5 h-5" :weight="'duotone'" />
          </button>
        </div>
      </div>
    </div>

    <!-- Messages -->
    <div ref="messagesContainer" class="h-[28rem] overflow-y-auto p-5 space-y-4 bg-gray-50/50">
      <template v-for="item in messages" :key="item.id">
        <div v-if="item.type === 'system'" class="flex justify-center animate-fade-in">
          <span class="text-xs text-text-muted bg-surface px-3 py-1 rounded-full">{{ item.content }}</span>
        </div>
        <MessageItem
          v-else
          :message="item"
          :is-sender="item.sender === username"
          @validate="send({ type: 'validate', messageId: $event })"
        />
      </template>

      <div v-if="messages.length === 0" class="flex flex-col items-center justify-center h-full text-center py-12">
        <div class="w-20 h-20 bg-surface rounded-full flex items-center justify-center mb-4">
          <PhChatCircleDots class="w-10 h-10 text-text-muted" :weight="'duotone'" />
        </div>
        <p class="text-text-secondary font-medium">No hay mensajes aún</p>
        <p class="text-text-muted text-sm mt-1">Envía el primer mensaje seguro</p>
      </div>
    </div>

    <!-- Input -->
    <form @submit.prevent="handleSend" class="border-t border-surface bg-white p-4">
      <div class="flex gap-3">
        <input v-model="newMessage" type="text" placeholder="Escribe tu mensaje..."
          :disabled="!connected"
          class="input-field flex-1 disabled:opacity-50 disabled:cursor-not-allowed" />
        <button type="submit" :disabled="!newMessage.trim() || !connected" class="btn-primary flex items-center gap-2">
          Enviar
          <PhPaperPlaneRight class="w-4 h-4" :weight="'duotone'" />
        </button>
      </div>
    </form>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import { PhUsers, PhSignOut, PhChatCircleDots, PhPaperPlaneRight } from '@phosphor-icons/vue'
import MessageItem from './MessageItem.vue'
import { useWebSocket } from '../composables/useWebSocket'

const props = defineProps({ username: String, roomId: String, token: String })
const emit = defineEmits(['disconnect'])

const messages = ref([])
const newMessage = ref('')
const messagesContainer = ref(null)
const { connected, connectionStatus, send, onMessage, disconnect, connect } = useWebSocket()

const BACKEND = 'http://127.0.0.1:8080'

let msgCounter = 0

// Descifrar mensaje via backend proxy
async function decryptMessage(ciphertext) {
  try {
    const res = await fetch(`${BACKEND}/decrypt`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ciphertext })
    })
    if (res.status === 403) return null
    const data = await res.json()
    return data.message || null
  } catch {
    return null
  }
}

// Cargar historial de mensajes desde la BD
async function loadHistory() {
  try {
    const res = await fetch(`${BACKEND}/messages/${props.roomId}`)
    if (!res.ok) return
    const history = await res.json()
    if (!history) return

    const promises = history.map(async (msg) => {
      const decrypted = await decryptMessage(msg.content)
      return {
        id: msg.id, type: 'message', sender: msg.sender,
        content: decrypted, status: msg.status, createdAt: msg.createdAt
      }
    })

    const resolved = await Promise.all(promises)
    messages.value.push(...resolved)
    scrollToBottom()
  } catch (e) {
    console.error('Error loading history:', e)
  }
}

onMessage((data) => {
  if (data.type === 'new_message') {
    decryptMessage(data.content).then(decrypted => {
      messages.value.push({
        id: data.messageId, type: 'message', sender: data.sender,
        content: decrypted, status: 'no_verificado', createdAt: new Date().toISOString()
      })
      scrollToBottom()
    })
  } else if (data.type === 'verified') {
    const msg = messages.value.find(m => m.id === data.messageId)
    if (msg) msg.status = 'verificado'
  } else if (data.type === 'user_joined') {
    messages.value.push({ id: `sys-${++msgCounter}`, type: 'system', content: `${data.username} se unió a la sala` })
    scrollToBottom()
  } else if (data.type === 'user_left') {
    messages.value.push({ id: `sys-${++msgCounter}`, type: 'system', content: `${data.username} salió de la sala` })
    scrollToBottom()
  }
})

function handleSend() {
  if (!newMessage.value.trim() || !connected.value) return
  send({ type: 'message', content: newMessage.value.trim() })
  newMessage.value = ''
}

function handleDisconnect() {
  disconnect()
  emit('disconnect')
}

function scrollToBottom() {
  nextTick(() => {
    if (messagesContainer.value) messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  })
}

onMounted(() => {
  loadHistory()
  connect({ username: props.username, roomId: props.roomId, token: props.token })
})
</script>

<style scoped>
.message-enter-active { transition: all 0.3s ease-out; }
.message-enter-from { opacity: 0; transform: translateY(10px); }
.slide-down-enter-active, .slide-down-leave-active { transition: all 0.3s ease; }
.slide-down-enter-from, .slide-down-leave-to { opacity: 0; transform: translateY(-100%); }
</style>
