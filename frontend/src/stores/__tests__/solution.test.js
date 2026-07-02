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
    store.handleStreamStart(false)
    expect(store.history).toHaveLength(1)
    expect(store.history[0].rounds).toHaveLength(1)
    expect(store.activeHistoryIndex).toBe(0)
  })

  it('handleStreamChunk updates response in round', () => {
    const store = useSolutionStore()
    store.handleStreamStart(false)
    store.handleStreamChunk('Hello')
    const round = store.history[0].rounds[0]
    expect(round.aiResponse).toBe('Hello')
  })

  it('handleStreamChunk appends to response', () => {
    const store = useSolutionStore()
    store.handleStreamStart(false)
    store.handleStreamChunk('Hello')
    store.handleStreamChunk(' World')
    const round = store.history[0].rounds[0]
    expect(round.aiResponse).toBe('Hello World')
  })

  it('handleThinkingChunk sets thinking state', () => {
    const store = useSolutionStore()
    store.handleStreamStart(false)
    store.handleThinkingChunk('thinking step 1')
    expect(store.isThinking).toBe(true)
    const round = store.history[0].rounds[0]
    expect(round.thinking).toBe('thinking step 1')
    expect(round.thinkingStatus).toBe('Thinking Process')
  })

  it('handleThinkingChunk detects code generation', () => {
    const store = useSolutionStore()
    store.handleStreamStart(false)
    store.handleThinkingChunk('writing function main()')
    expect(store.thinkingStatusText).toBe('Generating Code...')
  })

  it('handleSolution ends loading and stores data', () => {
    const store = useSolutionStore()
    store.isLoading = true
    store.handleStreamStart(false)
    store.handleSolution('final answer')
    expect(store.isLoading).toBe(false)
    const round = store.history[0].rounds[0]
    expect(round.aiResponse).toBe('final answer')
  })

  it('handleInlineError sets error on current round', () => {
    const store = useSolutionStore()
    store.handleStreamStart(false)
    const result = store.handleInlineError({ title: 'Error', desc: 'Something failed' })
    expect(result).toBe(true)
    expect(store.history[0].rounds[0].error.title).toBe('Error')
  })

  it('clearInlineError removes error', () => {
    const store = useSolutionStore()
    store.handleStreamStart(false)
    store.handleInlineError({ title: 'Error', desc: 'fail' })
    store.clearInlineError()
    expect(store.history[0].rounds[0].error).toBeNull()
  })

  it('deleteHistory removes item', () => {
    const store = useSolutionStore()
    store.handleStreamStart(false)
    expect(store.history).toHaveLength(1)
    store.deleteHistory(0)
    expect(store.history).toHaveLength(0)
  })

  it('selectHistory changes active index', () => {
    const store = useSolutionStore()
    store.handleStreamStart(false)
    store.handleStreamStart(false)
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
    store.handleStreamStart(false)
    store.handleStreamChunk('This is a long response for testing')
    const summary = store.getSummary(store.history[0])
    expect(summary).toBeTruthy()
  })
})
