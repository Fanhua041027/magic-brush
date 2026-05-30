"""Error handling module for Magic Brush Sidecar."""

import traceback
import sys
from datetime import datetime
from enum import Enum
from typing import Optional, Callable
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
    INVALID_INPUT = "INVALID_INPUT"
    TIMEOUT = "TIMEOUT"
    RESOURCE_BUSY = "RESOURCE_BUSY"


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


class ErrorHandler:
    """错误处理器"""

    def __init__(self, max_retries: int = 3):
        self.max_retries = max_retries
        self.error_log: list[dict] = []
        self.error_callbacks: dict[ErrorCode, list[Callable]] = {}

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

        # 根据错误类型确定错误代码
        code = ErrorCode.UNKNOWN
        recoverable = True

        if "audio" in error_msg.lower() or "sounddevice" in error_msg.lower():
            code = ErrorCode.AUDIO_INIT_FAILED
        elif "whisper" in error_msg.lower() or "model" in error_msg.lower():
            code = ErrorCode.MODEL_LOAD_FAILED
        elif "timeout" in error_msg.lower():
            code = ErrorCode.TIMEOUT
        elif "network" in error_msg.lower() or "connection" in error_msg.lower():
            code = ErrorCode.NETWORK_ERROR
        elif "memory" in error_msg.lower() or "resource" in error_msg.lower():
            code = ErrorCode.RESOURCE_BUSY
            recoverable = False

        details = f"{error_type}: {error_msg}\n{traceback.format_exc()}"

        return AppError(
            code=code,
            message=f"{context}: {error_msg}" if context else error_msg,
            details=details,
            recoverable=recoverable,
        )

    def _log_error(self, error: AppError, context: str):
        """记录错误日志"""
        log_entry = {
            "timestamp": error.timestamp,
            "code": error.code.value,
            "message": error.message,
            "context": context,
            "recoverable": error.recoverable,
        }
        self.error_log.append(log_entry)

        # 保持日志大小
        if len(self.error_log) > 1000:
            self.error_log = self.error_log[-500:]

        # 打印到控制台
        print(f"[ERROR] [{error.code.value}] {error.message}")
        if error.details:
            print(f"[ERROR] Details: {error.details[:200]}...")

    def _trigger_callbacks(self, error: AppError):
        """触发错误回调"""
        callbacks = self.error_callbacks.get(error.code, [])
        for callback in callbacks:
            try:
                callback(error)
            except Exception as e:
                print(f"[ERROR] Callback failed: {e}")

    def get_recent_errors(self, count: int = 10) -> list[dict]:
        """获取最近的错误"""
        return self.error_log[-count:]

    def clear_errors(self):
        """清除错误日志"""
        self.error_log.clear()


def retry_on_error(
    max_retries: int = 3,
    delay: float = 1.0,
    backoff: float = 2.0,
    exceptions: tuple = (Exception,),
):
    """重试装饰器"""
    def decorator(func):
        @wraps(func)
        def wrapper(*args, **kwargs):
            last_exception = None
            current_delay = delay

            for attempt in range(max_retries + 1):
                try:
                    return func(*args, **kwargs)
                except exceptions as e:
                    last_exception = e
                    if attempt < max_retries:
                        print(f"[RETRY] Attempt {attempt + 1}/{max_retries} failed: {e}")
                        print(f"[RETRY] Waiting {current_delay:.1f}s before retry...")
                        time.sleep(current_delay)
                        current_delay *= backoff
                    else:
                        print(f"[RETRY] All {max_retries} attempts failed")

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
    """安全执行函数"""
    try:
        return func(*args, **kwargs)
    except Exception as e:
        if error_handler:
            error_handler.handle_error(e, context)
        else:
            print(f"[ERROR] {context}: {e}")
        return default


# 全局错误处理器实例
error_handler = ErrorHandler()
