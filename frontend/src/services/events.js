/**
 * Wails Event Bus — 统一注册所有后端事件
 * 在 App.vue onMounted 中调用 initEvents() 一次即可
 */
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const handlers = {}

/**
 * 注册事件处理函数
 * @param {string} event - 事件名
 * @param {Function} handler - 处理函数
 */
export function on(event, handler) {
  if (!handlers[event]) {
    handlers[event] = []
    EventsOn(event, (...args) => {
      handlers[event].forEach(fn => fn(...args))
    })
  }
  handlers[event].push(handler)
}

/**
 * 移除事件的所有处理函数，同时清理 Wails 底层监听
 * @param {string} event - 事件名
 */
export function off(event) {
  delete handlers[event]
  EventsOff(event)
}

/**
 * 批量注册事件
 * @param {Object} map - { eventName: handler }
 */
export function onMany(map) {
  for (const [event, handler] of Object.entries(map)) {
    on(event, handler)
  }
}
