"""Audio capture module — ported from Murmur's audio.py for sidecar use."""

import queue
import sys
import threading
import time

import numpy as np
import sounddevice as sd

WHISPER_RATE = 16000
CHANNELS = 1
DTYPE = "float32"


def _find_working_config() -> tuple[int | None, int]:
    candidates: list[tuple[int | None, int]] = []
    if sys.platform == "win32":
        try:
            for api in sd.query_hostapis():
                if "WASAPI" in api["name"] and api["default_input_device"] >= 0:
                    dev = api["default_input_device"]
                    try:
                        native = int(sd.query_devices(dev, "input")["default_samplerate"])
                    except Exception:
                        native = 48000
                    candidates.append((dev, native))
        except Exception:
            pass
    for rate in (48000, 44100, 16000):
        candidates.append((None, rate))

    def _noop(indata, frames, time, status):
        pass

    for device, rate in candidates:
        stream = None
        try:
            stream = sd.InputStream(samplerate=rate, channels=CHANNELS, dtype=DTYPE, device=device, callback=_noop)
            stream.start()
            stream.stop()
            stream.close()
            return device, rate
        except Exception:
            if stream is not None:
                try:
                    stream.close()
                except Exception:
                    pass
    return None, WHISPER_RATE


def _resample(audio: np.ndarray, orig_rate: int) -> np.ndarray:
    if orig_rate == WHISPER_RATE or audio.size == 0:
        return audio
    target_len = int(len(audio) * WHISPER_RATE / orig_rate)
    return np.interp(np.linspace(0, len(audio) - 1, target_len), np.arange(len(audio)), audio).astype(np.float32)


_DEVICE, _RATE = _find_working_config()


class AudioRecorder:
    def __init__(self):
        self._frames: list[np.ndarray] = []
        self._lock = threading.Lock()
        self._recording = False
        self._streams: list[sd.InputStream] = []

        try:
            stream = sd.InputStream(
                samplerate=_RATE, channels=CHANNELS, dtype=DTYPE,
                device=_DEVICE, callback=self._callback,
            )
            stream.start()
            self._streams.append(stream)
        except Exception as e:
            print(f"[Audio] Failed to open input stream: {e}")

    def _callback(self, indata, frames, time_info, status):
        with self._lock:
            if self._recording:
                self._frames.append(indata.copy())

    def start_recording(self):
        with self._lock:
            self._frames = []
        self._recording = True

    def stop_recording(self) -> np.ndarray:
        self._recording = False
        with self._lock:
            if not self._frames:
                return np.zeros(0, dtype=np.float32)
            raw = np.concatenate(self._frames, axis=0).flatten()
        return _resample(raw, _RATE)

    def record_until_silence(self, max_seconds=30, silence_threshold=0.01, silence_duration=1.5) -> np.ndarray:
        self._recording = True
        with self._lock:
            self._frames = []

        chunks_per_second = 10
        silence_chunks_needed = int(silence_duration * chunks_per_second)
        max_chunks = int(max_seconds * chunks_per_second)
        silence_chunks = 0

        for _ in range(max_chunks):
            time.sleep(1 / chunks_per_second)
            with self._lock:
                if not self._frames:
                    continue
                rms = float(np.sqrt(np.mean(self._frames[-1] ** 2)))
            if rms < silence_threshold:
                silence_chunks += 1
            else:
                silence_chunks = 0
            if silence_chunks >= silence_chunks_needed:
                break

        self._recording = False
        with self._lock:
            if not self._frames:
                return np.zeros(0, dtype=np.float32)
            raw = np.concatenate(self._frames, axis=0).flatten()
        return _resample(raw, _RATE)

    def close(self):
        self._recording = False
        for s in self._streams:
            try:
                s.stop()
                s.close()
            except Exception:
                pass
