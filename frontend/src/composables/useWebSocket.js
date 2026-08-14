import { ref, onUnmounted } from 'vue'

export function useWebSocket() {
  const connected = ref(false)
  const connectionStatus = ref('connecting') // connecting | connected | disconnected | reconnecting
  const ws = ref(null)
  const listeners = []
  const pendingMessages = []
  let reconnectAttempts = 0
  let reconnectTimer = null

  function connect() {
    connectionStatus.value = 'connecting'
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    ws.value = new WebSocket(`${protocol}//${host}/ws`)

    ws.value.onopen = () => {
      connected.value = true
      connectionStatus.value = 'connected'
      reconnectAttempts = 0
      while (pendingMessages.length > 0) {
        ws.value.send(JSON.stringify(pendingMessages.shift()))
      }
    }

    ws.value.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        listeners.forEach(fn => fn(data))
      } catch (e) {
        console.error('Parse error:', e)
      }
    }

    ws.value.onclose = () => {
      connected.value = false
      connectionStatus.value = 'disconnected'
      scheduleReconnect()
    }

    ws.value.onerror = () => {
      connected.value = false
      connectionStatus.value = 'disconnected'
    }
  }

  function scheduleReconnect() {
    if (reconnectAttempts >= 5) {
      connectionStatus.value = 'failed'
      return
    }
    connectionStatus.value = 'reconnecting'
    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 10000)
    reconnectTimer = setTimeout(() => {
      reconnectAttempts++
      connect()
    }, delay)
  }

  function send(data) {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify(data))
    } else {
      pendingMessages.push(data)
    }
  }

  function onMessage(fn) {
    listeners.push(fn)
  }

  function disconnect() {
    clearTimeout(reconnectTimer)
    reconnectAttempts = 999 // prevent reconnect
    if (ws.value) {
      ws.value.close()
    }
  }

  onUnmounted(() => {
    clearTimeout(reconnectTimer)
    if (ws.value) {
      ws.value.close()
    }
  })

  connect()

  return { connected, connectionStatus, send, onMessage, disconnect }
}
