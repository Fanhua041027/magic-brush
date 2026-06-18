"""AI-Assistant Sidecar: HTTP server providing STT + KB search services."""

import argparse
import json
import os
import sys
import threading
import time

import numpy as np

from flask import Flask, jsonify, request
from flask_cors import CORS
from flask_sock import Sock

from audio import AudioRecorder
from transcribe import Transcriber
from knowledge_base import KnowledgeBase
from qwen_stt import QwenSTT
from qwen_asr_local import QwenASRLocal
from stt_manager import STTManager
from error_handler import ErrorHandler, AppError, ErrorCode, error_handler, safe_execute, retry_on_error

app = Flask(__name__)
CORS(app)  # 启用 CORS 支持
sock = Sock(app)

# Global state
recorder: AudioRecorder | None = None
kb: KnowledgeBase | None = None
stt_manager: STTManager | None = None  # STT 管理器（带自动备份）
recording_lock = threading.Lock()
is_recording = False

# WebSocket clients
ws_clients = set()

# 千问 API Key（硬编码）
QWEN_API_KEY = "sk-3ced1755eb8a44628ce5ff1e5789f4b7"


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
    })


@app.route("/api/errors", methods=["GET"])
def get_errors():
    """获取最近的错误"""
    count = request.args.get("count", 10, type=int)
    return jsonify({"errors": error_handler.get_recent_errors(count)})


@app.route("/api/errors/clear", methods=["POST"])
def clear_errors():
    """清除错误日志"""
    error_handler.clear_errors()
    return jsonify({"status": "ok"})


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
    })


@app.route("/api/stt/device", methods=["POST"])
def stt_set_device():
    global recorder
    data = request.get_json(silent=True) or {}
    device_id = data.get("device_id")
    device_name = data.get("device_name")  # 支持按名称查找设备

    if device_id is not None:
        try:
            device_id = int(device_id)
        except (ValueError, TypeError):
            return jsonify({"error": "Invalid device_id"}), 400

    if recorder is None:
        return jsonify({"error": "Audio recorder not initialized"}), 500
    try:
        from audio import init_audio
        # 初始化新设备
        init_audio(device_id, device_name)
        # 重新打开录音器
        recorder.reopen(device_id)
        from audio import get_device_config
        curr_dev, curr_rate = get_device_config()
        return jsonify({"status": "ok", "device_id": curr_dev, "sample_rate": curr_rate})
    except Exception as e:
        return jsonify({"error": str(e)}), 500


@app.route("/api/stt/status", methods=["GET"])
def stt_status():
    return jsonify({"recording": is_recording})


@app.route("/api/stt/start", methods=["POST"])
def stt_start():
    global is_recording
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
            recorder.start_recording()
            is_recording = True
        except Exception as e:
            error = error_handler.handle_error(e, "stt_start")
            return jsonify(error.to_dict()), 500
    return jsonify({"status": "recording"})


@app.route("/api/stt/start-streaming", methods=["POST"])
def stt_start_streaming():
    """开始流式转写（使用 STTManager）"""
    global is_recording
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

    with recording_lock:
        if is_recording:
            error = AppError(
                code=ErrorCode.RESOURCE_BUSY,
                message="Already recording",
                recoverable=True,
            )
            return jsonify(error.to_dict()), 409

        def streaming_callback(audio_chunk):
            """将音频块送入 STTManager 的流式引擎"""
            try:
                if audio_chunk.size > 0:
                    stt_manager.add_audio_chunk(audio_chunk)
            except Exception as e:
                error = error_handler.handle_error(e, "streaming_callback")
                broadcast_ws_message({
                    "type": "error",
                    "code": error.code.value,
                    "message": str(e),
                })

        recorder.set_streaming_callback(streaming_callback)
        try:
            recorder.start_recording()
            is_recording = True
            # 启动 STT 流式识别（STTManager 自动选择可用服务）
            def on_streaming_result(text):
                if text.strip():
                    broadcast_ws_message({"type": "stt-streaming", "text": text})
                    streaming_results.append(text)
            stt_manager.start_streaming(on_streaming_result, sample_rate=recorder.sample_rate)
            print("[STT] Streaming started via STTManager", flush=True)
        except Exception as e:
            error = error_handler.handle_error(e, "stt_start_streaming")
            return jsonify(error.to_dict()), 500
    return jsonify({"status": "recording"})


@sock.route("/ws")
def websocket_handler(ws):
    """WebSocket 处理器"""
    ws_clients.add(ws)
    print(f"[WebSocket] Client connected. Total: {len(ws_clients)}")
    try:
        while True:
            # 保持连接活跃
            data = ws.receive(timeout=30)
            if data is None:
                break
            # 处理客户端消息
            try:
                msg = json.loads(data)
                handle_ws_message(msg)
            except json.JSONDecodeError:
                pass
    except Exception as e:
        print(f"[WebSocket] Error: {e}")
    finally:
        ws_clients.discard(ws)
        print(f"[WebSocket] Client disconnected. Total: {len(ws_clients)}")


def broadcast_ws_message(message):
    """广播消息到所有 WebSocket 客户端"""
    dead_clients = set()
    for ws in ws_clients:
        try:
            ws.send(json.dumps(message))
        except Exception:
            dead_clients.add(ws)
    # 清理断开的连接
    ws_clients.difference_update(dead_clients)


def handle_ws_message(message):
    """处理 WebSocket 消息"""
    msg_type = message.get("type")
    if msg_type == "ping":
        broadcast_ws_message({"type": "pong"})
    elif msg_type == "start-streaming":
        # 触发流式转写
        pass


# 全局变量存储流式转写结果
streaming_results = []


@app.route("/api/stt/streaming-results", methods=["GET"])
def stt_streaming_results():
    """获取流式转写结果"""
    global streaming_results
    results = list(streaming_results)
    streaming_results.clear()
    return jsonify({"results": results})


@app.route("/api/stt/stop", methods=["POST"])
def stt_stop():
    global is_recording
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

    with recording_lock:
        if not is_recording:
            error = AppError(
                code=ErrorCode.RESOURCE_BUSY,
                message="Not recording",
                recoverable=True,
            )
            return jsonify(error.to_dict()), 409
        try:
            # 停止流式识别（如果已启动）
            streaming_text, streaming_service = stt_manager.stop_streaming()
            if streaming_text.strip():
                broadcast_ws_message({"type": "stt-streaming", "text": streaming_text})
                streaming_results.append(streaming_text)

            audio = recorder.stop_recording()
            is_recording = False
        except Exception as e:
            is_recording = False
            error = error_handler.handle_error(e, "stt_stop")
            return jsonify(error.to_dict()), 500

    # 获取最终转写结果  — 使用 STTManager 自动备份识别
    text = ""
    if streaming_results:
        text = "".join(streaming_results)
        streaming_results.clear()
    elif audio.size > 0:
        text, used_service = stt_manager.recognize(audio, sample_rate=recorder.sample_rate)
        if text:
            print(f"[STT] ✅ 识别成功 (服务: {STTManager.SERVICE_NAMES.get(used_service, used_service)})", flush=True)

    print(f"[STT] stop: text={text!r}", flush=True)
    return jsonify({"text": text, "service": stt_manager.get_primary_service() if stt_manager else None})


@app.route("/api/stt/record", methods=["POST"])
def stt_record():
    """Record until silence, then transcribe. Returns transcribed text."""
    if recorder is None or transcriber is None:
        return jsonify({"error": "STT not initialized"}), 500
    data = request.get_json(silent=True) or {}
    max_seconds = data.get("max_seconds", 30)
    try:
        audio = recorder.record_until_silence(max_seconds=max_seconds)
        if audio.size == 0:
            return jsonify({"text": ""})
        text = transcriber.transcribe(audio)
        return jsonify({"text": text})
    except Exception as e:
        return jsonify({"error": str(e)}), 500


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
    if not path or not os.path.isdir(path):
        error = AppError(
            code=ErrorCode.INVALID_INPUT,
            message=f"Invalid path: {path}",
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
        print("[Sidecar] [OK] Audio recorder ready")
    except Exception as e:
        error = error_handler.handle_error(e, "recorder_init")
        print(f"[Sidecar] [FAIL] Audio recorder init failed: {error.message}")
        recorder = None

    print(f"[Sidecar] Starting HTTP server on port {args.port}")
    print(f"[Sidecar] WebSocket at ws://127.0.0.1:{args.port}/ws")
    app.run(host="127.0.0.1", port=args.port, debug=False, threaded=True)


if __name__ == "__main__":
    main()
