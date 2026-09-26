/**
 * @vitest-environment happy-dom
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSolutionStore } from '../solution'

// Mock markdown-latex utils
vi.mock('../../utils/markdown-latex', () => ({
  renderMarkdownWithLatex: vi.fn((md) => `<div>${md}</div>`),
}))

// Mock API
vi.mock('../../services/api', () => ({
  api: {
    saveImageToFile: vi.fn(() => Promise.resolve()),
  },
}))

describe('SolutionStore', () => {
  const requestId = 'test-request'

  function startStream(store, keepContext = false) {
    store.beginRequest(requestId)
    store.handleStreamStart({ requestId }, keepContext)
  }

  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('initializes with empty state', () => {
    const store = useSolutionStore()
    expect(store.history).toEqual([])
    expect(store.isLoading).toBe(false)
    expect(store.isAppending).toBe(false)
    expect(store.isThinking).toBe(false)
    expect(store.errorState.show).toBe(false)
  })

  it('handleStreamStart creates new history item', () => {
    const store = useSolutionStore()
    startStream(store)
    expect(store.history).toHaveLength(1)
    expect(store.history[0].rounds).toHaveLength(1)
    expect(store.activeHistoryIndex).toBe(0)
  })

  it('handleStreamChunk updates response in round', () => {
    const store = useSolutionStore()
    startStream(store)
    store.handleStreamChunk({ requestId, chunk: 'Hello' })
    const round = store.history[0].rounds[0]
    expect(round.aiResponse).toBe('Hello')
  })

  it('handleStreamChunk appends to response', () => {
    const store = useSolutionStore()
    startStream(store)
    store.handleStreamChunk({ requestId, chunk: 'Hello' })
    store.handleStreamChunk({ requestId, chunk: ' World' })
    const round = store.history[0].rounds[0]
    expect(round.aiResponse).toBe('Hello World')
  })

  it('handleThinkingChunk sets thinking state', () => {
    const store = useSolutionStore()
    startStream(store)
    store.handleThinkingChunk({ requestId, thinking: 'thinking step 1' })
    expect(store.isThinking).toBe(true)
    const round = store.history[0].rounds[0]
    expect(round.thinking).toBe('thinking step 1')
    expect(round.thinkingStatus).toBe('Thinking Process')
  })

  it('handleThinkingChunk detects code generation', () => {
    const store = useSolutionStore()
    startStream(store)
    store.handleThinkingChunk({ requestId, thinking: 'writing function main()' })
    expect(store.thinkingStatusText).toBe('Generating Code...')
  })

  it('handleSolution ends loading and stores data', () => {
    const store = useSolutionStore()
    store.isLoading = true
    startStream(store)
    store.handleSolution({ requestId, content: 'final answer' })
    expect(store.isLoading).toBe(false)
    const round = store.history[0].rounds[0]
    expect(round.aiResponse).toBe('final answer')
  })

  it('handleInlineError sets error on current round', () => {
    const store = useSolutionStore()
    startStream(store)
    const result = store.handleInlineError({ title: 'Error', desc: 'Something failed' })
    expect(result).toBe(true)
    expect(store.history[0].rounds[0].error.title).toBe('Error')
  })

  it('clearInlineError removes error', () => {
    const store = useSolutionStore()
    startStream(store)
    store.handleInlineError({ title: 'Error', desc: 'fail' })
    store.clearInlineError()
    expect(store.history[0].rounds[0].error).toBeNull()
  })

  it('ignores events from stale requests', () => {
    const store = useSolutionStore()
    startStream(store)
    store.handleStreamChunk({ requestId: 'stale-request', chunk: 'stale' })
    store.handleThinkingChunk({ requestId: 'stale-request', thinking: 'stale' })
    expect(store.history[0].rounds[0].aiResponse).toBe('')
    expect(store.history[0].rounds[0].thinking).toBe('')
  })

  it('deleteHistory removes item', () => {
    const store = useSolutionStore()
    startStream(store)
    expect(store.history).toHaveLength(1)
    store.deleteHistory(0)
    expect(store.history).toHaveLength(0)
  })

  it('selectHistory changes active index', () => {
    const store = useSolutionStore()
    startStream(store)
    startStream(store)
    expect(store.history).toHaveLength(2)
    store.selectHistory(1)
    expect(store.activeHistoryIndex).toBe(1)
  })

  it('error state defaults are correct', () => {
    const store = useSolutionStore()
    expect(store.errorState.show).toBe(false)
    expect(store.errorState.icon).toBe('⚠️')
  })

  it('renderMarkdown returns wrapped content', () => {
    const store = useSolutionStore()
    const html = store.renderMarkdown('# Hello')
    expect(html).toContain('Hello')
  })

  it('getSummary returns truncated text', () => {
    const store = useSolutionStore()
    startStream(store)
    store.handleStreamChunk('This is a long response for testing')
    const summary = store.getSummary(store.history[0])
    expect(summary).toBeTruthy()
  })
})
