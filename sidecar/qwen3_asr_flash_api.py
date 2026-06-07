"""Client for Qwen3-ASR-Flash OpenAI-compatible API."""

from __future__ import annotations

import base64
import io
import wave
from typing import Any

import numpy as np
import requests


DEFAULT_BASE_URL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
DEFAULT_MODEL = "qwen3-asr-flash"


def _normalize_audio(audio_data: np.ndarray) -> np.ndarray:
    audio = np.asarray(audio_data, dtype=np.float32).flatten()
    if audio.size == 0:
        return audio
    max_abs = float(np.max(np.abs(audio)))
    if max_abs > 1.0:
        audio = audio / 32767.0
    return audio.astype(np.float32)


def encode_audio_as_data_uri(audio_data: np.ndarray, sample_rate: int) -> str:
    audio = _normalize_audio(audio_data)
    pcm16 = np.clip(audio, -1.0, 1.0)
    pcm16 = (pcm16 * 32767.0).astype(np.int16)
    with io.BytesIO() as buffer:
        with wave.open(buffer, "wb") as wav_file:
            wav_file.setnchannels(1)
            wav_file.setsampwidth(2)
            wav_file.setframerate(sample_rate)
            wav_file.writeframes(pcm16.tobytes())
        payload = base64.b64encode(buffer.getvalue()).decode("utf-8")
    return f"data:audio/wav;base64,{payload}"


def build_messages_payload(
    audio_data: np.ndarray,
    sample_rate: int = 16000,
    language: str = "zh",
    model: str = DEFAULT_MODEL,
) -> dict[str, Any]:
    return {
        "model": model,
        "messages": [
            {
                "role": "user",
                "content": [
                    {
                        "type": "input_audio",
                        "input_audio": {
                            "data": encode_audio_as_data_uri(audio_data, sample_rate),
                        },
                    }
                ],
            }
        ],
        "asr_options": {
            "language": language or "zh",
            "enable_itn": True,
        },
        "stream": False,
    }


def extract_transcript_text(response_json: dict[str, Any]) -> str:
    choices = response_json.get("choices") or []
    if not choices:
        return ""
    message = choices[0].get("message") or {}
    content = message.get("content")
    if isinstance(content, str):
        return content.strip()
    if isinstance(content, list):
        texts: list[str] = []
        for item in content:
            if isinstance(item, dict):
                text = item.get("text") or item.get("transcript") or item.get("content")
                if isinstance(text, str) and text.strip():
                    texts.append(text.strip())
            elif isinstance(item, str) and item.strip():
                texts.append(item.strip())
        return " ".join(texts).strip()
    return ""


class Qwen3ASRFlashAPI:
    def __init__(
        self,
        api_key: str,
        base_url: str = DEFAULT_BASE_URL,
        model: str = DEFAULT_MODEL,
        timeout_seconds: int = 60,
    ):
        self.api_key = (api_key or "").strip()
        self.base_url = (base_url or DEFAULT_BASE_URL).rstrip("/")
        self.model = model or DEFAULT_MODEL
        self.timeout_seconds = timeout_seconds

    def is_enabled(self) -> bool:
        return bool(self.api_key)

    def recognize(self, audio_data: np.ndarray, sample_rate: int = 16000, language: str = "zh") -> str:
        if not self.is_enabled():
            raise RuntimeError("Qwen3-ASR-Flash API key is empty")
        if audio_data is None or np.asarray(audio_data).size == 0:
            return ""

        response = requests.post(
            f"{self.base_url}/chat/completions",
            headers={
                "Authorization": f"Bearer {self.api_key}",
                "Content-Type": "application/json",
            },
            json=build_messages_payload(audio_data, sample_rate=sample_rate, language=language, model=self.model),
            timeout=self.timeout_seconds,
        )
        response.raise_for_status()
        transcript = extract_transcript_text(response.json())
        if not transcript:
            raise RuntimeError("Qwen3-ASR-Flash returned empty transcript")
        return transcript
