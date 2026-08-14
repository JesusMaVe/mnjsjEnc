<template>
  <div class="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100">
    <div class="max-w-lg mx-auto p-4 sm:p-6 lg:p-8">
      <header class="mb-8 text-center">
      </header>

      <ConnectionForm v-if="!connected" @connect="handleConnect" />
      <ChatRoom v-else :username="username" :room-id="roomId" @disconnect="handleDisconnect" />
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

function handleConnect(name, room) {
  username.value = name
  roomId.value = room
  connected.value = true
}

function handleDisconnect() {
  connected.value = false
  username.value = ''
  roomId.value = ''
}
</script>
