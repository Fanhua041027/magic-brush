"""Audio capture module — 线程安全、多设备兼容、内置音频增强。

核心改进：
- 线程安全的帧缓冲区（使用 Lock 保护所有访问）
- 自动设备检测和最佳采样率选择
- 内置音频归一化（提升低音量录音识别率）
- 更好的 Intel SST 驱动兼容性
- 静音检测和自动跳过
"""

import sys
import threading
import time
from typing import Any, Optional, Callable
from collections import deque

import numpy as np
import sounddevice as sd

TARGET_RATE = 16000  # Whisper/DashScope 最佳采样率
CHANNELS = 1
BLOCKSIZE_MS = 50  # 50ms blocks


def _get_device_attr(dev, attr):
    """兼容 dict 和对象两种 sounddevice 返回格式"""
    if isinstance(dev, dict):
        return dev.get(attr)
    return getattr(dev, attr, None)


# ── 设备管理 ─────────────────────────────────────────────

def list_input_devices() -> list[dict[str, Any]]:
    """列出所有音频输入设备，标记类型"""
    devices = []
    try:
        default_id = sd.default.device[0]
        all_devices = sd.query_devices()
        host_apis = sd.query_hostapis()

        for i, dev in enumerate(all_devices):
            max_input = _get_device_attr(dev, "max_input_channels")
            if not (max_input and max_input > 0):
                continue

            try:
                name = str(_get_device_attr(dev, "name") or "")
                name_lower = name.lower()

                # 设备类型检测
                device_type = "mic"
                if any(kw in name_lower for kw in
                       ["立体声混音", "stereo mix", "what u hear",
                        "wave out", "loopback", "音频输出"]):
                    device_type = "stereo_mix"
                elif any(kw in name_lower for kw in
                         ["cable output", "vb-audio virtual cable", "vb-audio point"]):
                    device_type = "cable"
                elif any(kw in name_lower for kw in ["麦克风", "microphone", "mic"]):
                    device_type = "mic"

                host_api_idx = _get_device_attr(dev, "host_api")
                host_api_name = "unknown"
                if host_api_idx is not None and host_api_idx < len(host_apis):
                    host_api_name = host_apis[host_api_idx].get("name", "unknown")

                default_rate = _get_device_attr(dev, "default_samplerate") or 16000

                devices.append({
                    "id": i,
                    "name": name,
                    "type": device_type,
                    "channels": max_input,
                    "default_samplerate": int(default_rate),
                    "host_api": host_api_name,
                    "is_default": (i == default_id),
                })
            except Exception as e:
                print(f"[Audio] Error processing device {i}: {e}", flush=True)
                continue
    except Exception as e:
        print(f"[Audio] Error listing devices: {e}", flush=True)

    return devices


# ── 全局设备配置 ─────────────────────────────────────────

_DEVICE_ID: Optional[int] = None
_DEVICE_NAME: Optional[str] = None
_SAMPLE_RATE: int = 48000  # Intel SST 通常需要 48kHz
_device_lock = threading.Lock()


def init_audio(device_id: Optional[int] = None, device_name: Optional[str] = None) -> tuple[Optional[int], int]:
    """初始化音频设备配置

    Args:
        device_id: 设备索引号
        device_name: 设备名称（模糊匹配）

    Returns:
        (device_id, sample_rate)
    """
    global _DEVICE_ID, _DEVICE_NAME, _SAMPLE_RATE

    with _device_lock:
        _DEVICE_NAME = device_name

        if device_name:
            # 按名称查找
            all_devices = sd.query_devices()
            for i, dev in enumerate(all_devices):
                max_input = _get_device_attr(dev, "max_input_channels")
                name = str(_get_device_attr(dev, "name") or "")
                if max_input and max_input > 0 and device_name.lower() in name.lower():
                    _DEVICE_ID = i
                    default_rate = _get_device_attr(dev, "default_samplerate") or 48000
                    _SAMPLE_RATE = int(default_rate)
                    print(f"[Audio] Selected device by name: [{i}] {name} @ {_SAMPLE_RATE}Hz", flush=True)
                    return _DEVICE_ID, _SAMPLE_RATE

            print(f"[Audio] Device not found by name: {device_name}", flush=True)

        if device_id is not None:
            _DEVICE_ID = device_id

        # 获取设备默认采样率
        if _DEVICE_ID is not None:
            try:
                dev_info = sd.query_devices(_DEVICE_ID)
                _SAMPLE_RATE = int(_get_device_attr(dev_info, "default_samplerate") or 48000)
            except Exception:
                pass

        return _DEVICE_ID, _SAMPLE_RATE


def get_device_config() -> tuple[Optional[int], int]:
    with _device_lock:
        return _DEVICE_ID, _SAMPLE_RATE


def _auto_select_device() -> Optional[int]:
    """自动选择最佳输入设备：立体声混音 > CABLE > 默认 > 第一个可用"""
    try:
        all_devices = sd.query_devices()
    except Exception:
        return None

    # 优先立体声混音（捕获系统内部音频）
    for i, dev in enumerate(all_devices):
        name = str(_get_device_attr(dev, "name") or "").lower()
        max_input = _get_device_attr(dev, "max_input_channels")
        if max_input and max_input > 0:
            if any(kw in name for kw in ["立体声混音", "stereo mix", "what u hear",
                                          "wave out", "loopback", "音频输出"]):
                return i

    # 其次 VB-Audio CABLE（音频隔离）
    for i, dev in enumerate(all_devices):
        name = str(_get_device_attr(dev, "name") or "").lower()
        max_input = _get_device_attr(dev, "max_input_channels")
        if max_input and max_input > 0:
            if any(kw in name for kw in ["cable output", "vb-audio"]):
                return i

    # 其次默认输入
    try:
        default_id = sd.default.device[0]
        if default_id is not None:
            return int(default_id)
    except Exception:
        pass

    # 最后第一个输入设备
    for i, dev in enumerate(all_devices):
        max_input = _get_device_attr(dev, "max_input_channels")
        if max_input and max_input > 0:
            return i
    return None


# 模块加载时自动初始化
init_audio(_auto_select_device())

# ── 音频电平回调（用于 VU 表） ──────────────────────────────

_audio_level_callback: Optional[Callable[[float], None]] = None

def set_audio_level_callback(callback: Optional[Callable[[float], None]]):
    """设置音频电平回调"""
    global _audio_level_callback
    _audio_level_callback = callback


# ── 音频处理工具 ─────────────────────────────────────────

def _resample(audio: np.ndarray, orig_rate: int, target_rate: int = TARGET_RATE) -> np.ndarray:
    """重采样到目标采样率"""
    if orig_rate == target_rate or audio.size == 0:
        return audio
    target_len = int(len(audio) * target_rate / orig_rate)
    if target_len < 1:
        return audio
    return np.interp(
        np.linspace(0, len(audio) - 1, target_len),
        np.arange(len(audio)),
        audio,
    ).astype(np.float32)


def _normalize_audio(audio: np.ndarray, target_rms: float = 0.05) -> np.ndarray:
    """音频归一化提升低音量录音"""
    if audio.size == 0:
        return audio

    rms = np.sqrt(np.mean(audio ** 2) + 1e-10)
    if rms < 0.0001:
        return audio

    gain = target_rms / rms
    gain = min(max(gain, 0.3), 5.0)
    return np.clip(audio * gain, -1.0, 1.0)


# ── 录音器 ─────────────────────────────────────────────

class AudioRecorder:
    """线程安全的录音器（回调模式，兼容 WDM-KS）"""

    def __init__(self):
        self._frames: deque = deque()
        self._lock = threading.Lock()
        self._recording = False
        self._sample_rate: int = _SAMPLE_RATE
        self._streaming_callback: Optional[Callable] = None
        self._streaming_interval = 0.5
        self._last_streaming_time = 0.0
        self._stream = None
        self._frame_counter = 0
        self._stream_active = False

        self._open_stream()

    def _audio_callback(self, indata, frames, time_info, status):
        """PortAudio 回调 — 在音频线程中调用，必须轻量"""
        if status:
            print(f"[Audio] Callback status: {status}", flush=True)

        if indata is None or indata.size == 0:
            return

        # 取单声道
        audio = indata[:, 0].copy() if indata.ndim > 1 else indata.copy()

        # RMS 电平（VU 表）
        try:
            rms = float(np.sqrt(np.mean(audio ** 2)))
            if np.isnan(rms) or np.isinf(rms):
                rms = 0.0
        except Exception:
            rms = 0.0
        if _audio_level_callback:
            _audio_level_callback(rms)

        # 录音帧存储
        with self._lock:
            if self._recording:
                self._frames.append(audio)
                self._frame_counter += 1

                # 流式回调（异步）
                now = time.time()
                if (self._streaming_callback and
                        now - self._last_streaming_time >= self._streaming_interval):
                    self._last_streaming_time = now
                    recent = list(self._frames)[-30:]
                    if recent:
                        chunk = np.concatenate(recent, axis=0).flatten()
                        threading.Thread(
                            target=self._streaming_callback,
                            args=(chunk,),
                            daemon=True,
                        ).start()

    def _open_stream(self):
        """打开音频流（带重试）"""
        max_retries = 3
        for attempt in range(max_retries):
            try:
                print(f"[Audio] Opening stream: device={_DEVICE_ID}, rate={self._sample_rate} (attempt {attempt + 1})", flush=True)
                blocksize = int(self._sample_rate * BLOCKSIZE_MS / 1000)
                blocksize = max(blocksize, 512)

                self._stream = sd.InputStream(
                    samplerate=self._sample_rate,
                    channels=CHANNELS,
                    dtype='float32',
                    device=_DEVICE_ID,
                    blocksize=blocksize,
                    callback=self._audio_callback,
                )
                self._stream.start()
                self._stream_active = True
                print(f"[Audio] Stream opened (callback mode, blocksize={blocksize}), device={_DEVICE_ID}", flush=True)
                return

            except Exception as e:
                print(f"[Audio] Stream open failed (attempt {attempt + 1}): {e}", flush=True)
                time.sleep(0.5)
                self._stream_active = False

        print(f"[Audio] Failed to open stream after {max_retries} attempts", flush=True)

    @property
    def device_id(self) -> Optional[int]:
        return _DEVICE_ID

    @property
    def sample_rate(self) -> int:
        return self._sample_rate

    def reopen(self, device_id: Optional[int] = None) -> bool:
        """重新打开音频流（设备切换时）"""
        print(f"[Audio] Reopening stream for device {device_id}", flush=True)

        if self._stream:
            try:
                self._stream.stop()
                self._stream.close()
            except Exception:
                pass
            self._stream = None

        with _device_lock:
            if device_id is not None:
                _DEVICE_ID = device_id
            self._sample_rate = _SAMPLE_RATE

        self._open_stream()
        return self._stream is not None and self._stream_active

    def set_streaming_callback(self, callback: Optional[Callable]):
        """设置流式回调"""
        self._streaming_callback = callback

    def start_recording(self):
        """开始录音"""
        with self._lock:
            self._frames.clear()
            self._last_streaming_time = time.time()
            self._recording = True
        print(f"[Audio] Recording started (device={_DEVICE_ID}, rate={self._sample_rate})", flush=True)

    def stop_recording(self) -> np.ndarray:
        """停止录音，返回增强后的音频"""
        self._recording = False
        self._streaming_callback = None

        audio_frames = []
        with self._lock:
            audio_frames = list(self._frames)
            self._frames.clear()

        if not audio_frames:
            print("[Audio] No frames captured", flush=True)
            return np.zeros(0, dtype=np.float32)

        raw = np.concatenate(audio_frames, axis=0).flatten()
        print(f"[Audio] Recording stopped: {len(raw)} samples, {len(audio_frames)} frames", flush=True)

        # 归一化音频（提升低音量录音识别率）
        normalized = _normalize_audio(raw)

        # 重采样到 16kHz
        return _resample(normalized, self._sample_rate)

    def record_until_silence(self, max_seconds=30, silence_threshold=0.003, silence_duration=2.0) -> np.ndarray:
        """录音直到检测到静音"""
        self.start_recording()

        chunks_per_second = 10
        silence_chunks_needed = int(silence_duration * chunks_per_second)
        max_chunks = int(max_seconds * chunks_per_second)
        silence_chunks = 0
        min_chunks = int(0.5 * chunks_per_second)

        for i in range(max_chunks):
            time.sleep(1.0 / chunks_per_second)

            with self._lock:
                if not self._frames:
                    continue
                recent = list(self._frames)[-3:]
                if recent:
                    rms = float(np.sqrt(np.mean(np.concatenate(recent) ** 2)))
                else:
                    rms = 0

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
        """关闭录音器"""
        self._recording = False
        self._stream_active = False

        if self._stream:
            try:
                self._stream.stop()
                self._stream.close()
            except Exception:
                pass
            self._stream = None

        print(f"[Audio] Recorder closed (processed {self._frame_counter} frames)", flush=True)
