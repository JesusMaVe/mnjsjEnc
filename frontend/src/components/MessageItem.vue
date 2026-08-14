<template>
  <div :class="isSender ? 'flex justify-end' : 'flex justify-start'" class="animate-slide-up">
    <div :class="[isSender ? 'bg-primary text-white rounded-2xl rounded-br-md' : 'bg-white text-text rounded-2xl rounded-bl-md shadow-message']" class="max-w-sm px-4 py-3">
      <div v-if="!isSender" class="flex items-center gap-2 mb-1.5">
        <div class="w-6 h-6 bg-surface rounded-full flex items-center justify-center">
          <span class="text-xs font-semibold text-text-secondary">{{ message.sender.charAt(0).toUpperCase() }}</span>
        </div>
        <span class="text-xs font-medium text-text-secondary">{{ message.sender }}</span>
      </div>

      <div v-if="message.content === null" class="flex items-center gap-2 py-1">
        <PhLock class="w-4 h-4 text-text-muted" :weight="'duotone'" />
        <span class="text-sm text-text-muted italic">No se pudo descifrar el mensaje</span>
      </div>
      <p v-else class="text-sm leading-relaxed">{{ message.content }}</p>

      <div class="flex items-center justify-between mt-2 gap-2">
        <span class="text-xs opacity-60">{{ formatTime(message.createdAt) }}</span>
        <div class="flex items-center gap-2">
          <span :class="message.status === 'verificado' ? 'bg-success-light text-success' : 'bg-error-light text-error'" class="inline-flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded-full">
            <PhCheckCircle v-if="message.status === 'verificado'" class="w-3 h-3" :weight="'fill'" />
            <PhCircle v-else class="w-3 h-3" :weight="'duotone'" />
            {{ message.status === 'verificado' ? 'Mensaje verificado' : 'No verificado' }}
          </span>
          <button v-if="!isSender && message.status !== 'verificado'" @click="$emit('validate', message.id)" class="inline-flex items-center gap-1 text-xs font-semibold text-primary hover:text-primary-dark bg-primary-light hover:bg-primary/10 px-2.5 py-1 rounded-full transition-all duration-200 active:scale-95">
            <PhShieldCheck class="w-3 h-3" :weight="'duotone'" />
            Verificar
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { PhCheckCircle, PhCircle, PhShieldCheck, PhLock } from '@phosphor-icons/vue'

defineProps({
  message: { type: Object, required: true },
  isSender: { type: Boolean, default: false }
})

defineEmits(['validate'])

function formatTime(dateStr) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleTimeString('es-ES', { hour: '2-digit', minute: '2-digit' })
}
</script>
