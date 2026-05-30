"""AI-Assistant Sidecar: HTTP server providing STT + KB search services."""

import argparse
import json
import os
import sys
import threading
import time

from flask import Flask, jsonify, request
from flask_cors import CORS
from flask_sock import Sock

from audio import AudioRecorder
from transcribe import Transcriber
from knowledge_base import KnowledgeBase
from qwen_stt import QwenSTT
from qwen_asr_local import QwenASRLocal
from error_handler import ErrorHandler, AppError, ErrorCode, error_handler, safe_execute, retry_on_error

app = Flask(__name__)
CORS(app)  # 启用 CORS 支持
sock = Sock(app)

# Global state
recorder: AudioRecorder | None = None
transcriber: Transcriber | None = None
kb: KnowledgeBase | None = None
qwen_stt: QwenSTT | None = None  # 千问云端语音识别服务
qwen_asr_local: QwenASRLocal | None = None  # 千问本地语音识别服务
recording_lock = threading.Lock()
is_recording = False
stt_service_type = "qwen_cloud"  # qwen_cloud, qwen_local, local_whisper
use_qwen_stt = True  # 是否使用千问 STT

# WebSocket clients
ws_clients = set()

# 千问 API Key（硬编码）
QWEN_API_KEY = "sk-3ced1755eb8a44628ce5ff1e5789f4b7"


# ── Health ──────────────────────────────────────────────────────────────

@app.route("/api/health", methods=["GET"])
def health():
    stt_ready = False
    stt_service = "none"

    if stt_service_type == "qwen_cloud" and qwen_stt is not None:
        stt_ready = True
        stt_service = "qwen_cloud"
    elif stt_service_type == "qwen_local" and qwen_asr_local is not None and qwen_asr_local.is_ready:
        stt_ready = True
        stt_service = "qwen_local"
    elif stt_service_type == "local_whisper" and transcriber is not None:
        stt_ready = True
        stt_service = "local_whisper"

    return jsonify({
        "status": "ok",
        "stt_ready": stt_ready,
        "stt_service": stt_service,
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
    """开始流式转写"""
    global is_recording
    if recorder is None:
        error = AppError(
            code=ErrorCode.AUDIO_INIT_FAILED,
            message="Audio recorder not initialized",
            recoverable=False,
        )
        return jsonify(error.to_dict()), 500

    # 检查 STT 服务是否可用
    if use_qwen_stt and qwen_stt is None:
        error = AppError(
            code=ErrorCode.TRANSCRIPTION_FAILED,
            message="Qwen STT not initialized",
            recoverable=False,
        )
        return jsonify(error.to_dict()), 500
    elif not use_qwen_stt and transcriber is None:
        error = AppError(
            code=ErrorCode.TRANSCRIPTION_FAILED,
            message="Local STT not initialized",
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
            """流式转写回调"""
            try:
                if audio_chunk.size > 0:
                    text = ""
                    if stt_service_type == "qwen_local" and qwen_asr_local:
                        # 使用千问本地 ASR
                        text = qwen_asr_local.recognize(audio_chunk)
                    elif stt_service_type == "qwen_cloud" and qwen_stt:
                        # 使用千问云端 STT
                        text = qwen_stt.recognize(audio_chunk)
                    elif transcriber:
                        # 使用本地 Whisper
                        text = transcriber.transcribe(audio_chunk)

                    if text.strip():
                        # 通过 WebSocket 发送转写结果
                        broadcast_ws_message({
                            "type": "stt-streaming",
                            "text": text
                        })
                        # 也存储到全局变量（兼容旧版本）
                        streaming_results.append(text)
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

    # 检查 STT 服务是否可用
    if use_qwen_stt and qwen_stt is None:
        error = AppError(
            code=ErrorCode.TRANSCRIPTION_FAILED,
            message="Qwen STT not initialized",
            recoverable=False,
        )
        return jsonify(error.to_dict()), 500
    elif not use_qwen_stt and transcriber is None:
        error = AppError(
            code=ErrorCode.TRANSCRIPTION_FAILED,
            message="Local STT not initialized",
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
            audio = recorder.stop_recording()
            is_recording = False
        except Exception as e:
            is_recording = False
            error = error_handler.handle_error(e, "stt_stop")
            return jsonify(error.to_dict()), 500

    if audio.size == 0:
        return jsonify({"text": ""})

    try:
        text = ""
        if stt_service_type == "qwen_local" and qwen_asr_local:
            # 使用千问本地 ASR
            text = qwen_asr_local.recognize(audio)
        elif stt_service_type == "qwen_cloud" and qwen_stt:
            # 使用千问云端 STT
            text = qwen_stt.recognize(audio)
        elif transcriber:
            # 使用本地 Whisper
            text = transcriber.transcribe(audio)

        return jsonify({"text": text})
    except Exception as e:
        error = error_handler.handle_error(e, "transcribe")
        # 如果失败，尝试其他服务
        if stt_service_type != "local_whisper" and transcriber:
            try:
                text = transcriber.transcribe(audio)
                return jsonify({"text": text})
            except Exception as e2:
                error2 = error_handler.handle_error(e2, "transcribe_fallback")
                return jsonify(error2.to_dict()), 500
        return jsonify(error.to_dict()), 500


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
    global recorder, transcriber, qwen_stt, qwen_asr_local, stt_service_type, use_qwen_stt

    parser = argparse.ArgumentParser(description="AI-Assistant Sidecar")
    parser.add_argument("--port", type=int, default=18765, help="HTTP port")
    parser.add_argument("--model", default="base", help="Whisper model size")
    parser.add_argument("--device", default="auto", help="Device: auto/cuda/cpu")
    parser.add_argument("--language", default="zh", help="Language: zh/en/auto")
    parser.add_argument("--sensitivity", type=float, default=0.5, help="Sensitivity: 0.0-1.0")
    parser.add_argument("--stt", default="local_whisper", choices=["qwen_cloud", "qwen_local", "local_whisper"],
                        help="STT service: qwen_cloud/qwen_local/local_whisper")
    args = parser.parse_args()

    print(f"[Sidecar] Initializing STT (service={args.stt}, model={args.model}, device={args.device}, language={args.language})...")

    # 初始化千问本地 ASR
    if args.stt == "qwen_local":
        try:
            print(f"[Sidecar] Initializing Qwen3-ASR-Flash local model...")
            qwen_asr_local = QwenASRLocal("Qwen/Qwen3-ASR-Flash")
            if qwen_asr_local.load_model():
                stt_service_type = "qwen_local"
                use_qwen_stt = False
                print(f"[Sidecar] Qwen3-ASR-Flash initialized successfully")
            else:
                print("[Sidecar] Qwen3-ASR-Flash load failed, falling back to cloud...")
                stt_service_type = "qwen_cloud"
                use_qwen_stt = True
        except Exception as e:
            error = error_handler.handle_error(e, "qwen_asr_local_init")
            print(f"[Sidecar] Qwen3-ASR-Flash init failed: {error.message}")
            stt_service_type = "qwen_cloud"
            use_qwen_stt = True

    # 初始化千问云端 STT
    if stt_service_type == "qwen_cloud" or args.stt == "qwen_cloud":
        try:
            print(f"[Sidecar] Initializing Qwen Cloud STT with API Key: {QWEN_API_KEY[:10]}...")
            qwen_stt = QwenSTT(QWEN_API_KEY)
            stt_service_type = "qwen_cloud"
            use_qwen_stt = True
            print(f"[Sidecar] Qwen Cloud STT initialized successfully")
        except Exception as e:
            error = error_handler.handle_error(e, "qwen_stt_init")
            print(f"[Sidecar] Qwen Cloud STT init failed: {error.message}")
            stt_service_type = "local_whisper"
            use_qwen_stt = False

    # 初始化本地 Whisper（作为备选）
    if stt_service_type == "local_whisper" or args.stt == "local_whisper":
        try:
            transcriber = Transcriber(model_name=args.model, device=args.device, language=args.language)
            stt_service_type = "local_whisper"
            use_qwen_stt = False
            print(f"[Sidecar] Local Whisper Transcriber ready (device={transcriber.device})")
        except Exception as e:
            error = error_handler.handle_error(e, "transcriber_init")
            print(f"[Sidecar] Local Whisper init failed: {error.message}")

    # 初始化录音器
    try:
        recorder = AudioRecorder()
        print("[Sidecar] Audio recorder ready")
    except Exception as e:
        error = error_handler.handle_error(e, "recorder_init")
        print(f"[Sidecar] Audio recorder init failed: {error.message}")

    stt_service_names = {
        "qwen_cloud": "千问云端 Qwen3-ASR-Flash",
        "qwen_local": "千问本地 Qwen3-ASR-Flash",
        "local_whisper": "本地 Whisper"
    }
    stt_service = stt_service_names.get(stt_service_type, "Unknown")
    print(f"[Sidecar] STT Service: {stt_service}")
    print(f"[Sidecar] Starting HTTP server on port {args.port}")
    print(f"[Sidecar] WebSocket available at ws://127.0.0.1:{args.port}/ws")
    app.run(host="127.0.0.1", port=args.port, debug=False)


if __name__ == "__main__":
    main()
