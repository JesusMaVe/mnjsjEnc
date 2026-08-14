<template>
  <div class="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100">
    <div class="max-w-lg mx-auto p-4 sm:p-6 lg:p-8">
      <header class="mb-8 text-center">
      </header>

      <ConnectionForm v-if="!connected" @connect="handleConnect" />
      <ChatRoom v-else :username="username" :room-id="roomId" :token="token" @disconnect="handleDisconnect" />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import ConnectionForm from './components/ConnectionForm.vue'
import ChatRoom from './components/ChatRoom.vue'

const connected = ref(false)
const username = ref('')
const roomId = ref('')
const token = ref('')

function handleConnect(name, room, roomToken) {
  username.value = name
  roomId.value = room
  token.value = roomToken
  connected.value = true
}

function handleDisconnect() {
  connected.value = false
  username.value = ''
  roomId.value = ''
  token.value = ''
}
</script>
