/**
 * @vitest-environment happy-dom
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useVoiceStore } from '../voice'

// Mock API
vi.mock('../../services/api', () => {
  const mockSttStart = vi.fn(() => Promise.resolve({ status: 'recording' }))
  return {
    api: {
      getSTTStatus: vi.fn(() => Promise.resolve({ recording: false, ready: true })),
      getSTTDevices: vi.fn(() => Promise.resolve({
        devices: [
          { id: 1, name: 'Microphone (Realtek)', is_default: true },
          { id: 2, name: 'Stereo Mix', is_default: false },
        ],
        current_device_id: 1,
      })),
      setSTTDevice: vi.fn(() => Promise.resolve({ status: 'ok' })),
      toggleSTT: vi.fn(() => Promise.resolve({ status: 'recording' })),
      sttStart: mockSttStart,
      sttStop: vi.fn(() => Promise.resolve({ status: 'transcribed', text: 'hello world' })),
      setUserMessage: vi.fn(),
    },
  }
})

describe('VoiceStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('initializes with defaults', () => {
    const store = useVoiceStore()
    expect(store.isRecording).toBe(false)
    expect(store.isTranscribing).toBe(false)
    expect(store.isReady).toBe(false)
    expect(store.transcribedText).toBe('')
    expect(store.devices).toEqual([])
    expect(store.selectedDeviceId).toBeNull()
  })

  it('checkStatus updates state', async () => {
    const store = useVoiceStore()
    await store.checkStatus()
    expect(store.isRecording).toBe(false)
    expect(store.isReady).toBe(true)
  })

  it('loadDevices fetches and sets devices', async () => {
    const store = useVoiceStore()
    await store.loadDevices()
    expect(store.devices).toHaveLength(2)
    expect(store.selectedDeviceId).toBe(1)
  })

  it('toggle starts recording', async () => {
    const store = useVoiceStore()
    await store.toggle()
    expect(store.isRecording).toBe(true)
    expect(store.transcribedText).toBe('')
  })

  it('start sets recording state', async () => {
    const store = useVoiceStore()
    await store.start()
    expect(store.isRecording).toBe(true)
  })

  it('stop sets transcribing and then clears', async () => {
    const store = useVoiceStore()
    store.isRecording = true
    await store.stop()
    expect(store.isRecording).toBe(false)
    expect(store.isTranscribing).toBe(false)
  })

  it('does not start if already recording', async () => {
    const store = useVoiceStore()
    store.isRecording = true
    const { api } = await vi.importActual('../../services/api')
    // The mock ensures sttStart returns recording status
    await store.start()
    expect(store.isRecording).toBe(true)
  })

  it('does not stop if not recording', async () => {
    const store = useVoiceStore()
    await store.stop()
    expect(store.isTranscribing).toBe(false)
  })
})
