<template>
  <div class="animate-fade-in bg-white rounded-2xl shadow-card p-8 max-w-md mx-auto">
    <div class="text-center mb-8">
      <div class="inline-flex items-center justify-center w-16 h-16 bg-primary-light rounded-2xl mb-4">
        <PhLockKey class="w-8 h-8 text-primary" :weight="'duotone'" />
      </div>
      <h2 class="font-display text-2xl font-bold text-text">Unirse a una sala</h2>
      <p class="text-text-secondary text-sm mt-2">Conecta con otro usuario para enviar mensajes seguros</p>
    </div>

    <form @submit.prevent="handleConnect" class="space-y-5">
      <div>
        <label class="block text-sm font-medium text-text mb-2">Tu nombre</label>
        <input v-model="username" type="text" required placeholder="Ej: Juan" class="input-field" />
      </div>
      <div>
        <label class="block text-sm font-medium text-text mb-2">ID de sala</label>
        <input v-model="roomId" type="text" required placeholder="Ej: sala-123" class="input-field" />
        <p class="text-xs text-text-muted mt-2">Comparte este ID con la otra persona</p>
      </div>
      <p v-if="error" class="text-sm text-red-600">{{ error }}</p>
      <button type="submit" :disabled="!username || !roomId || loading" class="btn-primary w-full">
        {{ loading ? 'Conectando...' : 'Conectar' }}
      </button>
    </form>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { PhLockKey } from '@phosphor-icons/vue'

const emit = defineEmits(['connect'])

const username = ref('')
const roomId = ref('')
const loading = ref(false)
const error = ref('')

const BACKEND = 'http://127.0.0.1:8080'

async function handleConnect() {
  if (!username.value || !roomId.value) return
  loading.value = true
  error.value = ''
  try {
    const resp = await fetch(`${BACKEND}/token?room=${encodeURIComponent(roomId.value)}`)
    if (!resp.ok) throw new Error('Failed to get room token')
    const data = await resp.json()
    emit('connect', username.value, roomId.value, data.token)
  } catch (e) {
    error.value = 'No se pudo conectar al servidor'
  } finally {
    loading.value = false
  }
}
</script>
