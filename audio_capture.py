"""
系统音频捕获模块 — 支持立体声混音自动检测、降噪、可变采样率

功能：
  - 自动检测 Stereo Mix / 立体声混音 / 环回设备
  - 自适应采样率（根据设备能力选择最佳）
  - 实时噪声门控和音频归一化
  - 可选的音频增强预处理
  - 线程安全的录音控制
"""

import sys
import struct
import wave
import time
import json
import io
import os
import base64
import threading
from typing import Optional

import sounddevice as sd
import numpy as np

# ── 常量 ──────────────────────────────────────────────────────
TARGET_SAMPLE_RATE = 16000  # Whisper/DashScope 最佳采样率
RECORD_CHANNELS = 1
BLOCKSIZE = 1024
DTYPE = 'int16'
DTYPE_NP = np.int16
DTYPE_SAMPLE_WIDTH = 2  # 16-bit

# ── 全局状态 ───────────────────────────────────────────────────
_recording = False
_audio_frames: list[np.ndarray] = []
_frames_lock = threading.Lock()
_selected_device_id: Optional[int] = None


# ================================================================
# 设备检测
# ================================================================

def list_devices() -> list[dict]:
    """列出所有音频输入设备，标记立体声混音设备"""
    devices = sd.query_devices()
    result = []
    try:
        default_id = sd.default.device[0]
    except Exception:
        default_id = None

    host_apis = sd.query_hostapis()

    for i, d in enumerate(devices):
        max_input = d.get('max_input_channels', 0) if isinstance(d, dict) else d.max_input_channels
        if max_input and max_input > 0:
            name = str(d.get('name', '')) if isinstance(d, dict) else str(d.name)
            name_lower = name.lower()

            device_type = "mic"
            stereo_mix_kw = ["立体声混音", "stereo mix", "what u hear", "wave out", "混合输出", "loopback", "音频输出", "speaker"]
            mic_kw = ["麦克风", "microphone", "mic"]
            line_in_kw = ["line in", "线路输入", "辅助"]
            if any(kw in name_lower for kw in stereo_mix_kw):
                device_type = "stereo_mix"
            elif any(kw in name_lower for kw in mic_kw):
                device_type = "mic"
            elif any(kw in name_lower for kw in line_in_kw):
                device_type = "line_in"

            host_api_idx = d.get('host_api', 0) if isinstance(d, dict) else d.host_api
            host_api = host_apis[host_api_idx]['name'] if host_api_idx < len(host_apis) else "unknown"

            default_rate = int(d.get('default_samplerate', 48000) if isinstance(d, dict) else d.default_samplerate)

            result.append({
                'index': i,
                'name': name,
                'type': device_type,
                'channels': max_input,
                'default_samplerate': default_rate,
                'host_api': host_api,
                'is_default': (i == default_id),
            })
    return result


def find_best_input_device() -> tuple[Optional[int], Optional[str]]:
    """
    自动选择最佳输入设备：
    1. 优先立体声混音（捕获系统内部音频）
    2. 其次默认输入设备
    3. 最后第一个可用设备
    """
    devices = list_devices()
    if not devices:
        return None, None

    # 优先查找立体声混音设备
    for d in devices:
        if d['type'] == 'stereo_mix':
            return d['index'], d['name']

    # 其次默认设备
    for d in devices:
        if d['is_default']:
            return d['index'], d['name']

    # 最后第一个可用设备
    return devices[0]['index'], devices[0]['name']


# ================================================================
# 音频处理
# ================================================================

def _resample(audio: np.ndarray, orig_rate: int, target_rate: int = TARGET_SAMPLE_RATE) -> np.ndarray:
    """重采样到目标采样率"""
    if orig_rate == target_rate or len(audio) == 0:
        return audio
    target_len = int(len(audio) * target_rate / orig_rate)
    return np.interp(
        np.linspace(0, len(audio) - 1, target_len),
        np.arange(len(audio)),
        audio
    ).astype(np.float32)


def apply_noise_gate(audio: np.ndarray, threshold: float = 0.001) -> np.ndarray:
    """
    噪声门控：仅对极低底噪做轻微衰减
    立体声混音音频较干净，门控应保守
    """
    if len(audio) == 0:
        return audio

    frame_size = int(TARGET_SAMPLE_RATE * 0.03)  # 30ms 帧
    if frame_size < 1:
        return audio

    result = audio.copy()
    for start in range(0, len(audio), frame_size):
        end = min(start + frame_size, len(audio))
        frame = audio[start:end]
        rms = np.sqrt(np.mean(frame ** 2) + 1e-10)
        if rms < threshold:
            gain = max(0.3, (rms / threshold) ** 2)
            result[start:end] = frame * gain
        # 高能量段不做放大，保持原始动态范围

    return result


def normalize_audio(audio: np.ndarray, target_rms: float = 0.04) -> np.ndarray:
    """
    音频归一化到目标 RMS 电平
    保守增益，避免系统音频失真
    """
    if len(audio) == 0:
        return audio

    rms = np.sqrt(np.mean(audio ** 2) + 1e-10)
    if rms < 0.0001:
        return audio

    gain = target_rms / rms
    gain = min(max(gain, 0.5), 3.0)
    audio = audio * gain

    return np.clip(audio, -1.0, 1.0)


def enhance_audio(audio: np.ndarray, sample_rate: int) -> np.ndarray:
    """
    完整的音频增强管线：
    1. 重采样到 16kHz
    2. 归一化
    3. 噪声门控
    """
    if len(audio) == 0:
        return audio

    # 确保 float32
    audio = audio.astype(np.float32) if audio.dtype != np.float32 else audio.copy()

    # 如果整数 PCM，缩放
    if np.abs(audio).max() > 1.0:
        audio = audio / 32767.0

    # 重采样
    if sample_rate != TARGET_SAMPLE_RATE:
        audio = _resample(audio, sample_rate, TARGET_SAMPLE_RATE)
        sample_rate = TARGET_SAMPLE_RATE

    # 归一化
    audio = normalize_audio(audio)

    # 噪声门控
    audio = apply_noise_gate(audio)

    return audio


# ================================================================
# 录音回调
# ================================================================

def audio_callback(indata, frames, time_info, status):
    """PortAudio 回调 — 线程安全地追加音频帧"""
    global _recording, _audio_frames
    if status:
        print(f"[AudioCapture] Status: {status}", flush=True)
    if _recording:
        with _frames_lock:
            _audio_frames.append(indata.copy())


# ================================================================
# 录音控制
# ================================================================

def start_recording(device_id: Optional[int] = None, duration: Optional[float] = None):
    """开始录音"""
    global _recording, _audio_frames, _selected_device_id

    if device_id is None:
        device_id, device_name = find_best_input_device()
        if device_id is None:
            return json.dumps({'error': 'No input device available'})
        _selected_device_id = device_id

    with _frames_lock:
        _recording = True
        _audio_frames = []

    # 获取设备实际采样率
    try:
        dev_info = sd.query_devices(device_id)
        actual_rate = int(dev_info.get('default_samplerate', 48000) if isinstance(dev_info, dict) else dev_info.default_samplerate)
    except Exception:
        actual_rate = 48000

    try:
        stream = sd.InputStream(
            device=device_id,
            samplerate=actual_rate,
            channels=RECORD_CHANNELS,
            dtype=DTYPE,
            blocksize=BLOCKSIZE,
            callback=audio_callback,
        )
        stream.start()

        if duration:
            time.sleep(duration)
            stream.stop()
            stream.close()
            with _frames_lock:
                _recording = False
            return _get_audio_wav(actual_rate)
        else:
            return json.dumps({
                'status': 'started',
                'device': device_id,
                'sample_rate': actual_rate,
            })
    except Exception as e:
        with _frames_lock:
            _recording = False
        return json.dumps({'error': str(e)})


def stop_and_save():
    """停止录音并返回增强后的 WAV 数据"""
    global _recording, _audio_frames
    with _frames_lock:
        _recording = False
        frames = list(_audio_frames)
        _audio_frames = []

    if not frames:
        return json.dumps({'error': 'No audio captured'})

    # 获取录音时的采样率
    sample_rate = _get_last_sample_rate() or 48000
    return _frames_to_wav_json(frames, sample_rate)


def _get_audio_wav(record_sample_rate: int):
    """将录制的音频转换为增强后的 WAV"""
    global _audio_frames
    with _frames_lock:
        if not _audio_frames:
            return json.dumps({'error': 'No audio captured'})
        frames = list(_audio_frames)
        _audio_frames = []

    return _frames_to_wav_json(frames, record_sample_rate)


def _frames_to_wav_json(frames: list[np.ndarray], record_sample_rate: int) -> str:
    """将音频帧列表转换为增强后的 WAV JSON 响应"""
    if not frames:
        return json.dumps({'error': 'No audio captured'})

    data = np.concatenate(frames, axis=0)

    # 确保是 int16 并展平
    if data.dtype != np.int16:
        if data.dtype == np.float32 or data.dtype == np.float64:
            data = (data * 32767).astype(np.int16)
        else:
            data = data.astype(np.int16)

    if data.ndim > 1:
        data = data.flatten()

    # 应用音频增强
    enhanced = enhance_audio(data.astype(np.float32) / 32767.0, record_sample_rate)

    # 转回 int16
    enhanced_int16 = (enhanced * 32767).astype(np.int16)

    # 保存到临时文件
    temp_path = os.path.join(os.environ.get('TEMP', '/tmp'), 'magic_brush_capture.wav')
    total_samples = len(enhanced_int16)
    duration_sec = total_samples / TARGET_SAMPLE_RATE

    with wave.open(temp_path, 'wb') as wf:
        wf.setnchannels(RECORD_CHANNELS)
        wf.setsampwidth(DTYPE_SAMPLE_WIDTH)
        wf.setframerate(TARGET_SAMPLE_RATE)
        wf.writeframes(enhanced_int16.tobytes())

    # 同时返回 base64
    wav_buffer = io.BytesIO()
    with wave.open(wav_buffer, 'wb') as wf:
        wf.setnchannels(RECORD_CHANNELS)
        wf.setsampwidth(DTYPE_SAMPLE_WIDTH)
        wf.setframerate(TARGET_SAMPLE_RATE)
        wf.writeframes(enhanced_int16.tobytes())

    b64_data = base64.b64encode(wav_buffer.getvalue()).decode('utf-8')

    return json.dumps({
        'status': 'success',
        'duration': duration_sec,
        'samples': total_samples,
        'sample_rate': TARGET_SAMPLE_RATE,
        'path': temp_path,
        'base64': b64_data,
    })


_last_sample_rate_value: Optional[int] = None


def _get_last_sample_rate() -> Optional[int]:
    """获取最后使用的采样率"""
    global _last_sample_rate_value
    return _last_sample_rate_value


# ================================================================
# CLI 入口
# ================================================================

if __name__ == '__main__':
    command = sys.argv[1] if len(sys.argv) > 1 else 'list'

    if command == 'list':
        devices = list_devices()
        # 标记推荐的立体声混音设备
        for d in devices:
            if d['type'] == 'stereo_mix':
                d['recommended'] = True
        print(json.dumps(devices, ensure_ascii=False))

    elif command == 'start':
        device = int(sys.argv[2]) if len(sys.argv) > 2 else None
        _last_sample_rate_value = device
        result = start_recording(device)
        print(result)

    elif command == 'stop':
        result = stop_and_save()
        print(result)

    elif command == 'capture':
        device = int(sys.argv[2]) if len(sys.argv) > 2 else None
        dur = float(sys.argv[3]) if len(sys.argv) > 3 else 5.0
        result = start_recording(device, dur)
        print(result)
