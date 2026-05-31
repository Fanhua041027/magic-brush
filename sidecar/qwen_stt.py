"""Qwen (千问) STT service using DashScope Paraformer API."""

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


class QwenSTTCallback(RecognitionCallback):
    """DashScope Recognition 回调"""

    def __init__(self):
        self.results = []
        self.error = None
        self._event = threading.Event()

    def on_event(self, result: RecognitionResult):
        # 尝试多种方式提取文本
        try:
            # 方式1: get_sentence()
            sentence = result.get_sentence()
            if sentence:
                self.results.append(sentence)
                return

            # 方式2: 直接访问 output
            if hasattr(result, 'output') and result.output:
                output = result.output
                if isinstance(output, dict):
                    # 从 sentence 列表中提取文本
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

    # 支持的模型列表（按优先级排序）
    SUPPORTED_MODELS = [
        "paraformer-realtime-v2",  # 唯一支持文件识别的实时模型
    ]

    def __init__(self, api_key: str):
        self.api_key = api_key
        dashscope.api_key = api_key
        self.model = self.SUPPORTED_MODELS[0]  # 使用第一个可用模型
        self.language = "auto"  # auto: 中英文混合识别
        self._failed_models = set()  # 记录失败的模型

        # 流式识别状态
        self._is_streaming = False
        self._stream_thread: Optional[threading.Thread] = None
        self._audio_buffer = []
        self._buffer_lock = threading.Lock()
        self._callback: Optional[Callable] = None

    def _get_next_model(self) -> str:
        """获取下一个可用的模型"""
        for model in self.SUPPORTED_MODELS:
            if model not in self._failed_models:
                return model
        # 所有模型都失败了，重置并使用第一个
        self._failed_models.clear()
        return self.SUPPORTED_MODELS[0]

    def _mark_model_failed(self, model: str):
        """标记模型为失败"""
        self._failed_models.add(model)
        print(f"[QwenSTT] Model {model} marked as failed", flush=True)
        # 切换到下一个模型
        self.model = self._get_next_model()
        print(f"[QwenSTT] Switched to model: {self.model}", flush=True)

    def recognize(self, audio_data: np.ndarray, sample_rate: int = 16000) -> str:
        """
        一句话识别
        :param audio_data: 音频数据 (numpy array)
        :param sample_rate: 采样率
        :return: 识别结果
        """
        try:
            print(f"[QwenSTT] recognize: audio_size={len(audio_data)}, sample_rate={sample_rate}", flush=True)

            # 确保音频数据是 float32 格式
            if audio_data.dtype != np.float32:
                audio_data = audio_data.astype(np.float32)

            # 如果是整数格式，转换为浮点
            if audio_data.max() > 1.0:
                audio_data = audio_data / 32767.0

            rms = np.sqrt(np.mean(audio_data**2))
            print(f"[QwenSTT] audio RMS={rms:.6f}", flush=True)

            # 如果音频太短或太安静，直接返回空
            if len(audio_data) < 1600 or rms < 0.001:
                print(f"[QwenSTT] audio too short or silent, skipping", flush=True)
                return ""

            # 重采样到 16000 Hz（千问 API 推荐采样率）
            if sample_rate != 16000:
                target_len = int(len(audio_data) * 16000 / sample_rate)
                audio_data = np.interp(
                    np.linspace(0, len(audio_data) - 1, target_len),
                    np.arange(len(audio_data)),
                    audio_data
                ).astype(np.float32)
                sample_rate = 16000
                print(f"[QwenSTT] resampled to 16000 Hz, new size={len(audio_data)}", flush=True)

            # 保存为临时 WAV 文件
            with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as f:
                sf.write(f.name, audio_data, sample_rate)
                tmp_path = f.name

            print(f"[QwenSTT] saved to {tmp_path}", flush=True)

            try:
                # 使用 DashScope SDK 进行识别
                cb = QwenSTTCallback()

                # 根据模型选择参数
                rec_params = {
                    "model": self.model,
                    "callback": cb,
                    "format": "wav",
                    "sample_rate": sample_rate,
                }

                # paraformer 模型支持 language_hints
                if "paraformer" in self.model:
                    rec_params["language_hints"] = ["zh", "en"]

                rec = Recognition(**rec_params)
                print(f"[QwenSTT] calling DashScope API...", flush=True)
                result = rec.call(tmp_path)

                print(f"[QwenSTT] result status={result.status_code}", flush=True)
                print(f"[QwenSTT] result output={result.output}", flush=True)

                # 从回调中获取结果
                if result.status_code == 200 and cb.results:
                    text = " ".join(cb.results)
                    # 转换为简体中文
                    text = zhconv.convert(text, "zh-hans")
                    print(f"[QwenSTT] recognized from callback: [{text}]", flush=True)
                    return text

                # 如果回调没有结果，直接从 output 中提取
                if result.status_code == 200 and result.output:
                    output = result.output
                    if isinstance(output, dict):
                        sentences = output.get('sentence', [])
                        texts = []
                        for s in sentences:
                            if isinstance(s, dict):
                                text = s.get('text', '')
                                if text and text.strip():
                                    texts.append(text.strip())
                        if texts:
                            combined = " ".join(texts)
                            # 转换为简体中文
                            combined = zhconv.convert(combined, "zh-hans")
                            print(f"[QwenSTT] recognized from output: [{combined}]", flush=True)
                            return combined

                # 检查是否需要切换模型
                if result.status_code == 44:
                    print(f"[QwenSTT] Model {self.model} not supported (status 44), switching...", flush=True)
                    self._mark_model_failed(self.model)
                    # 重试一次
                    return self.recognize(audio_data, 16000)

                if cb.error:
                    print(f"[QwenSTT] Recognition error: {cb.error}", flush=True)
                else:
                    print(f"[QwenSTT] No results, status={result.status_code}", flush=True)

            finally:
                # 清理临时文件
                try:
                    os.unlink(tmp_path)
                except Exception:
                    pass

        except Exception as e:
            print(f"[QwenSTT] Recognition error: {e}", flush=True)

        return ""

    def start_streaming(self, callback: Callable, sample_rate: int = 16000):
        """
        开始流式识别
        :param callback: 回调函数，接收识别结果
        :param sample_rate: 采样率
        """
        if self._is_streaming:
            return

        self._is_streaming = True
        self._callback = callback
        self._audio_buffer = []

        # 启动流式识别线程
        self._stream_thread = threading.Thread(
            target=self._streaming_worker,
            args=(sample_rate,),
            daemon=True,
        )
        self._stream_thread.start()

    def stop_streaming(self) -> str:
        """
        停止流式识别
        :return: 最终识别结果
        """
        if not self._is_streaming:
            return ""

        self._is_streaming = False

        # 等待线程结束
        if self._stream_thread:
            self._stream_thread.join(timeout=5)

        # 获取缓冲区中的剩余音频
        with self._buffer_lock:
            if self._audio_buffer:
                audio_data = np.concatenate(self._audio_buffer, axis=0).flatten()
                self._audio_buffer = []
                # 进行最终识别
                return self.recognize(audio_data)

        return ""

    def add_audio_chunk(self, audio_chunk: np.ndarray):
        """
        添加音频块到流式识别缓冲区
        :param audio_chunk: 音频块
        """
        if not self._is_streaming:
            return

        with self._buffer_lock:
            self._audio_buffer.append(audio_chunk)

    def _streaming_worker(self, sample_rate: int):
        """流式识别工作线程"""
        last_process_time = time.time()
        process_interval = 0.5  # 每 0.5 秒处理一次

        while self._is_streaming:
            current_time = time.time()

            # 检查是否需要处理
            if current_time - last_process_time >= process_interval:
                with self._buffer_lock:
                    if self._audio_buffer:
                        # 获取并清空缓冲区
                        audio_data = np.concatenate(self._audio_buffer, axis=0).flatten()
                        self._audio_buffer = []

                        # 进行识别
                        text = self.recognize(audio_data, sample_rate)
                        if text and self._callback:
                            self._callback(text)

                last_process_time = current_time

            time.sleep(0.1)
