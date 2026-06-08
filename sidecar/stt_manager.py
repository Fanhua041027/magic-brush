"""STT Manager — 带自动备份的语音识别服务管理器

支持服务链：
  主服务: Qwen3-ASR-Flash (本地) → 备份1: Qwen Cloud Paraformer → 备份2: Local Whisper

特性：
  - 初始化时同时加载所有可用服务
  - 识别时按优先级依次尝试，失败自动降级
  - 流式识别自动选择当前可用服务
  - 详细的识别来源报告
"""

import time
import threading
import numpy as np
from typing import Optional, Callable, List, Tuple


class STTManager:
    """语音识别管理器，带多级备份"""

    SERVICE_NAMES = {
        "qwen_local": "千问本地 Qwen3-ASR-Flash",
        "qwen_cloud": "千问云端 Paraformer v2",
        "local_whisper": "本地 Whisper",
    }

    # 服务优先级：索引越小优先级越高
    DEFAULT_PRIORITY = ["qwen_local", "qwen_cloud", "local_whisper"]

    def __init__(self, api_key: str = "", whisper_model: str = "medium",
                 whisper_device: str = "auto", whisper_language: str = "zh",
                 priority: Optional[List[str]] = None):
        """
        Args:
            priority: 服务优先级列表，如 ["qwen_cloud"] 只加载云端
                      默认 None 使用 DEFAULT_PRIORITY 全部加载
        """
        self.api_key = api_key
        self.whisper_model_name = whisper_model
        self.whisper_device = whisper_device
        self.whisper_language = whisper_language
        self._priority_config = priority or list(self.DEFAULT_PRIORITY)

        # 服务实例
        self._qwen_local = None
        self._qwen_cloud = None
        self._whisper = None

        # 服务状态
        self._service_ready: dict = {}
        self._current_priority = list(self._priority_config)
        self._lock = threading.Lock()

        # 统计
        self._usage_stats = {name: 0 for name in self.SERVICE_NAMES}

    # ── 初始化 ─────────────────────────────────────────────

    def initialize_all(self) -> List[str]:
        """按配置的优先级初始化服务，返回成功加载的服务列表"""
        loaded = []

        for svc in self._priority_config:
            ok = False
            if svc == "qwen_local":
                ok = self._init_qwen_local()
            elif svc == "qwen_cloud":
                ok = self._init_qwen_cloud()
            elif svc == "local_whisper":
                ok = self._init_whisper()

            if ok:
                loaded.append(svc)
                self._service_ready[svc] = True
                print(f"[STTManager] ✅ {self.SERVICE_NAMES[svc]} 就绪")
            else:
                print(f"[STTManager] ⏭️ {self.SERVICE_NAMES.get(svc, svc)} 不可用，跳过")

        self._current_priority = loaded

        if not loaded:
            print("[STTManager] ❌ 没有可用的 STT 服务")
        else:
            chain = " → ".join(self.SERVICE_NAMES[s] for s in loaded)
            print(f"[STTManager] 🔗 服务链: {chain}")

        return loaded

    def _init_qwen_local(self) -> bool:
        """初始化千问本地 ASR"""
        try:
            from qwen_asr_local import QwenASRLocal
            self._qwen_local = QwenASRLocal("Qwen/Qwen3-ASR-Flash")
            ok = self._qwen_local.load_model()
            if not ok:
                print("[STTManager] 千问本地加载失败")
                self._qwen_local = None
            return ok
        except Exception as e:
            print(f"[STTManager] 千问本地初始化异常: {e}")
            self._qwen_local = None
            return False

    def _init_qwen_cloud(self) -> bool:
        """初始化千问云端 STT"""
        if not self.api_key:
            print("[STTManager] 千问云端 API Key 为空，跳过")
            return False
        try:
            from qwen_stt import QwenSTT
            self._qwen_cloud = QwenSTT(self.api_key)
            return True
        except Exception as e:
            print(f"[STTManager] 千问云端初始化异常: {e}")
            self._qwen_cloud = None
            return False

    def _init_whisper(self) -> bool:
        """初始化本地 Whisper"""
        try:
            from transcribe import Transcriber
            self._whisper = Transcriber(
                model_name=self.whisper_model_name,
                device=self.whisper_device,
                language=self.whisper_language,
            )
            return True
        except Exception as e:
            print(f"[STTManager] Whisper 初始化异常: {e}")
            self._whisper = None
            return False

    # ── 服务状态 ─────────────────────────────────────────────

    def get_available_services(self) -> List[str]:
        """获取当前可用服务列表"""
        return list(self._current_priority)

    def get_primary_service(self) -> Optional[str]:
        """获取当前主服务"""
        return self._current_priority[0] if self._current_priority else None

    def get_service_name(self, service_id: str) -> str:
        """获取服务显示名称"""
        return self.SERVICE_NAMES.get(service_id, service_id)

    def get_usage_stats(self) -> dict:
        """获取使用统计"""
        return dict(self._usage_stats)

    def is_any_ready(self) -> bool:
        """是否有任何服务可用"""
        return len(self._current_priority) > 0

    # ── 核心识别 ─────────────────────────────────────────────

    def recognize(self, audio_data: np.ndarray, sample_rate: int = 16000) -> Tuple[str, str]:
        """
        识别音频（自动备份降级）

        Args:
            audio_data: 音频 numpy 数组
            sample_rate: 采样率

        Returns:
            (识别文本, 使用的服务名)
        """
        if audio_data is None or len(audio_data) == 0:
            return "", ""

        with self._lock:
            priority = list(self._current_priority)

        errors = []
        for service_id in priority:
            text = self._try_recognize(service_id, audio_data, sample_rate)
            if text:
                self._usage_stats[service_id] = self._usage_stats.get(service_id, 0) + 1
                source = self.SERVICE_NAMES.get(service_id, service_id)
                print(f"[STTManager] ✅ 识别成功 ({source}): {text[:50]}...")
                return text, service_id
            errors.append(f"{service_id}: 无结果")

        print(f"[STTManager] ❌ 所有服务都失败: {', '.join(errors)}")
        return "", ""

    def _try_recognize(self, service_id: str, audio_data: np.ndarray,
                       sample_rate: int) -> str:
        """尝试用指定服务识别"""
        try:
            if service_id == "qwen_local" and self._qwen_local:
                return self._qwen_local.recognize(audio_data, sample_rate)
            elif service_id == "qwen_cloud" and self._qwen_cloud:
                return self._qwen_cloud.recognize(audio_data, sample_rate)
            elif service_id == "local_whisper" and self._whisper:
                return self._whisper.transcribe(audio_data)
        except Exception as e:
            print(f"[STTManager] {service_id} 识别异常: {e}")
        return ""

    # ── 流式识别 ─────────────────────────────────────────────

    def start_streaming(self, callback: Callable, sample_rate: int = 16000):
        """
        开始流式识别（自动选择最高优先级可用服务）

        Returns:
            使用的服务名, None 如果无可用服务
        """
        for service_id in self._current_priority:
            engine = self._get_streaming_engine(service_id)
            if engine and hasattr(engine, 'start_streaming'):
                try:
                    engine.start_streaming(callback, sample_rate)
                    print(f"[STTManager] ▶️ 流式识别启动 ({self.SERVICE_NAMES[service_id]})")
                    return service_id
                except Exception as e:
                    print(f"[STTManager] {service_id} 流式启动失败: {e}")
                    continue
        print("[STTManager] ❌ 无可用流式识别服务")
        return None

    def stop_streaming(self) -> Tuple[str, str]:
        """
        停止流式识别

        Returns:
            (最终识别文本, 使用的服务名)
        """
        # 按优先级反向停止，使用最先启动的服务结果
        for service_id in reversed(self._current_priority):
            engine = self._get_streaming_engine(service_id)
            if engine and hasattr(engine, 'stop_streaming'):
                try:
                    text = engine.stop_streaming()
                    if text:
                        source = self.SERVICE_NAMES.get(service_id, service_id)
                        print(f"[STTManager] ⏹️ 流式停止 ({source})")
                        return text, service_id
                except Exception as e:
                    print(f"[STTManager] {service_id} 停止异常: {e}")
                    continue
        return "", ""

    def add_audio_chunk(self, audio_chunk: np.ndarray):
        """向所有已启动的流式服务添加音频块"""
        for service_id in self._current_priority:
            engine = self._get_streaming_engine(service_id)
            if engine and hasattr(engine, 'add_audio_chunk'):
                try:
                    engine.add_audio_chunk(audio_chunk)
                except Exception:
                    pass

    def _get_streaming_engine(self, service_id: str):
        """获取流式引擎实例"""
        if service_id == "qwen_local":
            return self._qwen_local.streaming if hasattr(self._qwen_local, 'streaming') and self._qwen_local else None
        elif service_id == "qwen_cloud":
            return self._qwen_cloud
        elif service_id == "local_whisper":
            return None  # Whisper 不支持流式
        return None

    # ── 健康检查 ─────────────────────────────────────────────

    def health_status(self) -> dict:
        """获取健康状态"""
        return {
            "available_services": self._current_priority,
            "primary": self.get_primary_service(),
            "service_names": {s: self.SERVICE_NAMES.get(s, s) for s in self._current_priority},
            "usage": self._usage_stats,
        }
