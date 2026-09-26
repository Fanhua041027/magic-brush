/**
 * Wails event multiplexer. One native subscription is shared by all handlers
 * so unmounting one component cannot remove another component's listener.
 */
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const subscriptions = new Map()

function removeNative(event, entry) {
  if (entry.nativeUnsubscribe) {
    entry.nativeUnsubscribe()
  } else {
    EventsOff(event)
  }
}

export function on(event, handler) {
  if (typeof handler !== 'function') return () => {}

  let entry = subscriptions.get(event)
  if (!entry) {
    entry = { handlers: new Set(), nativeUnsubscribe: null }
    entry.nativeUnsubscribe = EventsOn(event, (...args) => {
      for (const fn of [...entry.handlers]) fn(...args)
    })
    subscriptions.set(event, entry)
  }
  entry.handlers.add(handler)

  let active = true
  return () => {
    if (!active) return
    active = false
    entry.handlers.delete(handler)
    if (entry.handlers.size === 0 && subscriptions.get(event) === entry) {
      subscriptions.delete(event)
      removeNative(event, entry)
    }
  }
}

export function off(event, handler) {
  const entry = subscriptions.get(event)
  if (!entry) return
  if (handler) {
    entry.handlers.delete(handler)
    if (entry.handlers.size > 0) return
  } else {
    entry.handlers.clear()
  }
  subscriptions.delete(event)
  removeNative(event, entry)
}

export function onMany(map) {
  const cleanups = Object.entries(map).map(([event, handler]) => on(event, handler))
  let active = true
  return () => {
    if (!active) return
    active = false
    cleanups.forEach(cleanup => cleanup())
  }
}
