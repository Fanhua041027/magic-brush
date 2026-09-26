import {
  CancelRequest,
  CancelRunningTask,
  CheckScreenCapturePermission,
  ClearResume,
  GetAudioLevel,
  GetDomainCategories,
  GetInitStatus,
  GetModels,
  GetScreenshotPreview,
  GetSettings,
  GetSTTStatus,
  GetSTTDevices,
  SetSTTDevice,
  STTStart,
  STTStop,
  ToggleSTT,
  GetKBStatus,
  LoadKB,
  SearchKB,
  SelectKBDirectory,
  ClearKB,
  SetPendingUserMessage,
  MoveWindow,
  OpenScreenCaptureSettings,
  ParseResume,
  RemoveFocus,
  RequestScreenCapturePermission,
  RestoreFocus,
  SaveImageToFile,
  ScrollContent,
  SelectResume,
  SetWindowAlwaysOnTop,
  StartRecordingKey,
  StopRecordingKey,
  TestConnection,
  ToggleClickThrough,
  ToggleVisibility,
  TriggerSolve,
  TriggerScreenshot,
  TriggerSend,
  RemoveScreenshot,
  ClearScreenshots,
  UpdateSettings,
  ChatWithDeepSeek,
  ChatWithDeepSeekStream,
  ChatWithDeepSeekStreamWithContext,
  ChatWithScreenshot,
  IsStandaloneInterview,
  OpenStandaloneInterview,
  AuthCurrent,
  // Audio & Interview API
  AudioListDevices,
  AudioCapture,
  AudioTranscribe,
  AudioIsAvailable,
  GenerateInterviewAnswer,
} from '../../wailsjs/go/app/App'

import { Quit } from '../../wailsjs/runtime/runtime'

export const api = {
  getSettings: () => GetSettings(),
  syncSettings: (json) => UpdateSettings(json),
  updateSettings: (json) => UpdateSettings(json),

  getModels: (apiKey) => GetModels(apiKey),
  testConnection: (apiKey, model) => TestConnection(apiKey, model),

  triggerSolve: () => TriggerSolve(),
  triggerScreenshot: () => TriggerScreenshot(),
  triggerSend: () => TriggerSend(),
  removeScreenshot: (index) => RemoveScreenshot(index),
  clearScreenshots: () => ClearScreenshots(),
  cancelTask: () => CancelRunningTask(),
  cancelRequest: (requestId) => CancelRequest(requestId),

  startRecordingKey: (action) => StartRecordingKey(action),
  stopRecordingKey: () => StopRecordingKey(),

  selectResume: () => SelectResume(),
  clearResume: () => ClearResume(),
  parseResume: () => ParseResume(),

  restoreFocus: () => RestoreFocus(),
  removeFocus: () => RemoveFocus(),
  moveWindow: (x, y) => MoveWindow(x, y),
  scrollContent: (dir) => ScrollContent(dir),
  setAlwaysOnTop: (v) => SetWindowAlwaysOnTop(v),
  toggleVisibility: () => ToggleVisibility(),
  toggleClickThrough: () => ToggleClickThrough(),
  quit: () => Quit(),

  getInitStatus: () => GetInitStatus(),
  getDomainCategories: () => GetDomainCategories(),

  getScreenshotPreview: (q, s, g, n, m) => GetScreenshotPreview(q, s, g, n, m),
  checkScreenCapturePermission: () => CheckScreenCapturePermission(),
  requestScreenCapturePermission: () => RequestScreenCapturePermission(),
  openScreenCaptureSettings: () => OpenScreenCaptureSettings(),

  saveImageToFile: (b64) => SaveImageToFile(b64),

  getSTTStatus: () => GetSTTStatus(),
  getSTTDevices: () => GetSTTDevices(),
  setSTTDevice: (id) => SetSTTDevice(id),
  sttStart: () => STTStart(),
  sttStop: () => STTStop(),
  toggleSTT: () => ToggleSTT(),
  getKBStatus: () => GetKBStatus(),
  loadKB: (path) => LoadKB(path),
  searchKB: (query) => SearchKB(query),
  selectKBDirectory: () => SelectKBDirectory(),
  clearKB: () => ClearKB(),
  setUserMessage: (text) => SetPendingUserMessage(text),

  chatWithDeepSeek: (message) => ChatWithDeepSeek(message),
  chatWithDeepSeekStream: (requestId, message) => ChatWithDeepSeekStream(requestId, message),
  chatWithDeepSeekStreamWithContext: (requestId, messages) => ChatWithDeepSeekStreamWithContext(requestId, messages),
  chatWithScreenshot: (requestId, message, screenshot, context) => ChatWithScreenshot(requestId, message, screenshot, context),
  isStandaloneInterview: () => IsStandaloneInterview(),
  getCurrentUser: () => AuthCurrent(),
  openStandaloneInterview: () => OpenStandaloneInterview(),

  // Audio & Interview API
  audioListDevices: () => AudioListDevices(),
  audioCapture: (deviceID, durationSec) => AudioCapture(deviceID, durationSec),
  audioTranscribe: (base64Data) => AudioTranscribe(base64Data),
  audioIsAvailable: () => AudioIsAvailable(),
  generateInterviewAnswer: (query) => GenerateInterviewAnswer(query),

  audioLevel: async () => {
    try {
      return await GetAudioLevel()
    } catch {
      return { level: 0 }
    }
  },
}
