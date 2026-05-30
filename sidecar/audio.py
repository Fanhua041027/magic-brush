"""Audio capture module — using sounddevice callback."""

import sys
import threading
import time
from typing import Any

import numpy as np
import sounddevice as sd

WHISPER_RATE = 16000
CHANNELS = 1


def list_input_devices() -> list[dict[str, Any]]:
    """List all available audio input devices."""
    devices = []
    try:
        default_id = sd.default.device[0]
        for i, dev in enumerate(sd.query_devices()):
            if dev["max_input_channels"] > 0:
                name = dev["name"]
                # 标记特殊设备类型
                device_type = "mic"
                if "立体声混音" in name or "Stereo Mix" in name:
                    device_type = "stereo_mix"
                elif "麦克风" in name or "Microphone" in name:
                    device_type = "mic"
                devices.append({
                    "id": i,
                    "name": name,
                    "type": device_type,
                    "channels": dev["max_input_channels"],
                    "default_samplerate": int(dev["default_samplerate"]),
                    "host_api": sd.query_hostapis(dev["host_api"])["name"],
                    "is_default": i == default_id,
                })
    except Exception:
        pass
    return devices


_DEVICE_ID = None
_RATE = WHISPER_RATE
_DEVICE_NAME = None


def init_audio(device_id: int | None = None, device_name: str | None = None) -> tuple[int | None, int]:
    """初始化音频设备

    Args:
        device_id: 设备 ID（数字索引）
        device_name: 设备名称（模糊匹配）
    """
    global _DEVICE_ID, _RATE, _DEVICE_NAME
    _DEVICE_NAME = device_name

    if device_name:
        # 按名称查找设备
        for i, dev in enumerate(sd.query_devices()):
            if dev['max_input_channels'] > 0 and device_name in dev['name']:
                _DEVICE_ID = i
                _RATE = int(dev['default_samplerate'])
                print(f"[Audio] Found device by name: [{i}] {dev['name']}")
                return _DEVICE_ID, _RATE
        print(f"[Audio] Device not found by name: {device_name}")

    _DEVICE_ID = device_id
    _RATE = WHISPER_RATE
    return _DEVICE_ID, _RATE


def get_device_config() -> tuple[int | None, int]:
    return _DEVICE_ID, _RATE


init_audio(0)  # 使用设备 0 (Microsoft 声音映射器)


def _resample(audio: np.ndarray, orig_rate: int) -> np.ndarray:
    if orig_rate == WHISPER_RATE or audio.size == 0:
        return audio
    target_len = int(len(audio) * WHISPER_RATE / orig_rate)
    return np.interp(np.linspace(0, len(audio) - 1, target_len), np.arange(len(audio)), audio).astype(np.float32)


class AudioRecorder:
    """录音器 - 使用 sounddevice 回调"""

    def __init__(self):
        self._frames: list[np.ndarray] = []
        self._lock = threading.Lock()
        self._recording = False
        self._sample_rate: int = _RATE
        self._streaming_callback = None
        self._streaming_interval = 0.5
        self._last_streaming_time = 0
        self._stream = None

        # 打开音频流
        self._open_stream()

    def _open_stream(self):
        try:
            self._stream = sd.InputStream(
                samplerate=self._sample_rate,
                channels=CHANNELS,
                dtype='float32',
                device=_DEVICE_ID,
                callback=self._audio_callback,
                blocksize=int(self._sample_rate * 0.1),  # 0.1秒
            )
            self._stream.start()
        except Exception as e:
            print(f"[Audio] Failed to open stream: {e}")

    def _audio_callback(self, indata, frames, time_info, status):
        """音频回调"""
        audio = indata[:, 0].copy()
        with self._lock:
            if self._recording:
                self._frames.append(audio)
                now = time.time()
                if (self._streaming_callback and
                    now - self._last_streaming_time >= self._streaming_interval):
                    self._last_streaming_time = now
                    if self._frames:
                        chunk = np.concatenate(self._frames[-10:], axis=0).flatten()
                        threading.Thread(
                            target=self._streaming_callback,
                            args=(chunk,),
                            daemon=True
                        ).start()

    def reopen(self, device_id: int | None = None) -> bool:
        if self._stream:
            self._stream.stop()
            self._stream.close()
        self._stream = None
        # 更新采样率（设备切换后可能改变）
        self._sample_rate = _RATE
        self._open_stream()
        return self._stream is not None

    @property
    def device_id(self) -> int | None:
        return _DEVICE_ID

    @property
    def sample_rate(self) -> int:
        return self._sample_rate

    def set_streaming_callback(self, callback):
        self._streaming_callback = callback

    def start_recording(self):
        with self._lock:
            self._frames = []
            self._last_streaming_time = time.time()
        self._recording = True

    def stop_recording(self) -> np.ndarray:
        self._recording = False
        self._streaming_callback = None
        with self._lock:
            if not self._frames:
                return np.zeros(0, dtype=np.float32)
            raw = np.concatenate(self._frames, axis=0).flatten()
        return _resample(raw, self._sample_rate)

    def record_until_silence(self, max_seconds=30, silence_threshold=0.005, silence_duration=2.0) -> np.ndarray:
        self.start_recording()
        chunks_per_second = 10
        silence_chunks_needed = int(silence_duration * chunks_per_second)
        max_chunks = int(max_seconds * chunks_per_second)
        silence_chunks = 0
        min_chunks = int(0.5 * chunks_per_second)

        for i in range(max_chunks):
            time.sleep(1 / chunks_per_second)
            with self._lock:
                if not self._frames:
                    continue
                recent = self._frames[-min(3, len(self._frames)):]
                rms = float(np.sqrt(np.mean(np.concatenate(recent) ** 2)))
            if i < min_chunks:
                continue
            if rms < silence_threshold:
                silence_chunks += 1
            else:
                silence_chunks = 0
            if silence_chunks >= silence_chunks_needed:
                break

        return self.stop_recording()

    def close(self):
        if self._stream:
            self._stream.stop()
            self._stream.close()
