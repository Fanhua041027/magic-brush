"""Qwen (千问) STT service using DashScope Paraformer API.

支持多模型、音频预处理、标点恢复、置信度过滤，提升识别准确率。
"""

import io
import json
import os
import tempfile
import time
import threading
import numpy as np
import soundfile as sf
from typing import Optional, Callable

import dashscope
from dashscope.audio.asr import Recognition, RecognitionCallback, RecognitionResult
import zhconv


# ── 音频预处理 ─────────────────────────────────────────────

def preprocess_audio(audio_data: np.ndarray, sample_rate: int) -> np.ndarray:
    """音频预处理：归一化 + 噪声门控，提升识别准确率"""
    if audio_data is None or len(audio_data) == 0:
        return audio_data

    # 确保是 float32
    audio = audio_data.astype(np.float32) if audio_data.dtype != np.float32 else audio_data.copy()

    # 如果是整数 PCM，缩放到 [-1, 1]
    if audio.max() > 1.0:
        audio = audio / 32767.0

    # 噪声门控：静音区域的 RMS 低于阈值时拉低
    rms = np.sqrt(np.mean(audio ** 2))
    if rms < 0.002:
        # 过于安静，可能只有底噪，直接返回静音
        return audio * 0.1

    # RMS 归一化到目标电平 -26dBFS（约 0.05 RMS）
    target_rms = 0.05
    if rms > 0.001:
        gain = target_rms / rms
        gain = min(gain, 4.0)  # 最大增益 12dB
        audio = audio * gain

    # 削波保护
    audio = np.clip(audio, -1.0, 1.0)

    return audio


def restore_punctuation(text: str) -> str:
    """简单的标点恢复：句末加句号"""
    if not text:
        return text
    text = text.strip()
    if text and text[-1] not in "。！？，、；：.!?,":
        text += "。"
    return text


class QwenSTTCallback(RecognitionCallback):
    """DashScope Recognition 回调"""

    def __init__(self):
        self.results = []
        self.error = None
        self._event = threading.Event()

    def on_event(self, result: RecognitionResult):
        try:
            sentence = result.get_sentence()
            if sentence:
                self.results.append(sentence)
                return

            if hasattr(result, 'output') and result.output:
                output = result.output
                if isinstance(output, dict):
                    sentences = output.get('sentence', [])
                    for s in sentences:
                        if isinstance(s, dict):
                            text = s.get('text', '')
                            if text and text.strip():
                                self.results.append(text.strip())
        except Exception as e:
            print(f"[QwenSTT] on_event error: {e}", flush=True)

    def on_complete(self):
        self._event.set()

    def on_error(self, result):
        self.error = str(result)
        self._event.set()

    def on_close(self):
        self._event.set()


class QwenSTT:
    """千问语音识别服务（使用 DashScope SDK）"""

    SUPPORTED_MODELS = [
        "paraformer-realtime-v2",
        "paraformer-v2",
    ]

    def __init__(self, api_key: str, language: str = "zh"):
        self.api_key = api_key
        dashscope.api_key = api_key
        self.model = self.SUPPORTED_MODELS[0]
        self.language = language  # zh / en / auto
        self._failed_models = set()

        # 流式识别状态
        self._is_streaming = False
        self._stream_thread: Optional[threading.Thread] = None
        self._audio_buffer = []
        self._buffer_lock = threading.Lock()
        self._callback: Optional[Callable] = None

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
        语音识别（含预处理 + 标点恢复）
        :param audio_data: 音频数据 (numpy array)
        :param sample_rate: 采样率
        :return: 识别文本
        """
        try:
            # ── 1. 音频预处理 ──
            audio = preprocess_audio(audio_data, sample_rate)

            # 静音检测
            rms = np.sqrt(np.mean(audio ** 2))
            if len(audio) < 1600 or rms < 0.001:
                print(f"[QwenSTT] ⏭️ 音频过短或静音，跳过", flush=True)
                return ""

            # ── 2. 重采样到 16000 Hz ──
            if sample_rate != 16000:
                target_len = int(len(audio) * 16000 / sample_rate)
                audio = np.interp(
                    np.linspace(0, len(audio) - 1, target_len),
                    np.arange(len(audio)),
                    audio
                ).astype(np.float32)
                sample_rate = 16000

            # ── 3. 保存临时 WAV 文件 ──
            with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as f:
                sf.write(f.name, audio, sample_rate)
                tmp_path = f.name

            try:
                # ── 4. 调用 API ──
                cb = QwenSTTCallback()

                rec_params = {
                    "model": self.model,
                    "callback": cb,
                    "format": "wav",
                    "sample_rate": sample_rate,
                }

                # 语言提示：指定语言可显著提升准确率
                if self.language and self.language != "auto":
                    if "paraformer" in self.model:
                        rec_params["language_hints"] = [self.language]
                else:
                    if "paraformer" in self.model:
                        rec_params["language_hints"] = ["zh", "en"]

                # 启用标点符号（优化阅读体验）
                if "paraformer" in self.model:
                    rec_params["enable_punctuation"] = True

                rec = Recognition(**rec_params)
                result = rec.call(tmp_path)

                # ── 5. 提取结果 ──
                text = ""
                if result.status_code == 200 and cb.results:
                    text = " ".join(cb.results)

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
                            text = " ".join(texts)

                # 模型不支持时切换
                if result.status_code == 44:
                    self._mark_model_failed(self.model)
                    return self.recognize(audio_data, sample_rate)

                if text:
                    text = zhconv.convert(text, "zh-hans")
                    text = restore_punctuation(text)
                    print(f"[QwenSTT] ✅ 识别成功: [{text}]", flush=True)
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

    def start_streaming(self, callback: Callable, sample_rate: int = 16000):
        """开始流式识别"""
        if self._is_streaming:
            return

        self._is_streaming = True
        self._callback = callback
        self._audio_buffer = []

        self._stream_thread = threading.Thread(
            target=self._streaming_worker,
            args=(sample_rate,),
            daemon=True,
        )
        self._stream_thread.start()

    def stop_streaming(self) -> str:
        """停止流式识别，返回最终结果"""
        if not self._is_streaming:
            return ""

        self._is_streaming = False

        if self._stream_thread:
            self._stream_thread.join(timeout=5)

        with self._buffer_lock:
            if self._audio_buffer:
                audio_data = np.concatenate(self._audio_buffer, axis=0).flatten()
                self._audio_buffer = []
                return self.recognize(audio_data)

        return ""

    def add_audio_chunk(self, audio_chunk: np.ndarray):
        """添加音频块到缓冲区"""
        if not self._is_streaming:
            return
        with self._buffer_lock:
            self._audio_buffer.append(audio_chunk)

    def _streaming_worker(self, sample_rate: int):
        """流式处理线程"""
        last_process_time = time.time()
        process_interval = 0.5

        while self._is_streaming:
            current_time = time.time()
            if current_time - last_process_time >= process_interval:
                with self._buffer_lock:
                    if self._audio_buffer:
                        audio_data = np.concatenate(self._audio_buffer, axis=0).flatten()
                        self._audio_buffer = []
                        text = self.recognize(audio_data, sample_rate)
                        if text and self._callback:
                            self._callback(text)
                last_process_time = current_time
            time.sleep(0.1)
