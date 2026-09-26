"""Thread-safe audio capture with Windows loopback and legacy input fallback."""

import os
import queue
import sys
import threading
import time
from collections import deque
from typing import Any, Callable, Optional

import numpy as np
import sounddevice as sd

try:
    import soundcard as sc
except ImportError:  # Optional at runtime so legacy installs can still start.
    sc = None

TARGET_RATE = 16000
BLOCKSIZE_MS = 50
STREAMING_INTERVAL = 0.5
STREAM_QUEUE_SIZE = 8


def _log(message: str, flush: bool = True):
    # stdout is a JSON protocol for audio_capture.py; diagnostics belong on stderr.
    print(message, file=sys.stderr, flush=flush)


def _get_device_attr(dev, attr):
    if isinstance(dev, dict):
        return dev.get(attr)
    return getattr(dev, attr, None)


def _soundcard_id(device) -> str:
    return str(getattr(device, "id", "") or "")


def _soundcard_name(device) -> str:
    return str(getattr(device, "name", "") or "")


def _find_loopback_microphone(speaker):
    if speaker is None or sc is None:
        return None
    speaker_id = _soundcard_id(speaker)
    try:
        return sc.get_microphone(speaker_id, include_loopback=True)
    except Exception:
        pass

    speaker_name = _soundcard_name(speaker).lower()
    try:
        microphones = sc.all_microphones(include_loopback=True)
    except Exception:
        return None
    for microphone in microphones:
        mic_id = _soundcard_id(microphone)
        mic_name = _soundcard_name(microphone).lower()
        if speaker_id and speaker_id in mic_id:
            return microphone
        if speaker_name and speaker_name in mic_name and "loopback" in mic_name:
            return microphone
    return None


def _list_loopback_devices() -> list[dict[str, Any]]:
    if sc is None or sys.platform != "win32":
        return []
    comtypes_module = None
    try:
        try:
            import comtypes as comtypes_module
            comtypes_module.CoInitialize()
        except ImportError:
            pass

        default_speaker = sc.default_speaker()
        default_id = _soundcard_id(default_speaker)
        speakers = sc.all_speakers()
        devices = []
        for loopback_index, speaker in enumerate(speakers):
            microphone = _find_loopback_microphone(speaker)
            if microphone is None:
                continue
            speaker_id = _soundcard_id(speaker)
            name = _soundcard_name(speaker)
            devices.append({
                "id": -(loopback_index + 1),
                "index": -(loopback_index + 1),
                "name": name,
                "type": "system_audio",
                "channels": 2,
                "default_samplerate": 48000,
                "host_api": "Windows WASAPI",
                "is_default": bool(default_id and speaker_id == default_id),
                "recommended": True,
                "backend": "soundcard",
                "capture_mode": "loopback",
                "is_loopback": True,
                "speaker_id": speaker_id,
                "available": True,
            })
        return devices
    except Exception as exc:
        _log(f"[Audio] SoundCard device enumeration failed: {exc}", flush=True)
        return []
    finally:
        if comtypes_module is not None:
            try:
                comtypes_module.CoUninitialize()
            except Exception:
                pass


def _list_sounddevice_inputs(include_microphones: bool = True) -> list[dict[str, Any]]:
    devices = []
    try:
        default_id = sd.default.device[0]
        all_devices = sd.query_devices()
        host_apis = sd.query_hostapis()
    except Exception as exc:
        _log(f"[Audio] sounddevice enumeration failed: {exc}", flush=True)
        return devices

    for index, dev in enumerate(all_devices):
        max_input = _get_device_attr(dev, "max_input_channels") or 0
        if max_input <= 0:
            continue
        name = str(_get_device_attr(dev, "name") or "")
        lowered = name.lower()
        device_type = "mic"
        if any(term in lowered for term in (
            "立体声混音", "stereo mix", "what u hear", "wave out", "loopback", "音频输出"
        )):
            device_type = "stereo_mix"
        elif any(term in lowered for term in ("cable output", "vb-audio virtual cable", "vb-audio point")):
            device_type = "cable"
        elif not include_microphones:
            continue

        host_api_index = _get_device_attr(dev, "host_api")
        host_api_name = "unknown"
        if isinstance(host_api_index, int) and 0 <= host_api_index < len(host_apis):
            host_api_name = host_apis[host_api_index].get("name", "unknown")
        devices.append({
            "id": index,
            "index": index,
            "name": name,
            "type": device_type,
            "channels": int(max_input),
            "default_samplerate": int(_get_device_attr(dev, "default_samplerate") or 48000),
            "host_api": host_api_name,
            "is_default": index == default_id,
            "recommended": device_type in ("stereo_mix", "cable"),
            "backend": "sounddevice",
            "capture_mode": "input",
            "is_loopback": device_type == "stereo_mix",
            "available": True,
        })
    return devices


def list_input_devices() -> list[dict[str, Any]]:
    """Return loopback playback endpoints followed by legacy input devices."""
    return _list_loopback_devices() + _list_sounddevice_inputs()


_DEVICE_ID: Optional[int] = None
_DEVICE_NAME: Optional[str] = None
_SPEAKER_ID: Optional[str] = None
_BACKEND = os.environ.get("AUDIO_BACKEND", "auto").strip().lower()
_SAMPLE_RATE = 48000
_device_lock = threading.Lock()


def init_audio(
    device_id: Optional[int] = None,
    device_name: Optional[str] = None,
    backend: Optional[str] = None,
    speaker_id: Optional[str] = None,
) -> tuple[Optional[int], int]:
    global _DEVICE_ID, _DEVICE_NAME, _SPEAKER_ID, _BACKEND, _SAMPLE_RATE
    with _device_lock:
        if backend:
            _BACKEND = backend.strip().lower()
        _DEVICE_NAME = device_name
        _SPEAKER_ID = speaker_id
        if device_id is not None:
            _DEVICE_ID = device_id
        if _BACKEND == "sounddevice" or device_id is not None:
            if device_name and device_id is None:
                for device in _list_sounddevice_inputs():
                    if device_name.lower() in device["name"].lower():
                        _DEVICE_ID = int(device["id"])
                        break
            if _DEVICE_ID is not None:
                try:
                    info = sd.query_devices(_DEVICE_ID)
                    _SAMPLE_RATE = int(_get_device_attr(info, "default_samplerate") or 48000)
                except Exception:
                    pass
        return _DEVICE_ID, _SAMPLE_RATE


def get_device_config() -> tuple[Optional[int], int]:
    with _device_lock:
        return _DEVICE_ID, _SAMPLE_RATE


def _resample(audio: np.ndarray, original_rate: int, target_rate: int = TARGET_RATE) -> np.ndarray:
    if audio.size == 0 or original_rate == target_rate:
        return audio.astype(np.float32, copy=False)
    target_length = max(1, int(round(len(audio) * target_rate / original_rate)))
    return np.interp(
        np.linspace(0, len(audio) - 1, target_length),
        np.arange(len(audio)),
        audio,
    ).astype(np.float32)


def _canonicalize(audio: np.ndarray, sample_rate: int) -> np.ndarray:
    audio = np.asarray(audio)
    if audio.size == 0:
        return np.zeros(0, dtype=np.float32)
    if audio.ndim > 1:
        audio = np.mean(audio.astype(np.float32), axis=1)
    else:
        audio = audio.astype(np.float32, copy=False)
    if np.issubdtype(audio.dtype, np.integer) or np.max(np.abs(audio), initial=0.0) > 1.5:
        audio = audio / 32768.0
    audio = np.nan_to_num(audio, nan=0.0, posinf=0.0, neginf=0.0)
    return _resample(np.clip(audio, -1.0, 1.0), sample_rate)


def _normalize_audio(audio: np.ndarray, target_rms: float = 0.04) -> np.ndarray:
    if audio.size == 0:
        return audio
    rms = float(np.sqrt(np.mean(audio ** 2) + 1e-10))
    if rms < 0.0001:
        return audio
    gain = min(max(target_rms / rms, 0.5), 3.0)
    return np.clip(audio * gain, -1.0, 1.0)


_audio_level_callback: Optional[Callable[[float], None]] = None


def set_audio_level_callback(callback: Optional[Callable[[float], None]]):
    global _audio_level_callback
    _audio_level_callback = callback


class AudioRecorder:
    """Captures canonical mono 16 kHz audio without changing the playback route."""

    def __init__(self):
        self._frames: deque[np.ndarray] = deque()
        self._lock = threading.Lock()
        self._lifecycle_lock = threading.RLock()
        self._recording = False
        self._session_generation = 0
        self._streaming_callback: Optional[Callable[[np.ndarray], None]] = None
        self._stream_queue: queue.Queue[Optional[np.ndarray]] = queue.Queue(STREAM_QUEUE_SIZE)
        self._stream_pending: deque[np.ndarray] = deque()
        self._stream_pending_samples = 0
        self._stream_target_samples = int(TARGET_RATE * STREAMING_INTERVAL)
        self._capture_thread: Optional[threading.Thread] = None
        self._stream_thread: Optional[threading.Thread] = None
        self._capture_stop_event = threading.Event()
        self._legacy_stream = None
        self._source = None
        self._backend_name = "unavailable"
        self._source_name = ""
        self._native_sample_rate = 48000
        self._fallback_reason: Optional[str] = None
        self._dropped_chunks = 0
        self._open_source()

    @property
    def device_id(self) -> Optional[int]:
        return _DEVICE_ID

    @property
    def sample_rate(self) -> int:
        """Canonical STT output rate (kept for existing callers)."""
        return TARGET_RATE

    @property
    def native_sample_rate(self) -> int:
        return self._native_sample_rate

    @property
    def output_sample_rate(self) -> int:
        return TARGET_RATE

    def status(self) -> dict[str, Any]:
        with self._lifecycle_lock:
            return {
                "ready": self._source is not None or self._legacy_stream is not None,
                "backend": self._backend_name,
                "capture_mode": "loopback" if self._backend_name == "soundcard" else "input",
                "is_loopback": self._backend_name == "soundcard" or self._source_name.lower().find("stereo mix") >= 0,
                "source_name": self._source_name,
                "native_sample_rate": self._native_sample_rate,
                "output_sample_rate": TARGET_RATE,
                "fallback": self._fallback_reason is not None,
                "fallback_reason": self._fallback_reason,
                "dropped_chunks": self._dropped_chunks,
            }

    def _select_loopback(self):
        if sc is None or sys.platform != "win32":
            raise RuntimeError("SoundCard WASAPI loopback is unavailable")
        speakers = sc.all_speakers()
        selected = None
        if _SPEAKER_ID:
            selected = next((speaker for speaker in speakers if _soundcard_id(speaker) == _SPEAKER_ID), None)
        if selected is None and _DEVICE_NAME:
            lowered = _DEVICE_NAME.lower()
            selected = next((speaker for speaker in speakers if lowered in _soundcard_name(speaker).lower()), None)
        if selected is None:
            selected = sc.default_speaker()
        microphone = _find_loopback_microphone(selected)
        if microphone is None:
            raise RuntimeError(f"No loopback endpoint for {_soundcard_name(selected)}")
        return selected, microphone

    def _open_source(self):
        self.close()
        configured = _BACKEND or "auto"
        if configured in ("auto", "soundcard", "soundcard_loopback"):
            try:
                speaker, microphone = self._select_loopback()
                self._source = microphone
                self._source_name = _soundcard_name(speaker)
                self._backend_name = "soundcard"
                self._native_sample_rate = 48000
                _log(f"[Audio] Using WASAPI loopback: {self._source_name}", flush=True)
                return
            except Exception as exc:
                if configured in ("soundcard", "soundcard_loopback"):
                    raise
                self._fallback_reason = str(exc)
                _log(f"[Audio] Loopback unavailable, trying legacy input: {exc}", flush=True)

        candidates = _list_sounddevice_inputs(include_microphones=False)
        selected = None
        if _DEVICE_ID is not None:
            selected = next((device for device in _list_sounddevice_inputs() if device["id"] == _DEVICE_ID), None)
        if selected is None and _DEVICE_NAME:
            selected = next((device for device in candidates if _DEVICE_NAME.lower() in device["name"].lower()), None)
        if selected is None:
            selected = next((device for device in candidates if device["type"] == "stereo_mix"), None)
        if selected is None:
            selected = next((device for device in candidates if device["type"] == "cable"), None)
        if selected is None:
            raise RuntimeError("No WASAPI loopback, Stereo Mix, or virtual cable is available")

        self._native_sample_rate = selected["default_samplerate"]
        self._source_name = selected["name"]
        self._backend_name = "sounddevice"
        self._legacy_stream = sd.InputStream(
            samplerate=self._native_sample_rate,
            channels=min(2, selected["channels"]),
            dtype="float32",
            device=selected["id"],
            blocksize=max(512, int(self._native_sample_rate * BLOCKSIZE_MS / 1000)),
            callback=self._legacy_callback,
        )
        self._legacy_stream.start()
        _log(f"[Audio] Using legacy input: {self._source_name}", flush=True)

    def _legacy_callback(self, data, frames, time_info, status):
        if status:
            _log(f"[Audio] Callback status: {status}", flush=True)
        self._handle_native_block(data, self._native_sample_rate)

    def _handle_native_block(self, data: np.ndarray, sample_rate: int):
        block = _canonicalize(data, sample_rate)
        if block.size == 0:
            return
        rms = float(np.sqrt(np.mean(block ** 2) + 1e-10))
        if _audio_level_callback:
            _audio_level_callback(rms if np.isfinite(rms) else 0.0)
        with self._lock:
            if not self._recording:
                return
            self._frames.append(block)
            self._stream_pending.append(block)
            self._stream_pending_samples += len(block)
            if self._streaming_callback and self._stream_pending_samples >= self._stream_target_samples:
                self._enqueue_pending_locked()

    def _enqueue_pending_locked(self):
        if not self._stream_pending:
            return
        chunk = np.concatenate(list(self._stream_pending)).astype(np.float32, copy=False)
        self._stream_pending.clear()
        self._stream_pending_samples = 0
        try:
            self._stream_queue.put_nowait(chunk)
        except queue.Full:
            self._dropped_chunks += 1
            _log("[Audio] Streaming queue full; dropping chunk", flush=True)

    def _capture_loopback(self, source, stop_event: threading.Event, generation: int):
        frames = max(256, int(self._native_sample_rate * BLOCKSIZE_MS / 1000))
        com_initialized = False
        try:
            if sys.platform == "win32":
                try:
                    import comtypes
                    comtypes.CoInitialize()
                    com_initialized = True
                except ImportError:
                    pass
            with source.recorder(samplerate=self._native_sample_rate, channels=2) as recorder:
                while not stop_event.is_set():
                    block = recorder.record(numframes=frames)
                    with self._lock:
                        current = self._recording and self._session_generation == generation
                    if current:
                        self._handle_native_block(block, self._native_sample_rate)
        except Exception as exc:
            self._fallback_reason = f"Loopback capture failed: {exc}"
            _log(f"[Audio] {self._fallback_reason}", flush=True)
            stop_event.set()
        finally:
            if com_initialized:
                try:
                    comtypes.CoUninitialize()
                except Exception:
                    pass

    def _stream_worker(self, stream_queue, callback):
        while True:
            chunk = stream_queue.get()
            if chunk is None:
                return
            try:
                callback(chunk)
            except Exception as exc:
                _log(f"[Audio] Streaming callback failed: {exc}", flush=True)

    def set_streaming_callback(self, callback: Optional[Callable[[np.ndarray], None]]):
        with self._lifecycle_lock:
            self._streaming_callback = callback

    def start_recording(self):
        with self._lifecycle_lock:
            if self._recording:
                raise RuntimeError("Audio recording is already active")
            if self._source is None and self._legacy_stream is None:
                self._open_source()
            if self._capture_thread and self._capture_thread.is_alive():
                raise RuntimeError("Previous capture session has not stopped")
            if self._stream_thread and self._stream_thread.is_alive():
                raise RuntimeError("Previous streaming session has not stopped")

            self._session_generation += 1
            generation = self._session_generation
            self._capture_stop_event = threading.Event()
            self._stream_queue = queue.Queue(STREAM_QUEUE_SIZE)
            callback = self._streaming_callback
            with self._lock:
                self._frames.clear()
                self._stream_pending.clear()
                self._stream_pending_samples = 0
                self._recording = True
            if callback:
                self._stream_thread = threading.Thread(
                    target=self._stream_worker,
                    args=(self._stream_queue, callback),
                    daemon=True,
                )
                self._stream_thread.start()
            if self._backend_name == "soundcard":
                self._capture_thread = threading.Thread(
                    target=self._capture_loopback,
                    args=(self._source, self._capture_stop_event, generation),
                    daemon=True,
                )
                self._capture_thread.start()
            _log(f"[Audio] Recording started ({self._backend_name}: {self._source_name})", flush=True)

    def stop_recording(self) -> np.ndarray:
        with self._lifecycle_lock:
            self._capture_stop_event.set()
            if self._capture_thread:
                self._capture_thread.join(timeout=2.0)
                if self._capture_thread.is_alive():
                    raise RuntimeError("Audio capture thread did not stop")
                self._capture_thread = None
            with self._lock:
                if self._streaming_callback:
                    self._enqueue_pending_locked()
                self._recording = False
                frames = list(self._frames)
                self._frames.clear()
            if self._stream_thread:
                try:
                    self._stream_queue.put_nowait(None)
                except queue.Full:
                    try:
                        self._stream_queue.get_nowait()
                        self._stream_queue.put_nowait(None)
                    except queue.Empty:
                        pass
                self._stream_thread.join(timeout=2.0)
                if self._stream_thread.is_alive():
                    raise RuntimeError("Audio streaming thread did not stop")
                self._stream_thread = None
            self._streaming_callback = None
            if not frames:
                return np.zeros(0, dtype=np.float32)
            return _normalize_audio(np.concatenate(frames).astype(np.float32, copy=False))

    def record_until_silence(self, max_seconds=30, silence_threshold=0.003, silence_duration=2.0) -> np.ndarray:
        self.start_recording()
        silent_since = None
        deadline = time.monotonic() + max_seconds
        while time.monotonic() < deadline:
            time.sleep(0.1)
            with self._lock:
                recent = list(self._frames)[-3:]
            rms = float(np.sqrt(np.mean(np.concatenate(recent) ** 2))) if recent else 0.0
            if rms < silence_threshold:
                silent_since = silent_since or time.monotonic()
                if time.monotonic() - silent_since >= silence_duration and recent:
                    break
            else:
                silent_since = None
        return self.stop_recording()

    def reopen(
        self,
        device_id: Optional[int] = None,
        device_name: Optional[str] = None,
        backend: Optional[str] = None,
        speaker_id: Optional[str] = None,
    ) -> bool:
        with self._lifecycle_lock:
            if self._recording:
                raise RuntimeError("Cannot change audio source while recording")
            init_audio(device_id, device_name, backend, speaker_id)
            self._fallback_reason = None
            self._open_source()
            return self.status()["ready"]

    def close(self):
        with self._lifecycle_lock:
            if self._recording:
                try:
                    self.stop_recording()
                except Exception as exc:
                    _log(f"[Audio] Stop during close failed: {exc}", flush=True)
            self._session_generation += 1
            self._capture_stop_event.set()
            if self._capture_thread and self._capture_thread.is_alive():
                self._capture_thread.join(timeout=2.0)
            self._capture_thread = None
            if self._stream_thread and self._stream_thread.is_alive():
                try:
                    self._stream_queue.put_nowait(None)
                except queue.Full:
                    try:
                        self._stream_queue.get_nowait()
                        self._stream_queue.put_nowait(None)
                    except queue.Empty:
                        pass
                self._stream_thread.join(timeout=2.0)
            self._stream_thread = None
            self._recording = False
            self._streaming_callback = None
            if self._legacy_stream is not None:
                try:
                    self._legacy_stream.stop()
                    self._legacy_stream.close()
                except Exception:
                    pass
            self._legacy_stream = None
            self._source = None
