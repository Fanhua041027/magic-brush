/**
 * @vitest-environment happy-dom
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useChatStore } from '../chat'

// Mock Wails runtime bindings
vi.mock('../../../wailsjs/go/app/App', () => ({
  CancelRequest: vi.fn(() => Promise.resolve(true)),
  ChatWithDeepSeek: vi.fn(() => Promise.resolve('mock answer')),
  ChatWithDeepSeekStream: vi.fn(() => Promise.resolve()),
  ChatWithScreenshot: vi.fn(() => Promise.resolve()),
  ChatWithScreenshotSync: vi.fn(() => Promise.resolve('mock answer')),
  ChatWithDeepSeekStreamWithContext: vi.fn(() => Promise.resolve()),
}))

// Mock settings store
vi.mock('../settings', () => ({
  useSettingsStore: vi.fn(() => ({
    settings: { resumeContent: '' },
  })),
}))

function setupLocalStorage() {
  const store = {}
  Object.defineProperty(globalThis, 'localStorage', {
    value: {
      getItem: vi.fn((key) => store[key] ?? null),
      setItem: vi.fn((key, value) => { store[key] = String(value) }),
      removeItem: vi.fn((key) => { delete store[key] }),
      clear: vi.fn(() => { Object.keys(store).forEach(k => delete store[k]) }),
      get length() { return Object.keys(store).length },
      key: vi.fn((i) => Object.keys(store)[i] ?? null),
    },
    writable: true,
    configurable: true,
  })
}

describe('ChatStore', () => {
  const requestId = 'test-request'

  function startStream(store) {
    store.beginRequest(requestId)
  }

  beforeEach(() => {
    setActivePinia(createPinia())
    setupLocalStorage()
  })

  it('initializes with empty state', () => {
    const store = useChatStore()
    expect(store.isVisible).toBe(false)
    expect(store.isLoading).toBe(false)
    expect(store.messages).toEqual([])
  })

  it('show and hide work correctly', () => {
    const store = useChatStore()
    expect(store.isVisible).toBe(false)
    store.show()
    expect(store.isVisible).toBe(true)
    store.hide()
    expect(store.isVisible).toBe(false)
  })

  it('toggle flips visibility', () => {
    const store = useChatStore()
    store.toggle()
    expect(store.isVisible).toBe(true)
    store.toggle()
    expect(store.isVisible).toBe(false)
  })

  it('addMessage adds a message with metadata', () => {
    const store = useChatStore()
    store.addMessage('user', 'Hello')
    expect(store.messages).toHaveLength(1)
    expect(store.messages[0].role).toBe('user')
    expect(store.messages[0].content).toBe('Hello')
    expect(store.messages[0].time).toBeDefined()
    expect(store.messages[0].timestamp).toBeDefined()
  })

  it('adds multiple messages in order', () => {
    const store = useChatStore()
    store.addMessage('user', 'Q1')
    store.addMessage('assistant', 'A1')
    store.addMessage('user', 'Q2')
    expect(store.messages).toHaveLength(3)
    expect(store.messages[0].content).toBe('Q1')
    expect(store.messages[1].content).toBe('A1')
    expect(store.messages[2].content).toBe('Q2')
  })

  it('clearMessages removes all messages', () => {
    const store = useChatStore()
    store.addMessage('user', 'test')
    store.clearMessages()
    expect(store.messages).toHaveLength(0)
  })

  it('handleStreamChunk adds assistant message', () => {
    const store = useChatStore()
    startStream(store)
    store.handleStreamChunk({ requestId, chunk: 'Hello' })
    expect(store.messages).toHaveLength(1)
    expect(store.messages[0].role).toBe('assistant')
    expect(store.messages[0]._streaming).toBe(true)
    expect(store.messages[0].content).toBe('Hello')
  })

  it('handleStreamChunk appends to existing content', () => {
    const store = useChatStore()
    startStream(store)
    store.handleStreamChunk({ requestId, chunk: 'Hello' })
    store.handleStreamChunk({ requestId, chunk: ' World' })
    expect(store.messages[0].content).toBe('Hello World')
  })

  it('handleStreamDone ends streaming', () => {
    const store = useChatStore()
    startStream(store)
    store.handleStreamChunk({ requestId, chunk: 'test' })
    store.handleStreamDone({ requestId })
    expect(store.isLoading).toBe(false)
    expect(store.messages[0]._streaming).toBeUndefined()
  })

  it('handleStreamError adds error to message', () => {
    const store = useChatStore()
    startStream(store)
    store.handleStreamChunk({ requestId, chunk: 'partial' })
    store.handleStreamError({ requestId, error: 'API timeout' })
    expect(store.messages[0].content).toContain('API timeout')
    expect(store.isLoading).toBe(false)
  })

  it('ignores stream events from a different request', () => {
    const store = useChatStore()
    startStream(store)
    store.handleStreamChunk({ requestId: 'stale-request', chunk: 'stale' })
    expect(store.messages).toEqual([])
    expect(store.isLoading).toBe(true)
  })

  it('rejects invalid imports without changing messages', async () => {
    const store = useChatStore()
    store.addMessage('user', 'keep me')
    const file = new File([
      JSON.stringify({ version: 1, messages: [{ role: 'system', content: 'bad' }] }),
    ], 'bad.json', { type: 'application/json' })

    await expect(store.importHistory(file)).rejects.toThrow('消息角色无效')
    expect(store.messages).toHaveLength(1)
    expect(store.messages[0].content).toBe('keep me')
  })

  it('imports a validated versioned chat file atomically', async () => {
    const store = useChatStore()
    const file = new File([
      JSON.stringify({ version: 1, messages: [{ role: 'user', content: 'hello' }] }),
    ], 'chat.json', { type: 'application/json' })

    await expect(store.importHistory(file)).resolves.toBe(1)
    expect(store.messages[0]).toMatchObject({ role: 'user', content: 'hello' })
    expect(store.messages[0].id).toBeTruthy()
  })

  it('does not send empty messages', async () => {
    const store = useChatStore()
    await store.sendMessage('')
    expect(store.messages).toHaveLength(0)
    expect(store.isLoading).toBe(false)
  })

  it('does not double-send while loading', async () => {
    const store = useChatStore()
    startStream(store)
    await store.sendMessage('hello')
    expect(store.messages).toHaveLength(0)
  })

  it('saves and loads conversation list', () => {
    const store = useChatStore()
    store.addMessage('user', 'test')
    store.saveCurrentConversation()
    expect(store.savedConversations).toHaveLength(1)
    expect(store.savedConversations[0].title).toBe('test')
  })

  it('starts new conversation', () => {
    const store = useChatStore()
    store.addMessage('user', 'old msg')
    store.saveCurrentConversation()
    store.startNewConversation()
    expect(store.messages).toHaveLength(0)
    expect(store.savedConversations).toHaveLength(1)
  })

  it('deletes conversation', () => {
    const store = useChatStore()
    store.addMessage('user', 'test')
    store.saveCurrentConversation()
    const id = store.savedConversations[0].id
    store.deleteConversation(id)
    expect(store.savedConversations).toHaveLength(0)
  })

  it('has all expected methods', () => {
    const store = useChatStore()
    expect(store.sendMessage).toBeDefined()
    expect(store.sendMessageWithScreenshot).toBeDefined()
    expect(store.exportHistory).toBeDefined()
    expect(store.importHistory).toBeDefined()
    expect(store.handleStreamChunk).toBeDefined()
    expect(store.handleStreamDone).toBeDefined()
    expect(store.handleStreamError).toBeDefined()
  })
})
