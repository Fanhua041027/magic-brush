"""AI-Assistant Sidecar: HTTP server providing STT + KB search services.

支持优雅关闭（SIGINT/SIGTERM）和资源清理。
"""

import argparse
import atexit
import base64
import hmac
import os
import signal
import sys
import threading
import time

# Load local, gitignored credentials before reading environment configuration.
_env_file = os.path.join(os.path.dirname(__file__), ".env")
if os.path.exists(_env_file):
    for _line in open(_env_file, encoding="utf-8"):
        if "=" in _line and not _line.lstrip().startswith("#"):
            _key, _value = _line.strip().split("=", 1)
            os.environ.setdefault(_key, _value)

import numpy as np

from flask import Flask, jsonify, request

from audio import AudioRecorder, set_audio_level_callback
from transcribe import Transcriber
from knowledge_base import KnowledgeBase
from qwen_stt import QwenSTT
from qwen_asr_local import QwenASRLocal
from stt_manager import STTManager
from error_handler import ErrorHandler, AppError, ErrorCode, error_handler, safe_execute, retry_on_error

app = Flask(__name__)
app.config["MAX_CONTENT_LENGTH"] = 8 * 1024 * 1024

SIDECAR_TOKEN = os.environ.get("MAGIC_BRUSH_SIDECAR_TOKEN", "")
if not SIDECAR_TOKEN:
    raise RuntimeError("MAGIC_BRUSH_SIDECAR_TOKEN is required")

@app.before_request
def require_token():
    supplied = request.headers.get("Authorization", "")
    expected = "Bearer " + SIDECAR_TOKEN
    if not hmac.compare_digest(supplied, expected):
        return jsonify({"error": "unauthorized"}), 401

# Global state
recorder: AudioRecorder | None = None
kb: KnowledgeBase | None = None
stt_manager: STTManager | None = None
recording_lock = threading.RLock()
recording_operation_lock = threading.Lock()
is_recording = False
recording_generation = 0

# 流式转写结果（线程安全）
_streaming_results: list[str] = []
_streaming_results_lock = threading.Lock()

# 音频电平监控
_audio_level: float = 0.0
_audio_level_lock = threading.Lock()

# 千问 API Key — 优先从环境变量读取，其次硬编码
QWEN_API_KEY = os.environ.get("DASHSCOPE_API_KEY", "")
QWEN_BASE_URL = os.environ.get("QWEN_STT_BASE_URL", "https://ws-ghn8v2fudqm1a5bw.cn-beijing.maas.aliyuncs.com/compatible-mode/v1")
QWEN_STT_MODEL = os.environ.get("QWEN_STT_MODEL", "qwen-audio-3.0-asr-flash-streaming")


# ── 关闭与资源管理 ─────────────────────────────────────────

_shutting_down = False


@app.route("/api/shutdown", methods=["POST"])
def api_shutdown():
    """优雅关闭服务"""
    global _shutting_down
    _shutting_down = True
    print("[Sidecar] 正在优雅关闭...", flush=True)
    cleanup_resources()

    def delayed_shutdown():
        time.sleep(0.5)
        os._exit(0)

    threading.Thread(target=delayed_shutdown, daemon=True).start()
    return jsonify({"status": "shutting_down"})


@app.route("/api/cleanup", methods=["POST"])
def api_cleanup():
    """清理资源但不关闭"""
    cleanup_resources()
    return jsonify({"status": "cleaned"})


def cleanup_resources():
    """清理所有资源（线程安全）"""
    global recorder, kb, is_recording, recording_generation
    with recording_operation_lock:
        with recording_lock:
            recording_generation += 1
            is_recording = False
            rec = recorder
            manager = stt_manager
            recorder = None
            kb = None
            if rec is not None:
                try:
                    rec.set_streaming_callback(None)
                except Exception:
                    pass
        if manager is not None:
            try:
                manager.stop_streaming()
            except Exception:
                pass
        if rec is not None:
            try:
                rec.close()
                print("[Sidecar] Audio recorder closed", flush=True)
            except Exception as e:
                print(f"[Sidecar] Recorder close error: {e}", flush=True)
    with _streaming_results_lock:
        _streaming_results.clear()
    print("[Sidecar] Resources cleaned up", flush=True)


def signal_handler(signum, frame):
    """Signal handler for graceful shutdown"""
    global _shutting_down
    if _shutting_down:
        print("[Sidecar] Force exit...", flush=True)
        os._exit(1)
    _shutting_down = True
    print(f"[Sidecar] Signal {signum}, shutting down...", flush=True)
    cleanup_resources()
    os._exit(0)


# Register signal handlers
signal.signal(signal.SIGINT, signal_handler)
signal.signal(signal.SIGTERM, signal_handler)
atexit.register(cleanup_resources)


# ── Health ──────────────────────────────────────────────────────────────

@app.route("/api/health", methods=["GET"])
def health():
    stt_ready = stt_manager is not None and stt_manager.is_any_ready()
    primary = stt_manager.get_primary_service() if stt_manager else None
    available = stt_manager.get_available_services() if stt_manager else []
    service_names = {s: STTManager.SERVICE_NAMES.get(s, s) for s in available} if stt_manager else {}

    return jsonify({
        "status": "ok",
        "stt_ready": stt_ready,
        "stt_primary": primary,
        "stt_available": available,
        "stt_service_names": service_names,
        "stt_usage": stt_manager.get_usage_stats() if stt_manager else {},
        "kb_ready": kb is not None and kb.ready,
        "errors": len(error_handler.error_log),
        "audio": recorder.status() if recorder is not None else {"ready": False},
    })


@app.route("/api/errors", methods=["GET"])
def get_errors():
    """获取最近的错误"""
    count = request.args.get("count", 10, type=int)
    count = max(1, min(count or 10, 100))
    return jsonify({"errors": error_handler.get_recent_errors(count)})


@app.route("/api/errors/clear", methods=["POST"])
def clear_errors():
    """清除错误日志"""
    error_handler.clear_errors()
    return jsonify({"status": "ok"})


# ── 音频电平监控 ─────────────────────────────────────────────

@app.route("/api/audio/level", methods=["GET"])
def audio_level():
    """获取当前音频输入电平（VU 表用）"""
    with _audio_level_lock:
        return jsonify({"level": _audio_level})


# ── STT (Speech-to-Text) ───────────────────────────────────────────────

@app.route("/api/stt/devices", methods=["GET"])
def stt_devices():
    from audio import list_input_devices, get_device_config
    devices = list_input_devices()
    current_device, current_rate = get_device_config()
    return jsonify({
        "devices": devices,
        "current_device_id": current_device,
        "current_sample_rate": current_rate,
        "audio": recorder.status() if recorder is not None else {"ready": False},
    })


@app.route("/api/stt/device", methods=["POST"])
def stt_set_device():
    global recorder
    data = request.get_json(silent=True) or {}
    device_id = data.get("device_id")
    device_name = data.get("device_name")
    backend = data.get("backend")
    speaker_id = data.get("speaker_id")

    if device_id is not None:
        try:
            device_id = int(device_id)
            if device_id < 0 and not speaker_id:
                from audio import list_input_devices
                selected = next((item for item in list_input_devices() if item.get("id") == device_id), None)
                if selected and selected.get("is_loopback"):
                    backend = "soundcard_loopback"
                    speaker_id = selected.get("speaker_id")
                    device_id = None
        except (ValueError, TypeError):
            return jsonify({"error": "Invalid device_id"}), 400

    with recording_lock:
        if recorder is None:
            return jsonify({"error": "Audio recorder not initialized"}), 500
        if is_recording:
            return jsonify({"error": "Cannot change device while recording"}), 409
        try:
            if not recorder.reopen(device_id, device_name, backend, speaker_id):
                return jsonify({"error": "Audio source could not be opened"}), 500
            from audio import get_device_config
            curr_dev, curr_rate = get_device_config()
            return jsonify({"status": "ok", "device_id": curr_dev, "sample_rate": curr_rate})
        except Exception:
            return jsonify({"error": "Audio source could not be changed"}), 500


@app.route("/api/stt/status", methods=["GET"])
def stt_status():
    with recording_lock:
        recording = is_recording
    return jsonify({"recording": recording})


@app.route("/api/stt/start", methods=["POST"])
def stt_start():
    global is_recording, recording_generation
    if recorder is None:
        error = AppError(
            code=ErrorCode.AUDIO_INIT_FAILED,
            message="Audio recorder not initialized",
            recoverable=False,
        )
        return jsonify(error.to_dict()), 500
    with recording_lock:
        if is_recording:
            error = AppError(
                code=ErrorCode.RESOURCE_BUSY,
                message="Already recording",
                recoverable=True,
            )
            return jsonify(error.to_dict()), 409
        try:
            recording_generation += 1
            with _streaming_results_lock:
                _streaming_results.clear()
            recorder.set_streaming_callback(None)
            recorder.start_recording()
            is_recording = True
        except Exception as e:
            error = error_handler.handle_error(e, "stt_start")
            return jsonify(error.to_dict()), 500
    return jsonify({"status": "recording"})


@app.route("/api/stt/start-streaming", methods=["POST"])
def stt_start_streaming():
    """开始流式转写（使用 STTManager）"""
    global is_recording, recording_generation
    with recording_operation_lock:
        with recording_lock:
            rec = recorder
            manager = stt_manager
            if rec is None:
                error = AppError(
                    code=ErrorCode.AUDIO_INIT_FAILED,
                    message="Audio recorder not initialized",
                    recoverable=False,
                )
                return jsonify(error.to_dict()), 500
            if manager is None or not manager.is_any_ready():
                error = AppError(
                    code=ErrorCode.TRANSCRIPTION_FAILED,
                    message="No STT service available",
                    recoverable=False,
                )
                return jsonify(error.to_dict()), 500
            if is_recording:
                error = AppError(
                    code=ErrorCode.RESOURCE_BUSY,
                    message="Already recording",
                    recoverable=True,
                )
                return jsonify(error.to_dict()), 409
            recording_generation += 1
            generation = recording_generation
            is_recording = True

        with _streaming_results_lock:
            _streaming_results.clear()

        def streaming_callback(audio_chunk):
            """将音频块送入 STTManager 的流式引擎"""
            try:
                with recording_lock:
                    active = is_recording and recording_generation == generation
                if active and audio_chunk.size > 0:
                    manager.add_audio_chunk(audio_chunk)
            except Exception as e:
                error_handler.handle_error(e, "streaming_callback")

        def on_streaming_result(text):
            if not text.strip():
                return
            with recording_lock:
                active = is_recording and recording_generation == generation
            if active:
                with _streaming_results_lock:
                    _streaming_results.append(text)

        rec.set_streaming_callback(streaming_callback)
        capture_started = False
        stream_started = False
        try:
            rec.start_recording()
            capture_started = True
            service = manager.start_streaming(on_streaming_result, sample_rate=16000)
            if not service:
                raise RuntimeError("No streaming STT service available")
            stream_started = True
            print(f"[STT] Streaming started via STTManager ({service})", flush=True)
            return jsonify({"status": "recording"})
        except Exception as e:
            with recording_lock:
                if recording_generation == generation:
                    is_recording = False
                    recording_generation += 1
            try:
                rec.set_streaming_callback(None)
            except Exception:
                pass
            if capture_started:
                try:
                    rec.stop_recording()
                except Exception:
                    pass
            if stream_started:
                try:
                    manager.stop_streaming()
                except Exception:
                    pass
            with _streaming_results_lock:
                _streaming_results.clear()
            error = error_handler.handle_error(e, "stt_start_streaming")
            return jsonify(error.to_dict()), 503 if not stream_started else 500


@app.route("/api/stt/streaming-results", methods=["GET"])
def stt_streaming_results():
    """获取流式转写结果（线程安全）"""
    global _streaming_results
    with _streaming_results_lock:
        results = list(_streaming_results)
        _streaming_results.clear()
    return jsonify({"results": results})


@app.route("/api/stt/stop", methods=["POST"])
def stt_stop():
    global is_recording, recording_generation
    if recorder is None:
        error = AppError(
            code=ErrorCode.AUDIO_INIT_FAILED,
            message="Audio recorder not initialized",
            recoverable=False,
        )
        return jsonify(error.to_dict()), 500

    if stt_manager is None or not stt_manager.is_any_ready():
        error = AppError(
            code=ErrorCode.TRANSCRIPTION_FAILED,
            message="No STT service available",
            recoverable=False,
        )
        return jsonify(error.to_dict()), 500

    with recording_operation_lock:
        with recording_lock:
            if not is_recording:
                error = AppError(
                    code=ErrorCode.RESOURCE_BUSY,
                    message="Not recording",
                    recoverable=True,
                )
                return jsonify(error.to_dict()), 409
            rec = recorder
            manager = stt_manager
            is_recording = False
            recording_generation += 1
            try:
                rec.set_streaming_callback(None)
            except Exception:
                pass
        try:
            # Join capture workers outside recording_lock; callbacks may need it.
            audio = rec.stop_recording()
            streaming_text, streaming_service = manager.stop_streaming()
            if streaming_text.strip():
                with _streaming_results_lock:
                    _streaming_results.append(streaming_text)
        except Exception as e:
            error = error_handler.handle_error(e, "stt_stop")
            return jsonify(error.to_dict()), 500

    # 获取最终转写结果  — 使用 STTManager 自动备份降级
    text = ""
    with _streaming_results_lock:
        if _streaming_results:
            text = "".join(_streaming_results)
            _streaming_results.clear()
    if not text and audio.size > 0:
        text, used_service = manager.recognize(audio, sample_rate=16000)
        if text:
            print(f"[STT] ✅ 识别成功 (服务: {STTManager.SERVICE_NAMES.get(used_service, used_service)})", flush=True)

    print(f"[STT] stop: text_length={len(text)}", flush=True)
    return jsonify({"text": text, "service": manager.get_primary_service() if manager else None})


@app.route("/api/stt/record", methods=["POST"])
def stt_record():
    """Record until silence, then transcribe using STTManager."""
    global is_recording, recording_generation
    with recording_lock:
        if recorder is None:
            return jsonify({"error": "Audio recorder not initialized"}), 500
        if stt_manager is None or not stt_manager.is_any_ready():
            return jsonify({"error": "No STT service available"}), 500
        if is_recording:
            return jsonify({"error": "Already recording"}), 409

        data = request.get_json(silent=True) or {}
        max_seconds = data.get("max_seconds", 30)
        try:
            max_seconds = float(max_seconds)
            if not 1 <= max_seconds <= 120:
                raise ValueError
        except (ValueError, TypeError):
            return jsonify({"error": "max_seconds must be between 1 and 120"}), 400
        is_recording = True
        recording_generation += 1
        try:
            audio = recorder.record_until_silence(max_seconds=max_seconds)
            if audio.size == 0:
                return jsonify({"text": ""})
            text, used_service = stt_manager.recognize(audio, sample_rate=16000)
            return jsonify({"text": text, "service": used_service})
        except Exception as e:
            error = error_handler.handle_error(e, "stt_record")
            return jsonify(error.to_dict()), 500
        finally:
            is_recording = False
            recording_generation += 1


# ── 系统音频转写 ───────────────────────────────────

@app.route("/api/stt/transcribe", methods=["POST"])
def stt_transcribe():
    """接收 base64 WAV 音频，使用 STTManager 转写"""
    if stt_manager is None or not stt_manager.is_any_ready():
        return jsonify({"text": "", "error": "STT unavailable"}), 400
    data = request.get_json(silent=True) or {}
    audio_b64 = data.get("audio", "")
    if not isinstance(audio_b64, str) or not audio_b64:
        return jsonify({"text": "", "error": "No audio"}), 400
    if len(audio_b64) > 7 * 1024 * 1024:
        return jsonify({"text": "", "error": "Audio payload too large"}), 413
    try:
        raw = base64.b64decode(audio_b64, validate=True)
        if len(raw) > 5 * 1024 * 1024:
            return jsonify({"text": "", "error": "Audio payload too large"}), 413
        import io, soundfile as sf
        audio_arr, rate = sf.read(io.BytesIO(raw), dtype="float32", always_2d=True)
        from audio import _canonicalize
        audio_arr = _canonicalize(audio_arr, rate)
        text, used = stt_manager.recognize(audio_arr, sample_rate=16000)
        return jsonify({"status": "transcribed", "text": text, "service": used})
    except Exception as e:
        err = error_handler.handle_error(e, "stt_transcribe")
        return jsonify({"text": "", "error": str(e)}), 400


# ── KB (Knowledge Base) ─────────────────────────────────────────────

@app.route("/api/kb/info", methods=["GET"])
def kb_info():
    if kb is None or not kb.ready:
        return jsonify({"ready": False, "file_count": 0, "section_count": 0, "kb_path": ""})
    return jsonify({
        "ready": True,
        "file_count": getattr(kb, 'file_count', 0),
        "section_count": len(kb.chunks),
        "kb_path": kb.kb_path or "",
    })

@app.route("/api/kb/search", methods=["POST"])
def kb_search():
    global kb
    if kb is None or not kb.ready:
        error = AppError(
            code=ErrorCode.KB_LOAD_FAILED,
            message="KB not loaded",
            recoverable=True,
        )
        return jsonify(error.to_dict()), 400
    data = request.get_json(silent=True) or {}
    query = data.get("query", "")
    top_k = data.get("top_k", 5)
    if not isinstance(query, str) or len(query) > 2000:
        return jsonify({"error": "query is invalid or too long"}), 400
    try:
        top_k = max(1, min(int(top_k), 20))
    except (TypeError, ValueError):
        return jsonify({"error": "top_k is invalid"}), 400
    if not query:
        return jsonify({"results": []})
    try:
        results = kb.search(query, top_k=top_k)
        return jsonify({"results": results})
    except Exception as e:
        error = error_handler.handle_error(e, "kb_search")
        return jsonify(error.to_dict()), 500


@app.route("/api/kb/load", methods=["POST"])
def kb_load():
    global kb
    data = request.get_json(silent=True) or {}
    path = data.get("path", "")
    if not isinstance(path, str) or len(path) > 4096 or not path or not os.path.isdir(path):
        error = AppError(
            code=ErrorCode.INVALID_INPUT,
            message="Invalid knowledge base path",
            recoverable=True,
        )
        return jsonify(error.to_dict()), 400
    try:
        if kb is None:
            kb = KnowledgeBase()
        result = kb.load(path)
        return jsonify({"status": "ok", **result})
    except Exception as e:
        error = error_handler.handle_error(e, "kb_load")
        return jsonify(error.to_dict()), 500


# ── Main ───────────────────────────────────────────────────────────────

def main():
    global recorder, stt_manager

    parser = argparse.ArgumentParser(description="AI-Assistant Sidecar")
    parser.add_argument("--port", type=int, default=18765, help="HTTP port")
    parser.add_argument("--model", default="medium", help="Whisper model size")
    parser.add_argument("--device", default="auto", help="Device: auto/cuda/cpu")
    parser.add_argument("--language", default="zh", help="Language: zh/en/auto")
    parser.add_argument("--sensitivity", type=float, default=0.5, help="Sensitivity: 0.0-1.0")
    parser.add_argument("--stt", default="qwen_cloud", choices=["qwen_cloud", "qwen_local", "local_whisper"],
                        help="STT service: qwen_cloud(仅云端) / qwen_local(本地+云端备份) / local_whisper")
    args = parser.parse_args()

    print(f"[Sidecar] Initializing STT service chain...")
    print(f"[Sidecar]    模型: {args.model}, 设备: {args.device}, 语言: {args.language}")

    # 根据 --stt 参数构建优先级列表
    priority_map = {
        "qwen_cloud": ["qwen_cloud"],                         # 仅云端
        "qwen_local": ["qwen_local", "qwen_cloud", "local_whisper"],  # 本地->云端->Whisper
        "local_whisper": ["local_whisper"],                    # 仅 Whisper
    }
    priority = priority_map.get(args.stt, ["qwen_cloud"])

    print(f"[Sidecar]    优先级: {' -> '.join(priority)}")

    # 使用 STTManager 初始化服务（按优先级依次加载）
    stt_manager = STTManager(
        api_key=QWEN_API_KEY,
        whisper_model=args.model,
        whisper_device=args.device,
        whisper_language=args.language,
        priority=priority,
        qwen_base_url=QWEN_BASE_URL,
        qwen_model=QWEN_STT_MODEL,
    )
    loaded = stt_manager.initialize_all()

    if loaded:
        chain = " -> ".join(STTManager.SERVICE_NAMES[s] for s in stt_manager.get_available_services())
        print(f"[Sidecar] [OK] STT chain: {chain}")
    else:
        print(f"[Sidecar] [FAIL] No STT service available")

    # 初始化录音器
    try:
        recorder = AudioRecorder()
        # 注册 VU 表电平回调
        def on_audio_level(rms: float):
            global _audio_level
            with _audio_level_lock:
                _audio_level = rms
        set_audio_level_callback(on_audio_level)
        print("[Sidecar] [OK] Audio recorder ready")
    except Exception as e:
        error = error_handler.handle_error(e, "recorder_init")
        print(f"[Sidecar] [FAIL] Audio recorder init failed: {error.message}")
        recorder = None

    print(f"[Sidecar] Starting HTTP server on port {args.port}")
    app.run(host="127.0.0.1", port=args.port, debug=False, threaded=True)


if __name__ == "__main__":
    main()
