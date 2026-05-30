"""Speech-to-text transcription — ported from Murmur's transcribe.py."""

import glob
import os
import site
import sys

import numpy as np
import zhconv


def _add_cuda_dll_dirs() -> None:
    if sys.platform != "win32":
        return
    dirs: list[str] = []
    for site_dir in site.getsitepackages():
        for sub in ("nvidia/cublas/bin", "nvidia/cudnn/bin", "nvidia/cuda_runtime/bin"):
            dll_dir = os.path.join(site_dir, sub.replace("/", os.sep))
            if os.path.isdir(dll_dir):
                dirs.append(dll_dir)
    cuda_path = os.environ.get("CUDA_PATH") or os.environ.get("CUDA_HOME")
    if cuda_path:
        bin_dir = os.path.join(cuda_path, "bin")
        if os.path.isdir(bin_dir):
            dirs.append(bin_dir)
    for pattern in [r"C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v*\bin", r"C:\CUDA\v*\bin"]:
        for path in sorted(glob.glob(pattern), reverse=True):
            if os.path.isdir(path):
                dirs.append(path)
    if dirs:
        os.environ["PATH"] = os.pathsep.join(dirs) + os.pathsep + os.environ.get("PATH", "")


_add_cuda_dll_dirs()

try:
    import onnxruntime as _ort
    _ort.set_default_logger_severity(3)
except Exception:
    pass

from faster_whisper import WhisperModel  # noqa: E402


def detect_device() -> str:
    try:
        import ctranslate2
        if ctranslate2.get_cuda_device_count() > 0:
            return "cuda"
    except Exception:
        pass
    return "cpu"


class Transcriber:
    def __init__(self, model_name: str = "base", device: str = "auto", compute_type: str = "auto", language: str = "zh"):
        self._model_name = model_name
        self._language = language if language != "auto" else ""

        if device == "auto":
            self.device = detect_device()
        else:
            self.device = device

        if compute_type == "auto":
            self.compute_type = "float16" if self.device == "cuda" else "int8_float32"
        else:
            self.compute_type = compute_type

        try:
            self._model = WhisperModel(model_name, device=self.device, compute_type=self.compute_type)
        except ValueError as exc:
            if "compute type" in str(exc).lower() and self.compute_type == "float16":
                self.compute_type = "int8_float32"
                self._model = WhisperModel(model_name, device=self.device, compute_type=self.compute_type)
            else:
                raise

    def switch_to_cpu(self):
        self._model = WhisperModel(self._model_name, device="cpu")
        self.device = "cpu"

    def transcribe(self, audio: np.ndarray) -> str:
        if audio.size == 0:
            return ""
        ALLOWED_LANGS = {"zh", "en"}
        segments, info = self._model.transcribe(
            audio,
            language=self._language or None,
            vad_filter=True,
            vad_parameters=dict(
                min_silence_duration_ms=500,  # 最小静音持续时间（毫秒）
                speech_pad_ms=200,  # 语音填充时间（毫秒）
            ),
            beam_size=5,  # 束搜索大小
            best_of=5,  # 最佳候选数量
            temperature=0.0,  # 温度参数（0.0 表示确定性输出）
            compression_ratio_threshold=2.4,  # 压缩率阈值
            log_prob_threshold=-1.0,  # 对数概率阈值
            no_speech_threshold=0.6,  # 无语音阈值
            condition_on_previous_text=True,  # 基于前文条件化
        )
        detected = info.language if info else None
        if detected and detected not in ALLOWED_LANGS:
            return ""
        text = " ".join(seg.text for seg in segments).strip()
        if detected == "zh":
            text = zhconv.convert(text, "zh-hans")
        return text
