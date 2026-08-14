import { ref, onUnmounted } from 'vue'

export function useWebSocket() {
  const connected = ref(false)
  const connectionStatus = ref('connecting') // connecting | connected | disconnected | reconnecting
  const ws = ref(null)
  const listeners = []
  const pendingMessages = []
  let reconnectAttempts = 0
  let reconnectTimer = null
  let connectOptions = null

  function connect(options) {
    connectOptions = options
    connectionStatus.value = 'connecting'
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const url = `${protocol}//${host}/ws`
    console.log('[WS] Connecting to:', url, 'with options:', options)

    ws.value = new WebSocket(url)

    ws.value.onopen = () => {
      console.log('[WS] Connected!')
      connected.value = true
      connectionStatus.value = 'connected'
      reconnectAttempts = 0
      // Send join with token
      const joinMsg = {
        type: 'join',
        username: connectOptions.username,
        roomId: connectOptions.roomId,
        token: connectOptions.token
      }
      console.log('[WS] Sending join:', joinMsg)
      ws.value.send(JSON.stringify(joinMsg))
      while (pendingMessages.length > 0) {
        const msg = pendingMessages.shift()
        console.log('[WS] Sending pending:', msg)
        ws.value.send(JSON.stringify(msg))
      }
    }

    ws.value.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        console.log('[WS] Received:', data)
        listeners.forEach(fn => fn(data))
      } catch (e) {
        console.error('[WS] Parse error:', e)
      }
    }

    ws.value.onclose = (event) => {
      console.log('[WS] Disconnected:', event.code, event.reason)
      connected.value = false
      connectionStatus.value = 'disconnected'
      scheduleReconnect()
    }

    ws.value.onerror = (error) => {
      console.error('[WS] Error:', error)
      connected.value = false
      connectionStatus.value = 'disconnected'
    }
  }

  function scheduleReconnect() {
    if (reconnectAttempts >= 5 || !connectOptions) {
      connectionStatus.value = 'failed'
      return
    }
    connectionStatus.value = 'reconnecting'
    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 10000)
    reconnectTimer = setTimeout(() => {
      reconnectAttempts++
      connect(connectOptions)
    }, delay)
  }

  function send(data) {
    console.log('[WS] send() called:', data, 'readyState:', ws.value?.readyState)
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      console.log('[WS] Sending directly')
      ws.value.send(JSON.stringify(data))
    } else {
      console.log('[WS] Queuing (not connected)')
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

  return { connected, connectionStatus, send, onMessage, disconnect, connect }
}
