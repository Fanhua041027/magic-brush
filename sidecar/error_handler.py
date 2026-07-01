"""Error handling module — 增强错误恢复、健康监控、自动重试"""

import time
import traceback
import sys
import threading
from datetime import datetime, timedelta
from enum import Enum
from typing import Optional, Callable, Any
from functools import wraps


class ErrorCode(Enum):
    """错误代码枚举"""
    UNKNOWN = "UNKNOWN"
    AUDIO_INIT_FAILED = "AUDIO_INIT_FAILED"
    AUDIO_RECORD_FAILED = "AUDIO_RECORD_FAILED"
    TRANSCRIPTION_FAILED = "TRANSCRIPTION_FAILED"
    MODEL_LOAD_FAILED = "MODEL_LOAD_FAILED"
    KB_LOAD_FAILED = "KB_LOAD_FAILED"
    KB_SEARCH_FAILED = "KB_SEARCH_FAILED"
    NETWORK_ERROR = "NETWORK_ERROR"
    API_ERROR = "API_ERROR"
    INVALID_INPUT = "INVALID_INPUT"
    TIMEOUT = "TIMEOUT"
    RESOURCE_BUSY = "RESOURCE_BUSY"
    SERVICE_UNAVAILABLE = "SERVICE_UNAVAILABLE"


class AppError(Exception):
    """应用错误基类"""

    def __init__(
        self,
        code: ErrorCode,
        message: str,
        details: Optional[str] = None,
        recoverable: bool = True,
        retry_count: int = 0,
    ):
        self.code = code
        self.message = message
        self.details = details
        self.recoverable = recoverable
        self.retry_count = retry_count
        self.timestamp = datetime.now().isoformat()
        super().__init__(message)

    def to_dict(self) -> dict:
        return {
            "code": self.code.value,
            "message": self.message,
            "details": self.details,
            "recoverable": self.recoverable,
            "retry_count": self.retry_count,
            "timestamp": self.timestamp,
        }

    def __str__(self) -> str:
        return f"[{self.code.value}] {self.message}"


class ServiceHealth:
    """服务健康状态追踪器"""

    def __init__(self, name: str, cooldown: float = 30.0):
        self.name = name
        self.cooldown = cooldown  # 故障后冷却时间（秒）
        self._healthy = True
        self._last_failure: Optional[datetime] = None
        self._failure_count = 0
        self._consecutive_failures = 0
        self._lock = threading.Lock()

    @property
    def is_healthy(self) -> bool:
        with self._lock:
            if not self._healthy and self._last_failure:
                # 冷却期后自动恢复
                elapsed = (datetime.now() - self._last_failure).total_seconds()
                if elapsed >= self.cooldown:
                    self._healthy = True
                    self._consecutive_failures = 0
                    print(f"[Health] {self.name} 冷却期结束，自动恢复健康")
            return self._healthy

    def record_failure(self):
        with self._lock:
            self._healthy = False
            self._last_failure = datetime.now()
            self._failure_count += 1
            self._consecutive_failures += 1
            print(f"[Health] {self.name} 故障 (连续{self._consecutive_failures}次, 总计{self._failure_count}次)")

    def record_success(self):
        with self._lock:
            self._healthy = True
            self._consecutive_failures = 0

    def get_status(self) -> dict:
        with self._lock:
            return {
                "name": self.name,
                "healthy": self._healthy,
                "failure_count": self._failure_count,
                "consecutive_failures": self._consecutive_failures,
                "last_failure": self._last_failure.isoformat() if self._last_failure else None,
            }


class ErrorHandler:
    """错误处理器 — 带自动恢复和健康监控"""

    def __init__(self, max_retries: int = 3):
        self.max_retries = max_retries
        self.error_log: list[dict] = []
        self._lock = threading.Lock()
        self.error_callbacks: dict[ErrorCode, list[Callable]] = {}
        self.health_monitors: dict[str, ServiceHealth] = {}

    def register_service(self, name: str, cooldown: float = 30.0) -> ServiceHealth:
        """注册服务健康监控"""
        monitor = ServiceHealth(name, cooldown)
        self.health_monitors[name] = monitor
        return monitor

    def get_service_health(self, name: str) -> Optional[ServiceHealth]:
        return self.health_monitors.get(name)

    def all_services_healthy(self) -> bool:
        return all(m.is_healthy for m in self.health_monitors.values())

    def register_callback(self, code: ErrorCode, callback: Callable):
        """注册错误回调"""
        if code not in self.error_callbacks:
            self.error_callbacks[code] = []
        self.error_callbacks[code].append(callback)

    def handle_error(self, error: Exception, context: str = "") -> AppError:
        """处理错误"""
        if isinstance(error, AppError):
            app_error = error
        else:
            app_error = self._wrap_error(error, context)

        # 记录错误
        self._log_error(app_error, context)

        # 触发回调
        self._trigger_callbacks(app_error)

        return app_error

    def _wrap_error(self, error: Exception, context: str) -> AppError:
        """将普通异常包装为 AppError"""
        error_type = type(error).__name__
        error_msg = str(error)
        error_lower = error_msg.lower()

        code = ErrorCode.UNKNOWN
        recoverable = True

        if "audio" in error_lower or "sounddevice" in error_lower or "portaudio" in error_lower:
            code = ErrorCode.AUDIO_INIT_FAILED
        elif "whisper" in error_lower or "model" in error_lower or "cuda" in error_lower:
            code = ErrorCode.MODEL_LOAD_FAILED
        elif "timeout" in error_lower:
            code = ErrorCode.TIMEOUT
        elif "network" in error_lower or "connection" in error_lower or "dns" in error_lower:
            code = ErrorCode.NETWORK_ERROR
        elif "memory" in error_lower or "resource" in error_lower:
            code = ErrorCode.RESOURCE_BUSY
            recoverable = False
        elif "api" in error_lower or "http" in error_lower or "status" in error_lower:
            code = ErrorCode.API_ERROR
        elif "permission" in error_lower or "access" in error_lower:
            code = ErrorCode.SERVICE_UNAVAILABLE
            recoverable = False

        details = f"{error_type}: {error_msg}\n{traceback.format_exc()}"

        return AppError(
            code=code,
            message=f"{context}: {error_msg}" if context else error_msg,
            details=details,
            recoverable=recoverable,
        )

    def _log_error(self, error: AppError, context: str):
        """记录错误日志（线程安全）"""
        log_entry = {
            "timestamp": error.timestamp,
            "code": error.code.value,
            "message": error.message,
            "context": context,
            "recoverable": error.recoverable,
        }
        with self._lock:
            self.error_log.append(log_entry)
            if len(self.error_log) > 1000:
                self.error_log = self.error_log[-500:]

        print(f"[ERROR] [{error.code.value}] {error.message}", flush=True)
        if error.details:
            print(f"[ERROR] Details: {error.details[:200]}...", flush=True)

    def _trigger_callbacks(self, error: AppError):
        """触发错误回调"""
        callbacks = self.error_callbacks.get(error.code, [])
        for callback in callbacks:
            try:
                callback(error)
            except Exception as e:
                print(f"[ERROR] Callback failed: {e}", flush=True)

    def get_recent_errors(self, count: int = 10) -> list[dict]:
        """获取最近的错误（线程安全）"""
        with self._lock:
            return list(self.error_log[-count:])

    def clear_errors(self):
        """清除错误日志"""
        with self._lock:
            self.error_log.clear()

    def get_stats(self) -> dict:
        """获取错误统计"""
        with self._lock:
            total = len(self.error_log)
            by_code: dict[str, int] = {}
            for entry in self.error_log:
                code = entry["code"]
                by_code[code] = by_code.get(code, 0) + 1
            return {
                "total_errors": total,
                "by_code": by_code,
                "service_health": {
                    name: m.get_status()
                    for name, m in self.health_monitors.items()
                },
            }


def retry_on_error(
    max_retries: int = 3,
    delay: float = 1.0,
    backoff: float = 2.0,
    exceptions: tuple = (Exception,),
    health_monitor: Optional[ServiceHealth] = None,
):
    """增强重试装饰器 — 保指数退避 + 健康监控

    Args:
        max_retries: 最大重试次数
        delay: 初始延迟（秒）
        backoff: 退避倍数
        exceptions: 捕获的异常类型
        health_monitor: 可选健康监控器
    """
    def decorator(func):
        @wraps(func)
        def wrapper(*args, **kwargs):
            last_exception = None
            current_delay = delay

            for attempt in range(max_retries + 1):
                try:
                    result = func(*args, **kwargs)
                    if health_monitor:
                        health_monitor.record_success()
                    return result
                except exceptions as e:
                    last_exception = e
                    if health_monitor:
                        health_monitor.record_failure()

                    if attempt < max_retries:
                        print(f"[RETRY] {func.__name__} 尝试 {attempt + 1}/{max_retries} 失败: {e}", flush=True)
                        print(f"[RETRY] 等待 {current_delay:.1f}s...", flush=True)
                        time.sleep(current_delay)
                        current_delay *= backoff
                    else:
                        print(f"[RETRY] {func.__name__} 所有 {max_retries} 次尝试均失败", flush=True)

            raise last_exception

        return wrapper
    return decorator


def safe_execute(
    func: Callable,
    *args,
    default=None,
    error_handler: Optional[ErrorHandler] = None,
    context: str = "",
    **kwargs,
):
    """安全执行函数 — 捕获所有异常"""
    try:
        return func(*args, **kwargs)
    except Exception as e:
        if error_handler:
            error_handler.handle_error(e, context)
        else:
            print(f"[ERROR] {context}: {e}", flush=True)
        return default


# 全局错误处理器实例
error_handler = ErrorHandler(max_retries=3)
