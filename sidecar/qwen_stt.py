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


class QwenSTTCallback(RecognitionCallback):
    """DashScope Recognition 回调"""

    def __init__(self):
        self.results = []
        self.error = None
        self._event = threading.Event()

    def on_event(self, result: RecognitionResult):
        sentence = result.get_sentence()
        if sentence:
            self.results.append(sentence)

    def on_complete(self):
        self._event.set()

    def on_error(self, result):
        self.error = str(result)
        self._event.set()

    def on_close(self):
        self._event.set()


class QwenSTT:
    """千问语音识别服务（使用 DashScope SDK）"""

    def __init__(self, api_key: str):
        self.api_key = api_key
        dashscope.api_key = api_key
        self.model = "paraformer-realtime-v2"

        # 流式识别状态
        self._is_streaming = False
        self._stream_thread: Optional[threading.Thread] = None
        self._audio_buffer = []
        self._buffer_lock = threading.Lock()
        self._callback: Optional[Callable] = None

    def recognize(self, audio_data: np.ndarray, sample_rate: int = 16000) -> str:
        """
        一句话识别
        :param audio_data: 音频数据 (numpy array)
        :param sample_rate: 采样率
        :return: 识别结果
        """
        try:
            # 确保音频数据是 float32 格式
            if audio_data.dtype != np.float32:
                audio_data = audio_data.astype(np.float32)

            # 如果是整数格式，转换为浮点
            if audio_data.max() > 1.0:
                audio_data = audio_data / 32767.0

            # 保存为临时 WAV 文件
            with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as f:
                sf.write(f.name, audio_data, sample_rate)
                tmp_path = f.name

            try:
                # 使用 DashScope SDK 进行识别
                cb = QwenSTTCallback()
                rec = Recognition(
                    model=self.model,
                    callback=cb,
                    format="wav",
                    sample_rate=sample_rate,
                )
                result = rec.call(tmp_path)

                if result.status_code == 200 and cb.results:
                    return " ".join(cb.results)
                elif cb.error:
                    print(f"[QwenSTT] Recognition error: {cb.error}")

            finally:
                # 清理临时文件
                try:
                    os.unlink(tmp_path)
                except Exception:
                    pass

        except Exception as e:
            print(f"[QwenSTT] Recognition error: {e}")

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
