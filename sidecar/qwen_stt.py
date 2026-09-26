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
import socket
import base64
import requests
import websocket
import uuid
import queue
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

    # Step 3: RMS normalization (conservative gain for system audio)
    rms = np.sqrt(np.mean(audio ** 2) + 1e-10)
    if rms > 0.0001:
        target_rms = 0.06
        gain = target_rms / rms
        gain = min(max(gain, 0.5), 3.0)
        audio = audio * gain
    else:
        return audio * 0.1

    # Step 4: Noise gate (conservative, preserve system audio detail)
    if noise_reduction:
        frame_size = int(sample_rate * 0.04)  # 40ms frames
        if frame_size > 0:
            for start in range(0, len(audio), frame_size):
                end = min(start + frame_size, len(audio))
                frame = audio[start:end]
                frame_rms = np.sqrt(np.mean(frame ** 2) + 1e-10)
                if frame_rms < 0.0015:
                    gain = max(0.3, (frame_rms / 0.0015) ** 2)
                    audio[start:end] = frame * gain

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

def _extract_text(value) -> list[str]:
    """Extract transcript text from DashScope's varying response shapes."""
    found = []
    if isinstance(value, dict):
        text = value.get('text')
        if isinstance(text, str) and text.strip():
            found.append(text.strip())
        for key in ('sentence', 'sentences', 'output', 'result'):
            if key in value:
                found.extend(_extract_text(value[key]))
    elif isinstance(value, (list, tuple)):
        for item in value:
            found.extend(_extract_text(item))
    return found


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

    def __init__(self, api_key: str, language: str = "zh", base_url: str = "https://dashscope.aliyuncs.com/compatible-mode/v1", model: str = "paraformer-realtime-v2"):
        self.api_key = api_key
        self.base_url = base_url.rstrip('/')
        self.model = model
        dashscope.api_key = api_key
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
        self._stream_queue = queue.Queue(maxsize=100)
        self._stream_ws = None
        self._stream_texts = []
        self._stream_error = None
        self._stream_ready = threading.Event()
        self._stream_stop_lock = threading.Lock()
        self._stream_sentence_ids = set()
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

    @staticmethod
    def _event_name(event: dict) -> str:
        header = event.get("header") or {}
        return header.get("event") or header.get("action") or ""

    @staticmethod
    def _sentence_text(event: dict) -> str:
        if QwenSTT._event_name(event) != "result-generated":
            return ""
        sentence = (((event.get("payload") or {}).get("output") or {}).get("sentence") or {})
        if sentence.get("heartbeat") is True or sentence.get("sentence_end") is not True:
            return ""
        text = sentence.get("text")
        return text.strip() if isinstance(text, str) else ""

    def _websocket_url(self) -> str:
        """Build the DashScope duplex inference WebSocket endpoint."""
        url = self.base_url
        if url.startswith("https://"):
            url = "wss://" + url[len("https://"):]
        elif url.startswith("http://"):
            url = "ws://" + url[len("http://"):]
        if "/compatible-mode/v1" in url:
            return url.replace("/compatible-mode/v1", "/api-ws/v1/inference")
        if url.endswith("/api/v1"):
            return url[:-len("/api/v1")] + "/api-ws/v1/inference"
        return url.rstrip("/") + "/api-ws/v1/inference"

    def _recognize_websocket(self, audio: np.ndarray) -> str:
        """Recognize audio through Qwen's duplex WebSocket protocol."""
        task_id = str(uuid.uuid4())
        headers = [f"Authorization: Bearer {self.api_key}", "User-Agent: magic-brush/1.0"]
        workspace_id = os.environ.get("DASHSCOPE_WORKSPACE_ID", "").strip()
        if workspace_id:
            headers.append(f"X-DashScope-WorkSpace: {workspace_id}")
        ws = websocket.create_connection(
            self._websocket_url(),
            header=headers,
            timeout=30,
        )
        try:
            ws.send(json.dumps({
                "header": {"action": "run-task", "task_id": task_id, "streaming": "duplex"},
                "payload": {
                    "task_group": "audio", "task": "asr", "function": "recognition",
                    "model": self.model,
                    "parameters": {
                        "format": "pcm", "sample_rate": 16000,
                        "language_hints": [self.language] if self.language and self.language != "auto" else ["zh", "en"],
                        "semantic_punctuation_enabled": True,
                        "heartbeat": True,
                    },
                    "input": {},
                },
            }), opcode=websocket.ABNF.OPCODE_TEXT)
            started = False
            texts = []
            pcm = np.asarray(audio, dtype=np.float32)
            pcm16 = np.clip(pcm, -1.0, 1.0).astype(np.float32)
            pcm16 = (pcm16 * 32767.0).astype("<i2").tobytes()
            for offset in range(0, len(pcm16), 3200):
                while not started:
                    message = ws.recv()
                    if isinstance(message, bytes):
                        continue
                    try:
                        event = json.loads(message)
                    except (TypeError, json.JSONDecodeError):
                        continue
                    action = self._event_name(event)
                    if action == "task-started":
                        started = True
                    elif action in ("task-failed", "error"):
                        raise RuntimeError(str(event))
                ws.send(pcm16[offset:offset + 3200], opcode=websocket.ABNF.OPCODE_BINARY)
                ws.settimeout(0.01)
                try:
                    while True:
                        message = ws.recv()
                        if isinstance(message, str):
                            try:
                                event = json.loads(message)
                                text = self._sentence_text(event)
                                if text:
                                    texts.append(text)
                            except json.JSONDecodeError:
                                pass
                except (websocket.WebSocketTimeoutException, socket.timeout):
                    pass
                finally:
                    ws.settimeout(30)
            ws.send(json.dumps({
                "header": {"action": "finish-task", "task_id": task_id, "streaming": "duplex"},
                "payload": {"input": {}},
            }), opcode=websocket.ABNF.OPCODE_TEXT)
            while True:
                message = ws.recv()
                if isinstance(message, bytes):
                    continue
                event = json.loads(message)
                text = self._sentence_text(event)
                if text:
                    texts.append(text)
                event_name = self._event_name(event)
                if event_name == "task-finished":
                    break
                if event_name == "task-failed":
                    header = event.get("header") or {}
                    raise RuntimeError(f"{header.get('error_code', 'TASK_FAILED')}: {header.get('error_message', '')}")
            return self._clean_text("".join(dict.fromkeys(t.strip() for t in texts if t.strip())))
        finally:
            ws.close()

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

            if self.model.startswith("qwen-audio"):
                text = self._recognize_websocket(audio)
                print(f"[QwenSTT] ASR result: text_length={len(text)}", flush=True)
                return text

            # ── 2. 保存临时 WAV ──
            with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as f:
                sf.write(f.name, audio, 16000)
                tmp_path = f.name

            try:
                if self.model.startswith("qwen-audio"):
                    wav_buffer = io.BytesIO()
                    sf.write(wav_buffer, audio, 16000, format="WAV", subtype="PCM_16")
                    encoded = base64.b64encode(wav_buffer.getvalue()).decode("ascii")
                    response = requests.post(
                        self.base_url.replace("/compatible-mode/v1", "/api/v1") + "/services/aigc/multimodal-generation/generation",
                        headers={"Authorization": "Bearer " + self.api_key, "Content-Type": "application/json"},
                        json={"model": self.model, "input": {"messages": [{"role": "user", "content": [
                            {"audio": "data:audio/wav;base64," + encoded},
                            {"text": "请准确转写这段音频，只输出识别文字。"},
                        ]}]}, "parameters": {"temperature": 0}},
                        timeout=120,
                    )
                    if not response.ok:
                        raise RuntimeError(f"ASR {response.status_code}: {response.text[:500]}")
                    payload = response.json()
                    text = "".join(_extract_text(payload.get("output", payload))).strip()
                    print(f"[QwenSTT] ASR result: text_length={len(text)}", flush=True)
                    return self._clean_text(text)

                # ── 3. 调用 DashScope API ──
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
                    texts = _extract_text(result.output)
                    if texts:
                        text = "".join(dict.fromkeys(texts))

                # 模型不支持的错误码 44 → 切换模型重试
                if result.status_code == 44:
                    self._mark_model_failed(self.model)
                    return self.recognize(audio_data, sample_rate)

                if text:
                    # 简繁转换
                    text = zhconv.convert(text, "zh-hans")
                    text = restore_punctuation(text)
                    text = self._clean_text(text)
                    print(f"[QwenSTT] ✅ 识别成功 ({len(text)} chars)", flush=True)
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
        """Start one persistent duplex WebSocket session and await readiness."""
        with self._stream_stop_lock:
            if self._is_streaming:
                return False
            self._is_streaming = True
            self._callback = callback
            self._stop_event.clear()
            self._stream_error = None
            self._stream_ready.clear()
            self._stream_texts = []
            self._stream_sentence_ids = set()
            self._stream_queue = queue.Queue(maxsize=100)
            self._stream_thread = threading.Thread(target=self._streaming_worker, daemon=True)
            self._stream_thread.start()
        if not self._stream_ready.wait(timeout=10):
            self._stream_error = self._stream_error or "streaming session startup timeout"
            self.stop_streaming()
            return False
        if self._stream_error:
            self.stop_streaming()
            return False
        return True

    def stop_streaming(self) -> str:
        """Flush audio, finish the duplex task, and return committed text."""
        with self._stream_stop_lock:
            if not self._is_streaming and not self._stream_thread:
                return ""
            self._is_streaming = False
            self._stop_event.set()
            try:
                self._stream_queue.put_nowait(None)
            except queue.Full:
                pass
            thread = self._stream_thread
        if thread:
            thread.join(timeout=15)
            if thread.is_alive():
                self._stream_error = self._stream_error or "streaming worker stop timeout"
            with self._stream_stop_lock:
                if self._stream_thread is thread and not thread.is_alive():
                    self._stream_thread = None
        if self._stream_error:
            print(f"[QwenSTT] streaming error: {self._stream_error}", flush=True)
        return self._clean_text("".join(self._stream_texts))

    def add_audio_chunk(self, audio_chunk: np.ndarray):
        """Queue canonical audio as mono PCM16 without blocking capture."""
        if not self._is_streaming:
            return
        chunk = np.asarray(audio_chunk, dtype=np.float32)
        if chunk.ndim > 1:
            chunk = chunk.mean(axis=1)
        pcm = (np.clip(chunk.reshape(-1), -1, 1) * 32767).astype("<i2").tobytes()
        try:
            self._stream_queue.put_nowait(pcm)
        except queue.Full:
            print("[QwenSTT] streaming queue full; dropping audio chunk", flush=True)

    def _streaming_worker(self):
        """Own the persistent WebSocket and both duplex directions."""
        ws = None
        try:
            headers = [f"Authorization: Bearer {self.api_key}", "User-Agent: magic-brush/1.0"]
            workspace = os.environ.get("DASHSCOPE_WORKSPACE_ID", "").strip()
            if workspace:
                headers.append(f"X-DashScope-WorkSpace: {workspace}")
            ws = websocket.create_connection(self._websocket_url(), header=headers, timeout=30)
            task_id = str(uuid.uuid4())
            ws.send(json.dumps({"header": {"action": "run-task", "task_id": task_id, "streaming": "duplex"}, "payload": {"task_group": "audio", "task": "asr", "function": "recognition", "model": self.model, "parameters": {"format": "pcm", "sample_rate": 16000, "language_hints": [self.language] if self.language != "auto" else ["zh", "en"], "semantic_punctuation_enabled": True, "heartbeat": True}, "input": {}}}))
            while True:
                event = json.loads(ws.recv())
                name = self._event_name(event)
                if name == "task-failed":
                    raise RuntimeError(f"{event.get('header', {}).get('error_code')}: {event.get('header', {}).get('error_message')}")
                if name == "task-started":
                    self._stream_ready.set()
                    break
            ws.settimeout(0.05)
            while self._is_streaming or not self._stream_queue.empty():
                try:
                    chunk = self._stream_queue.get(timeout=0.05)
                    if chunk is None:
                        continue
                    ws.send(chunk, opcode=websocket.ABNF.OPCODE_BINARY)
                except queue.Empty:
                    pass
                try:
                    while True:
                        msg = ws.recv()
                        if isinstance(msg, str):
                            event = json.loads(msg)
                            sentence = (((event.get("payload") or {}).get("output") or {}).get("sentence") or {})
                            if self._event_name(event) == "result-generated" and not sentence.get("heartbeat"):
                                text = sentence.get("text", "").strip()
                                sentence_id = sentence.get("sentence_id")
                                key = str(sentence_id) if sentence_id is not None else text
                                if text and sentence.get("sentence_end") is True and key not in self._stream_sentence_ids:
                                    self._stream_sentence_ids.add(key)
                                    self._stream_texts.append(text)
                                    if self._callback:
                                        self._callback(text)
                except (websocket.WebSocketTimeoutException, socket.timeout):
                    pass
            ws.settimeout(30)
            ws.send(json.dumps({"header": {"action": "finish-task", "task_id": task_id, "streaming": "duplex"}, "payload": {"input": {}}}))
            while True:
                event = json.loads(ws.recv())
                name = self._event_name(event)
                if name == "result-generated":
                    text = self._sentence_text(event)
                    sentence = (((event.get("payload") or {}).get("output") or {}).get("sentence") or {})
                    sentence_id = sentence.get("sentence_id")
                    key = str(sentence_id) if sentence_id is not None else text
                    if text and key not in self._stream_sentence_ids:
                        self._stream_sentence_ids.add(key)
                        self._stream_texts.append(text)
                        if self._callback:
                            self._callback(text)
                if name == "task-finished":
                    break
                if name == "task-failed":
                    raise RuntimeError(str(event))
        except Exception as exc:
            self._stream_error = str(exc)
        finally:
            if ws:
                ws.close()
