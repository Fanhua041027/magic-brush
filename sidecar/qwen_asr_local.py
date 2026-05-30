"""Qwen3-ASR-Flash 本地语音识别服务"""

import os
import time
import threading
import numpy as np
from typing import Optional, Callable
import tempfile
import soundfile as sf


class QwenASRLocal:
    """Qwen3-ASR-Flash 本地语音识别"""

    def __init__(self, model_path: str = None):
        self.model_path = model_path or "Qwen/Qwen3-ASR-Flash"
        self.pipeline = None
        self.is_loading = False
        self.is_ready = False
        self._lock = threading.Lock()

    def load_model(self):
        """加载模型"""
        if self.is_ready:
            return True

        if self.is_loading:
            # 等待加载完成
            while self.is_loading:
                time.sleep(0.1)
            return self.is_ready

        self.is_loading = True

        try:
            print(f"[QwenASRLocal] Loading model: {self.model_path}...")
            start_time = time.time()

            # 尝试使用 transformers 加载
            import torch
            from transformers import AutoProcessor, AutoModelForSpeechSeq2Seq

            # 获取模型路径
            model_path = self.model_path
            if model_path.startswith("Qwen/"):
                # 检查本地缓存
                import os
                cache_dir = os.path.expanduser("~/.cache/modelscope/hub/models")
                local_path = os.path.join(cache_dir, model_path.replace("/", "/"))
                if os.path.exists(local_path):
                    model_path = local_path
                    print(f"[QwenASRLocal] Using local model: {model_path}")

            # 加载处理器和模型
            self.processor = AutoProcessor.from_pretrained(model_path)
            self.model = AutoModelForSpeechSeq2Seq.from_pretrained(
                model_path,
                torch_dtype=torch.float16 if torch.cuda.is_available() else torch.float32,
                device_map="auto" if torch.cuda.is_available() else None,
            )

            elapsed = time.time() - start_time
            print(f"[QwenASRLocal] Model loaded in {elapsed:.2f}s")
            self.is_ready = True
            return True

        except Exception as e:
            print(f"[QwenASRLocal] Failed to load model: {e}")
            self.is_ready = False
            return False

        finally:
            self.is_loading = False

    def recognize(self, audio_data: np.ndarray, sample_rate: int = 16000) -> str:
        """
        识别音频
        :param audio_data: 音频数据 (numpy array)
        :param sample_rate: 采样率
        :return: 识别结果
        """
        if not self.is_ready:
            if not self.load_model():
                return ""

        try:
            import torch

            # 确保音频数据是 float32 格式
            if audio_data.dtype != np.float32:
                audio_data = audio_data.astype(np.float32)

            # 如果是整数格式，转换为浮点
            if audio_data.max() > 1.0:
                audio_data = audio_data / 32767.0

            # 处理音频输入
            inputs = self.processor(
                audio_data,
                sampling_rate=sample_rate,
                return_tensors="pt"
            )

            # 移动到正确的设备
            if torch.cuda.is_available():
                inputs = {k: v.to("cuda") for k, v in inputs.items()}

            # 生成文本
            with torch.no_grad():
                predicted_ids = self.model.generate(
                    inputs["input_features"],
                    max_new_tokens=256,
                )

            # 解码文本
            text = self.processor.batch_decode(predicted_ids, skip_special_tokens=True)[0]
            return text.strip()

        except Exception as e:
            print(f"[QwenASRLocal] Recognition error: {e}")
            return ""

    def recognize_streaming(self, audio_chunks: list, sample_rate: int = 16000) -> str:
        """
        流式识别（合并多个音频块）
        :param audio_chunks: 音频块列表
        :param sample_rate: 采样率
        :return: 识别结果
        """
        if not audio_chunks:
            return ""

        # 合并音频块
        audio_data = np.concatenate(audio_chunks, axis=0).flatten()
        return self.recognize(audio_data, sample_rate)


class QwenASRLocalStreaming:
    """Qwen3-ASR-Flash 本地流式语音识别"""

    def __init__(self, model_path: str = None):
        self.model_path = model_path or "Qwen/Qwen3-ASR-Flash"
        self.asr = QwenASRLocal(model_path)
        self._is_streaming = False
        self._audio_buffer = []
        self._buffer_lock = threading.Lock()
        self._callback: Optional[Callable] = None
        self._stream_thread: Optional[threading.Thread] = None

    def load_model(self):
        """加载模型"""
        return self.asr.load_model()

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
                return self.asr.recognize(audio_data)

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
        process_interval = 1.0  # 每 1.0 秒处理一次（本地模型处理较慢）

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
                        text = self.asr.recognize(audio_data, sample_rate)
                        if text and self._callback:
                            self._callback(text)

                last_process_time = current_time

            time.sleep(0.1)
