"""Audio capture module — using sounddevice with blocking read."""

import sys
import threading
import time
from typing import Any

import numpy as np
import sounddevice as sd

WHISPER_RATE = 16000
CHANNELS = 1


def _get_device_attr(dev, attr):
    """兼容 dict 和对象两种 sounddevice 返回格式"""
    if isinstance(dev, dict):
        return dev.get(attr)
    return getattr(dev, attr, None)


def list_input_devices() -> list[dict[str, Any]]:
    """List all available audio input devices."""
    devices = []
    try:
        default_id = sd.default.device[0]
        all_devices = sd.query_devices()
        host_apis = sd.query_hostapis()
        for i, dev in enumerate(all_devices):
            max_input = _get_device_attr(dev, "max_input_channels")
            if max_input and max_input > 0:
                try:
                    name = str(_get_device_attr(dev, "name"))
                    # 标记特殊设备类型
                    device_type = "mic"
                    if "立体声混音" in name or "Stereo Mix" in name:
                        device_type = "stereo_mix"
                    elif "麦克风" in name or "Microphone" in name:
                        device_type = "mic"
                    host_api_idx = _get_device_attr(dev, "host_api")
                    host_api = host_apis[host_api_idx]["name"] if host_api_idx is not None else "unknown"
                    devices.append({
                        "id": i,
                        "name": name,
                        "type": device_type,
                        "channels": max_input,
                        "default_samplerate": int(_get_device_attr(dev, "default_samplerate") or 16000),
                        "host_api": host_api,
                        "is_default": i == default_id,
                    })
                except Exception as e:
                    print(f"[Audio] Error processing device {i}: {e}")
                    continue
    except Exception as e:
        print(f"[Audio] Error listing devices: {e}")
    return devices


_DEVICE_ID = None
_RATE = 48000  # 强制使用 48000 Hz（Intel SST 需要）
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
            max_input = _get_device_attr(dev, "max_input_channels")
            name = str(_get_device_attr(dev, "name"))
            if max_input and max_input > 0 and device_name in name:
                _DEVICE_ID = i
                print(f"[Audio] Found device by name: [{i}] {name} (rate={_RATE})")
                return _DEVICE_ID, _RATE
        print(f"[Audio] Device not found by name: {device_name}")

    _DEVICE_ID = device_id
    return _DEVICE_ID, _RATE


def get_device_config() -> tuple[int | None, int]:
    return _DEVICE_ID, _RATE


# 自动检测并使用默认输入设备
def _auto_select_device():
    """自动选择最佳输入设备"""
    try:
        default_id = sd.default.device[0]
        if default_id is not None:
            return default_id
    except Exception:
        pass
    return 0  # 默认使用设备 0

init_audio(_auto_select_device())


def _resample(audio: np.ndarray, orig_rate: int) -> np.ndarray:
    if orig_rate == WHISPER_RATE or audio.size == 0:
        return audio
    target_len = int(len(audio) * WHISPER_RATE / orig_rate)
    return np.interp(np.linspace(0, len(audio) - 1, target_len), np.arange(len(audio)), audio).astype(np.float32)


class AudioRecorder:
    """录音器 - 使用 sounddevice 阻塞读取"""

    def __init__(self):
        self._frames: list[np.ndarray] = []
        self._lock = threading.Lock()
        self._recording = False
        self._sample_rate: int = _RATE
        self._streaming_callback = None
        self._streaming_interval = 0.5
        self._last_streaming_time = 0
        self._stream = None
        self._read_thread = None
        self._stop_event = threading.Event()

        # 打开音频流
        self._open_stream()

    def _open_stream(self):
        try:
            print(f"[Audio] Opening stream: device={_DEVICE_ID}, rate={self._sample_rate}")
            self._stream = sd.InputStream(
                samplerate=self._sample_rate,
                channels=CHANNELS,
                dtype='float32',
                device=_DEVICE_ID,
                blocksize=int(self._sample_rate * 0.1),  # 0.1秒
            )
            self._stream.start()
            print(f"[Audio] Stream opened successfully")
            # 启动读取线程
            self._stop_event.clear()
            self._read_thread = threading.Thread(target=self._read_audio, daemon=True)
            self._read_thread.start()
        except Exception as e:
            print(f"[Audio] Failed to open stream: {e}")

    def _read_audio(self):
        """独立线程读取音频数据"""
        while not self._stop_event.is_set():
            try:
                data, overflowed = self._stream.read(int(self._sample_rate * 0.1))
                if overflowed:
                    print("[Audio] Warning: audio buffer overflow")
                audio = data[:, 0].copy()
                with self._lock:
                    if self._recording:
                        self._frames.append(audio)
                        # 调试：每秒打印一次音频 RMS
                        if len(self._frames) % 10 == 0:
                            rms = np.sqrt(np.mean(audio**2))
                            print(f"[Audio] read thread: RMS={rms:.6f}, frames={len(self._frames)}", flush=True)
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
            except Exception as e:
                if not self._stop_event.is_set():
                    print(f"[Audio] Read error: {e}")
                break

    def reopen(self, device_id: int | None = None) -> bool:
        self._stop_event.set()
        if self._read_thread:
            self._read_thread.join(timeout=1)
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
        self._stop_event.set()
        if self._read_thread:
            self._read_thread.join(timeout=1)
        if self._stream:
            self._stream.stop()
            self._stream.close()
