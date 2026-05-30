/**
 * WebSocket service for real-time communication
 */

class WebSocketService {
  constructor() {
    this.ws = null
    this.url = 'ws://127.0.0.1:18765/ws'
    this.reconnectAttempts = 0
    this.maxReconnectAttempts = 10
    this.reconnectDelay = 1000
    this.listeners = new Map()
    this.isConnected = false
    this.messageQueue = []
    this.lastErrorTime = 0
    this.errorThrottle = 5000 // 5 秒内不重复报错
  }

  /**
   * 连接 WebSocket
   */
  connect() {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return
    }

    try {
      this.ws = new WebSocket(this.url)

      this.ws.onopen = () => {
        console.log('[WebSocket] Connected')
        this.isConnected = true
        this.reconnectAttempts = 0

        // 发送队列中的消息
        while (this.messageQueue.length > 0) {
          const message = this.messageQueue.shift()
          this.send(message)
        }

        // 触发连接事件
        this.emit('connected')
      }

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          this.handleMessage(data)
        } catch (e) {
          console.error('[WebSocket] Failed to parse message:', e)
        }
      }

      this.ws.onclose = (event) => {
        console.log('[WebSocket] Disconnected:', event.code, event.reason)
        this.isConnected = false
        this.emit('disconnected', { code: event.code, reason: event.reason })

        // 自动重连
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
          this.reconnectAttempts++
          console.log(`[WebSocket] Reconnecting (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`)
          setTimeout(() => this.connect(), this.reconnectDelay * this.reconnectAttempts)
        }
      }

      this.ws.onerror = (error) => {
        console.error('[WebSocket] Error:', error)
        // 限制错误提示频率
        const now = Date.now()
        if (now - this.lastErrorTime > this.errorThrottle) {
          this.lastErrorTime = now
          this.emit('error', error)
        }
      }
    } catch (e) {
      console.error('[WebSocket] Connection failed:', e)
      this.emit('error', e)
    }
  }

  /**
   * 断开连接
   */
  disconnect() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.isConnected = false
    this.reconnectAttempts = this.maxReconnectAttempts // 阻止自动重连
  }

  /**
   * 发送消息
   */
  send(message) {
    if (!this.isConnected) {
      this.messageQueue.push(message)
      return false
    }

    try {
      this.ws.send(JSON.stringify(message))
      return true
    } catch (e) {
      console.error('[WebSocket] Send failed:', e)
      return false
    }
  }

  /**
   * 处理接收到的消息
   */
  handleMessage(data) {
    const { type } = data

    switch (type) {
      case 'stt-streaming':
        this.emit('stt-streaming', data.text)
        break
      case 'error':
        this.emit('error', data)
        break
      case 'pong':
        // 心跳响应
        break
      default:
        this.emit(type, data)
    }
  }

  /**
   * 注册事件监听器
   */
  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    this.listeners.get(event).add(callback)
    return () => this.off(event, callback)
  }

  /**
   * 移除事件监听器
   */
  off(event, callback) {
    if (this.listeners.has(event)) {
      this.listeners.get(event).delete(callback)
    }
  }

  /**
   * 触发事件
   */
  emit(event, data) {
    if (this.listeners.has(event)) {
      this.listeners.get(event).forEach(callback => {
        try {
          callback(data)
        } catch (e) {
          console.error(`[WebSocket] Event handler error (${event}):`, e)
        }
      })
    }
  }

  /**
   * 发送心跳
   */
  startHeartbeat(interval = 30000) {
    this.heartbeatInterval = setInterval(() => {
      this.send({ type: 'ping' })
    }, interval)
  }

  /**
   * 停止心跳
   */
  stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval)
      this.heartbeatInterval = null
    }
  }
}

// 创建单例
export const wsService = new WebSocketService()

export default wsService
