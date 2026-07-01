"""Qwen (千问) STT service using DashScope Paraformer API.

全面重写：
- 增强的音频预处理（去噪、归一化、重采样）
- 自动去重（防止短音频片段产生重复文本）
- 改进的流式处理
- 模型自动切换和降级
- 详细的调试日志
"""

import io
import json
import os
import tempfile
import time
import threading
import re
from collections import deque
from typing import Optional, Callable

import numpy as np
import soundfile as sf

import dashscope
from dashscope.audio.asr import Recognition, RecognitionCallback, RecognitionResult
import zhconv


# ── 文本去重 ─────────────────────────────────────────────

class TextDeduplicator:
    """文本去重器 — 避免短音频片段产生重复输出"""

    def __init__(self, max_history: int = 10):
        self._history: deque = deque(maxlen=max_history)
        self._last_text: str = ""

    def is_duplicate(self, text: str) -> bool:
        """判断文本是否与最近的输出重复"""
        if not text:
            return True
        text = text.strip()

        # 与上一条完全一致
        if text == self._last_text:
            return True

        # 上一条包含本条（增量重复）
        if self._last_text and text in self._last_text:
            return True

        # 本条被上一条包含（增量重复的不同方向）
        if self._last_text and self._last_text in text:
            return True  # 可能是累积重复

        # 检查历史中的部分匹配
        text_chars = set(text)
        for prev in self._history:
            if prev:
                prev_chars = set(prev)
                overlap = len(text_chars & prev_chars)
                if len(text_chars) > 0 and overlap / len(text_chars) > 0.85:
                    return True

        return False

    def add(self, text: str):
        """添加文本到历史"""
        text = text.strip()
        if text:
            self._history.append(text)
            self._last_text = text

    def reset(self):
        """重置历史"""
        self._history.clear()
        self._last_text = ""


# ── 音频预处理 ─────────────────────────────────────────────

def preprocess_audio(
    audio_data: np.ndarray,
    sample_rate: int,
    target_rate: int = 16000,
    noise_reduction: bool = True,
) -> np.ndarray:
    """
    增强的音频预处理管线

    步骤：
    1. 确保 float32 格式
    2. 重采样到目标采样率 (16kHz)
    3. RMS 归一化到目标电平
    4. 噪声门控
    5. 削波保护
    """
    if audio_data is None or len(audio_data) == 0:
        return audio_data

    # ── 步骤 1: 确保 float32 ──
    audio = audio_data.astype(np.float32) if audio_data.dtype != np.float32 else audio_data.copy()
    if np.abs(audio).max() > 1.0:
        audio = audio / 32767.0

    # ── 步骤 2: 重采样 ──
    if sample_rate != target_rate and sample_rate > 0:
        target_len = int(len(audio) * target_rate / sample_rate)
        audio = np.interp(
            np.linspace(0, len(audio) - 1, target_len),
            np.arange(len(audio)),
            audio,
        ).astype(np.float32)
        sample_rate = target_rate

    # ── 步骤 3: RMS 归一化 ──
    rms = np.sqrt(np.mean(audio ** 2) + 1e-10)
    if rms > 0.0001:
        target_rms = 0.08  # -22dBFS，略高于默认，提升弱音频
        gain = target_rms / rms
        gain = min(max(gain, 0.5), 6.0)
        audio = audio * gain
    else:
        # 几乎静音
        return audio * 0.1

    # ── 步骤 4: 噪声门控 ──
    if noise_reduction:
        frame_size = int(sample_rate * 0.03)  # 30ms 帧
        if frame_size > 0:
            for start in range(0, len(audio), frame_size):
                end = min(start + frame_size, len(audio))
                frame = audio[start:end]
                frame_rms = np.sqrt(np.mean(frame ** 2) + 1e-10)
                if frame_rms < 0.005:
                    gain = (frame_rms / 0.005) ** 2
                    audio[start:end] = frame * gain * 0.5

    # ── 步骤 5: 削波保护 ──
    return np.clip(audio, -1.0, 1.0)


def restore_punctuation(text: str) -> str:
    """智能标点恢复"""
    if not text:
        return text
    text = text.strip()

    # 如果已正确标点，不动
    if text and text[-1] in "。！？，、；：.!?,":
        return text

    # 句末加句号
    if text:
        text += "。"

    return text


# ── DashScope 回调 ────────────────────────────────────────

class QwenSTTCallback(RecognitionCallback):
    """DashScope Recognition 回调"""

    def __init__(self):
        self.results = []
        self.error = None
        self._event = threading.Event()
        self._deduplicator = TextDeduplicator()

    def on_event(self, result: RecognitionResult):
        try:
            sentence = result.get_sentence()
            if sentence:
                # 提取句子文本
                if isinstance(sentence, dict):
                    text = sentence.get('text', '')
                else:
                    text = str(sentence)
                if text and text.strip():
                    self.results.append(text.strip())
                return

            if hasattr(result, 'output') and result.output:
                output = result.output
                if isinstance(output, dict):
                    sentences = output.get('sentence', [])
                    for s in sentences:
                        if isinstance(s, dict):
                            text = s.get('text', '')
                            if text and text.strip():
                                # 去重
                                if not self._deduplicator.is_duplicate(text.strip()):
                                    self.results.append(text.strip())
                                    self._deduplicator.add(text.strip())
        except Exception as e:
            print(f"[QwenSTT] on_event error: {e}", flush=True)

    def on_complete(self):
        print(f"[QwenSTT] Recognition complete, got {len(self.results)} results", flush=True)
        self._event.set()

    def on_error(self, result):
        error_msg = str(result)
        print(f"[QwenSTT] Recognition error: {error_msg}", flush=True)
        self.error = error_msg
        self._event.set()

    def on_close(self):
        print("[QwenSTT] Recognition closed", flush=True)
        self._event.set()


class QwenSTT:
    """千问语音识别服务（DashScope Paraformer API）"""

    SUPPORTED_MODELS = [
        "paraformer-realtime-v2",
        "paraformer-v2",
    ]

    def __init__(self, api_key: str, language: str = "zh"):
        self.api_key = api_key
        dashscope.api_key = api_key
        self.model = self.SUPPORTED_MODELS[0]
        self.language = language
        self._failed_models = set()
        self._deduplicator = TextDeduplicator()

        # 流式识别状态
        self._is_streaming = False
        self._stream_thread: Optional[threading.Thread] = None
        self._audio_buffer: list[np.ndarray] = []
        self._buffer_lock = threading.Lock()
        self._callback: Optional[Callable] = None
        self._stop_event = threading.Event()

    def _get_next_model(self) -> str:
        for model in self.SUPPORTED_MODELS:
            if model not in self._failed_models:
                return model
        self._failed_models.clear()
        return self.SUPPORTED_MODELS[0]

    def _mark_model_failed(self, model: str):
        self._failed_models.add(model)
        print(f"[QwenSTT] ⚠️ 模型 {model} 标记失败，切换中...", flush=True)
        self.model = self._get_next_model()
        print(f"[QwenSTT] 🔄 切换到模型: {self.model}", flush=True)

    def recognize(self, audio_data: np.ndarray, sample_rate: int = 16000) -> str:
        """
        语音识别（含增强预处理 + 自动去重 + 标点恢复）

        Args:
            audio_data: 音频 numpy 数组
            sample_rate: 原始采样率

        Returns:
            识别文本
        """
        try:
            # ── 1. 预处理 ──
            audio = preprocess_audio(audio_data, sample_rate)

            # 静音检测
            rms = np.sqrt(np.mean(audio ** 2) + 1e-10)
            if len(audio) < 1600 or rms < 0.002:
                print(f"[QwenSTT] ⏭️ 音频过短 ({len(audio)} samples) 或静音 (RMS={rms:.6f})", flush=True)
                return ""

            # ── 2. 保存临时 WAV ──
            with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as f:
                sf.write(f.name, audio, 16000)
                tmp_path = f.name

            try:
                # ── 3. 调用 API ──
                cb = QwenSTTCallback()

                rec_params = {
                    "model": self.model,
                    "callback": cb,
                    "format": "wav",
                    "sample_rate": 16000,
                }

                # 语言提示
                if self.language and self.language != "auto":
                    rec_params["language_hints"] = [self.language]
                else:
                    rec_params["language_hints"] = ["zh", "en"]

                # 标点符号
                rec_params["enable_punctuation"] = True

                rec = Recognition(**rec_params)
                result = rec.call(tmp_path)

                # ── 4. 提取结果 ──
                text = ""
                if result.status_code == 200 and cb.results:
                    # 去重合并
                    seen = set()
                    unique_results = []
                    for r in cb.results:
                        r_clean = r.strip()
                        if r_clean and r_clean not in seen:
                            seen.add(r_clean)
                            unique_results.append(r_clean)
                    text = "".join(unique_results)

                # fallback: 从 output.sentence 提取
                if not text and result.status_code == 200 and result.output:
                    output = result.output
                    if isinstance(output, dict):
                        sentences = output.get('sentence', [])
                        texts = []
                        for s in sentences:
                            if isinstance(s, dict):
                                t = s.get('text', '')
                                if t and t.strip():
                                    texts.append(t.strip())
                        if texts:
                            text = "".join(texts)

                # 模型不支持的错误码 44 → 切换模型重试
                if result.status_code == 44:
                    self._mark_model_failed(self.model)
                    return self.recognize(audio_data, sample_rate)

                if text:
                    # 简繁转换
                    text = zhconv.convert(text, "zh-hans")
                    text = restore_punctuation(text)
                    text = self._clean_text(text)
                    print(f"[QwenSTT] ✅ 识别成功 ({len(text)} chars): [{text[:80]}...]", flush=True)
                else:
                    print(f"[QwenSTT] ⚠️ 无结果, status={result.status_code}", flush=True)

                return text

            finally:
                try:
                    os.unlink(tmp_path)
                except Exception:
                    pass

        except Exception as e:
            print(f"[QwenSTT] ❌ 识别异常: {e}", flush=True)

        return ""

    def _clean_text(self, text: str) -> str:
        """清理识别文本中的常见噪声"""
        if not text:
            return text

        # 移除纯标点结果
        text = text.strip()
        if all(c in "。！？，、；：.!?,\"' " for c in text):
            return ""

        # 移除重复的连续标点
        text = re.sub(r'([。！？，、；：])\1+', r'\1', text)

        # 移除过长静音标记
        text = re.sub(r'[\s_]{3,}', ' ', text)

        return text.strip()

    # ── 流式接口 ─────────────────────────────────────────────

    def start_streaming(self, callback: Callable, sample_rate: int = 16000):
        """开始流式识别"""
        if self._is_streaming:
            return

        self._is_streaming = True
        self._callback = callback
        self._audio_buffer = []
        self._stop_event.clear()
        self._deduplicator.reset()

        self._stream_thread = threading.Thread(
            target=self._streaming_worker,
            args=(sample_rate,),
            daemon=True,
        )
        self._stream_thread.start()
        print("[QwenSTT] ▶️ Streaming started", flush=True)

    def stop_streaming(self) -> str:
        """停止流式识别，返回累积的最终结果"""
        if not self._is_streaming:
            return ""

        self._is_streaming = False
        self._stop_event.set()

        if self._stream_thread:
            self._stream_thread.join(timeout=5.0)

        with self._buffer_lock:
            if self._audio_buffer:
                try:
                    audio_data = np.concatenate(self._audio_buffer, axis=0).flatten()
                    self._audio_buffer = []
                    text = self.recognize(audio_data)
                    print(f"[QwenSTT] ⏹️ Streaming final result ({len(text)} chars)", flush=True)
                    return text
                except Exception as e:
                    print(f"[QwenSTT] ⏹️ Streaming final error: {e}", flush=True)

        return ""

    def add_audio_chunk(self, audio_chunk: np.ndarray):
        """添加音频块到缓冲区（线程安全）"""
        if not self._is_streaming:
            return
        with self._buffer_lock:
            # 确保是一维
            chunk = audio_chunk.flatten() if audio_chunk.ndim > 1 else audio_chunk
            self._audio_buffer.append(chunk)

    def _streaming_worker(self, sample_rate: int):
        """流式处理线程 — 定期识别累积的音频"""
        last_process_time = time.time()
        process_interval = 0.8  # 每 800ms 处理一次（给足够音频提升准确率）

        while self._is_streaming and not self._stop_event.is_set():
            current_time = time.time()
            if current_time - last_process_time >= process_interval:
                with self._buffer_lock:
                    if self._audio_buffer:
                        audio_data = np.concatenate(self._audio_buffer, axis=0).flatten()
                        # 保留最后一个块用于增量（防止丢失边界）
                        buffer_copy = list(self._audio_buffer)
                        self._audio_buffer = []

                try:
                    text = self.recognize(audio_data, sample_rate)
                    if text and self._callback:
                        self._callback(text)
                except Exception as e:
                    print(f"[QwenSTT] Streaming process error: {e}", flush=True)

                last_process_time = current_time
            else:
                time.sleep(0.1)

        print("[QwenSTT] Streaming worker stopped", flush=True)
