"""AI-Assistant Sidecar: HTTP server providing STT + KB search services."""

from __future__ import annotations

import argparse
import json
import os
import threading
from typing import Callable

import numpy as np
from flask import Flask, jsonify, request
from flask_cors import CORS
from flask_sock import Sock

from audio import AudioRecorder, WHISPER_RATE
from error_handler import AppError, ErrorCode, error_handler
from knowledge_base import KnowledgeBase
from qwen3_asr_flash_api import DEFAULT_BASE_URL, Qwen3ASRFlashAPI
from qwen_asr_local import QwenASRLocal
from qwen_stt import QwenSTT
from stt_flow import build_final_recognition_chain, normalize_stt_service
from transcribe import Transcriber


app = Flask(__name__)
CORS(app)
sock = Sock(app)


recorder: AudioRecorder | None = None
transcriber: Transcriber | None = None
kb: KnowledgeBase | None = None
qwen_stt: QwenSTT | None = None
qwen_asr_local: QwenASRLocal | None = None
qwen3_flash_api: Qwen3ASRFlashAPI | None = None

recording_lock = threading.Lock()
is_recording = False
stt_service_type = "qwen_cloud"
stt_language = "zh"

ws_clients: set = set()
streaming_results: list[str] = []

# Leave the API key hardcoded here for now so only one location needs editing.
QWEN_API_KEY = "sk-55dc9df537ad47b8b582d6fa937d032f"
QWEN_API_BASE_URL = DEFAULT_BASE_URL


def _safe_log(message: str) -> None:
    try:
        print(message, flush=True)
        return
    except UnicodeEncodeError:
        pass
    except Exception:
        return

    try:
        fallback = message.encode("ascii", "backslashreplace").decode("ascii")
        print(fallback, flush=True)
    except Exception:
        return


def _provider_ready(provider: str) -> bool:
    provider = normalize_stt_service(provider)
    if provider == "qwen3_asr_flash":
        return qwen3_flash_api is not None and qwen3_flash_api.is_enabled()
    if provider == "qwen_cloud":
        return qwen_stt is not None
    if provider == "qwen_local":
        return qwen_asr_local is not None and qwen_asr_local.is_ready
    if provider == "local_whisper":
        return transcriber is not None
    return False


def _emit_error(message: str, context: str) -> None:
    error = error_handler.handle_error(RuntimeError(message), context)
    broadcast_ws_message(
        {
            "type": "error",
            "code": error.code.value,
            "message": error.message,
        }
    )


def _append_streaming_text(text: str) -> None:
    cleaned = (text or "").strip()
    if not cleaned:
        return
    streaming_results.append(cleaned)
    broadcast_ws_message({"type": "stt-streaming", "text": cleaned})


def _transcribe_stream_chunk(audio_chunk: np.ndarray) -> None:
    try:
        if audio_chunk.size == 0:
            return
        if stt_service_type == "qwen_cloud" and qwen_stt:
            qwen_stt.add_audio_chunk(audio_chunk)
            return
        if stt_service_type == "qwen_local" and qwen_asr_local:
            _append_streaming_text(qwen_asr_local.recognize(audio_chunk, sample_rate=recorder.sample_rate if recorder else 16000))
            return
        if transcriber:
            _append_streaming_text(transcriber.transcribe(audio_chunk))
    except Exception as exc:
        error = error_handler.handle_error(exc, "streaming_callback")
        broadcast_ws_message(
            {
                "type": "error",
                "code": error.code.value,
                "message": error.message,
            }
        )


def _start_selected_streaming() -> None:
    if stt_service_type == "qwen_cloud" and qwen_stt and recorder:
        qwen_stt.start_streaming(_append_streaming_text, sample_rate=recorder.sample_rate)


def _stop_selected_streaming() -> None:
    if stt_service_type == "qwen_cloud" and qwen_stt and qwen_stt._is_streaming:
        final_text = qwen_stt.stop_streaming()
        if final_text.strip():
            _append_streaming_text(final_text)


def _recognize_with_provider(provider: str, audio: np.ndarray, sample_rate: int) -> str:
    provider = normalize_stt_service(provider)
    if provider == "qwen3_asr_flash":
        if not qwen3_flash_api:
            return ""
        return qwen3_flash_api.recognize(audio, sample_rate=sample_rate, language=stt_language)
    if provider == "qwen_cloud":
        if not qwen_stt:
            return ""
        return qwen_stt.recognize(audio, sample_rate=sample_rate)
    if provider == "qwen_local":
        if not qwen_asr_local:
            return ""
        return qwen_asr_local.recognize(audio, sample_rate=sample_rate)
    if provider == "local_whisper":
        if not transcriber:
            return ""
        return transcriber.transcribe(audio)
    return ""


def _recognize_final_audio(audio: np.ndarray, sample_rate: int) -> str:
    if audio.size == 0:
        return ""

    last_error: Exception | None = None
    for provider in build_final_recognition_chain(stt_service_type):
        if provider != "qwen3_asr_flash" and not _provider_ready(provider):
            continue
        try:
            text = _recognize_with_provider(provider, audio, sample_rate)
            if text.strip():
                _safe_log(f"[STT] Final provider: {provider}")
                return text.strip()
        except Exception as exc:
            last_error = exc
            error = error_handler.handle_error(exc, f"final_transcribe_{provider}")
            print(f"[STT] Provider {provider} failed: {error.message}", flush=True)

    if last_error is not None:
        raise last_error
    return ""


def _initialize_qwen_cloud() -> bool:
    global qwen_stt
    if qwen_stt is not None:
        return True
    if not QWEN_API_KEY.strip():
        print("[Sidecar] Qwen Cloud STT skipped: API key is empty", flush=True)
        return False
    qwen_stt = QwenSTT(QWEN_API_KEY)
    print("[Sidecar] Qwen Cloud STT initialized", flush=True)
    return True


def _initialize_qwen_local() -> bool:
    global qwen_asr_local
    if qwen_asr_local is not None and qwen_asr_local.is_ready:
        return True
    qwen_asr_local = QwenASRLocal("Qwen/Qwen3-ASR-Flash")
    if not qwen_asr_local.load_model():
        return False
    print("[Sidecar] Qwen local STT initialized", flush=True)
    return True


def _initialize_local_whisper(model: str, device: str, language: str) -> bool:
    global transcriber
    if transcriber is not None:
        return True
    transcriber = Transcriber(model_name=model, device=device, language=language)
    print(f"[Sidecar] Local Whisper Transcriber ready (device={transcriber.device})", flush=True)
    return True


def _initialize_runtime_provider(requested_service: str, model: str, device: str, language: str) -> str:
    requested = normalize_stt_service(requested_service)
    for provider in build_final_recognition_chain(requested)[1:]:
        try:
            if provider == "qwen_cloud" and _initialize_qwen_cloud():
                return provider
            if provider == "qwen_local" and _initialize_qwen_local():
                return provider
            if provider == "local_whisper" and _initialize_local_whisper(model, device, language):
                return provider
        except Exception as exc:
            error = error_handler.handle_error(exc, f"init_{provider}")
            print(f"[Sidecar] {provider} init failed: {error.message}", flush=True)
    return requested


@app.errorhandler(Exception)
def handle_unexpected_exception(exc):
    error = error_handler.handle_error(exc, "unhandled_exception")
    return jsonify(error.to_dict()), 500


@app.route("/api/health", methods=["GET"])
def health():
    stt_ready = any(
        provider == "qwen3_asr_flash" and qwen3_flash_api is not None and qwen3_flash_api.is_enabled()
        or _provider_ready(provider)
        for provider in build_final_recognition_chain(stt_service_type)
    )
    return jsonify(
        {
            "status": "ok",
            "stt_ready": stt_ready,
            "stt_service": stt_service_type,
            "kb_ready": kb is not None and kb.ready,
            "errors": len(error_handler.error_log),
        }
    )


@app.route("/api/errors", methods=["GET"])
def get_errors():
    count = request.args.get("count", 10, type=int)
    return jsonify({"errors": error_handler.get_recent_errors(count)})


@app.route("/api/errors/clear", methods=["POST"])
def clear_errors():
    error_handler.clear_errors()
    return jsonify({"status": "ok"})


@app.route("/api/stt/devices", methods=["GET"])
def stt_devices():
    from audio import get_device_config, list_input_devices

    devices = list_input_devices()
    current_device, current_rate = get_device_config()
    return jsonify(
        {
            "devices": devices,
            "current_device_id": current_device,
            "current_sample_rate": current_rate,
        }
    )


@app.route("/api/stt/device", methods=["POST"])
def stt_set_device():
    global recorder
    data = request.get_json(silent=True) or {}
    device_id = data.get("device_id")
    device_name = data.get("device_name")
    if device_id is not None:
        try:
            device_id = int(device_id)
        except (ValueError, TypeError):
            return jsonify({"error": "Invalid device_id"}), 400

    if recorder is None:
        return jsonify({"error": "Audio recorder not initialized"}), 500
    try:
        from audio import get_device_config, init_audio

        init_audio(device_id, device_name)
        recorder.reopen(device_id)
        curr_dev, curr_rate = get_device_config()
        return jsonify({"status": "ok", "device_id": curr_dev, "sample_rate": curr_rate})
    except Exception as exc:
        return jsonify({"error": str(exc)}), 500


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
            streaming_results.clear()
            recorder.start_recording()
            is_recording = True
        except Exception as exc:
            error = error_handler.handle_error(exc, "stt_start")
            return jsonify(error.to_dict()), 500
    return jsonify({"status": "recording"})


@app.route("/api/stt/start-streaming", methods=["POST"])
def stt_start_streaming():
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

        recorder.set_streaming_callback(_transcribe_stream_chunk)
        try:
            streaming_results.clear()
            recorder.start_recording()
            is_recording = True
            _start_selected_streaming()
        except Exception as exc:
            error = error_handler.handle_error(exc, "stt_start_streaming")
            return jsonify(error.to_dict()), 500

    return jsonify({"status": "recording"})


@sock.route("/ws")
def websocket_handler(ws):
    ws_clients.add(ws)
    print(f"[WebSocket] Client connected. Total: {len(ws_clients)}", flush=True)
    try:
        while True:
            data = ws.receive(timeout=30)
            if data is None:
                break
            try:
                message = json.loads(data)
            except json.JSONDecodeError:
                continue
            handle_ws_message(message)
    finally:
        ws_clients.discard(ws)
        print(f"[WebSocket] Client disconnected. Total: {len(ws_clients)}", flush=True)


def broadcast_ws_message(message):
    dead_clients = set()
    for ws in ws_clients:
        try:
            ws.send(json.dumps(message))
        except Exception:
            dead_clients.add(ws)
    ws_clients.difference_update(dead_clients)


def handle_ws_message(message):
    msg_type = message.get("type")
    if msg_type == "ping":
        broadcast_ws_message({"type": "pong"})


@app.route("/api/stt/streaming-results", methods=["GET"])
def stt_streaming_results():
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

    with recording_lock:
        if not is_recording:
            error = AppError(
                code=ErrorCode.RESOURCE_BUSY,
                message="Not recording",
                recoverable=True,
            )
            return jsonify(error.to_dict()), 409
        try:
            _stop_selected_streaming()
            audio = recorder.stop_recording()
            is_recording = False
        except Exception as exc:
            is_recording = False
            error = error_handler.handle_error(exc, "stt_stop")
            return jsonify(error.to_dict()), 500

    try:
        text = _recognize_final_audio(audio, sample_rate=WHISPER_RATE)
    except Exception as exc:
        error = error_handler.handle_error(exc, "stt_final_recognition")
        return jsonify(error.to_dict()), 500

    _safe_log(f"[STT] stop: text={text!r}")
    return jsonify({"text": text})


@app.route("/api/stt/record", methods=["POST"])
def stt_record():
    if recorder is None:
        return jsonify({"error": "Audio recorder not initialized"}), 500
    data = request.get_json(silent=True) or {}
    max_seconds = data.get("max_seconds", 30)
    try:
        audio = recorder.record_until_silence(max_seconds=max_seconds)
        if audio.size == 0:
            return jsonify({"text": ""})
        text = _recognize_final_audio(audio, sample_rate=WHISPER_RATE)
        return jsonify({"text": text})
    except Exception as exc:
        return jsonify({"error": str(exc)}), 500


@app.route("/api/kb/info", methods=["GET"])
def kb_info():
    if kb is None or not kb.ready:
        return jsonify({"ready": False, "file_count": 0, "section_count": 0, "kb_path": ""})
    return jsonify(
        {
            "ready": True,
            "file_count": getattr(kb, "file_count", 0),
            "section_count": len(kb.chunks),
            "kb_path": kb.kb_path or "",
        }
    )


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
    except Exception as exc:
        error = error_handler.handle_error(exc, "kb_search")
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
    except Exception as exc:
        error = error_handler.handle_error(exc, "kb_load")
        return jsonify(error.to_dict()), 500


def main():
    global recorder, qwen3_flash_api, stt_language, stt_service_type

    parser = argparse.ArgumentParser(description="AI-Assistant Sidecar")
    parser.add_argument("--port", type=int, default=18765, help="HTTP port")
    parser.add_argument("--model", default="medium", help="Whisper model size")
    parser.add_argument("--device", default="auto", help="Device: auto/cuda/cpu")
    parser.add_argument("--language", default="zh", help="Language: zh/en/auto")
    parser.add_argument("--sensitivity", type=float, default=0.5, help="Sensitivity: 0.0-1.0")
    parser.add_argument(
        "--stt",
        default="local_whisper",
        choices=["qwen", "qwen_cloud", "qwen_local", "local_whisper"],
        help="STT service: qwen_cloud/qwen_local/local_whisper",
    )
    args = parser.parse_args()

    stt_language = args.language
    requested_service = normalize_stt_service(args.stt)
    qwen3_flash_api = Qwen3ASRFlashAPI(api_key=QWEN_API_KEY, base_url=QWEN_API_BASE_URL)

    _safe_log(
        f"[Sidecar] Initializing STT (service={requested_service}, model={args.model}, device={args.device}, language={args.language})..."
    )
    stt_service_type = _initialize_runtime_provider(requested_service, args.model, args.device, args.language)

    try:
        recorder = AudioRecorder()
        _safe_log("[Sidecar] Audio recorder ready")
    except Exception as exc:
        error = error_handler.handle_error(exc, "recorder_init")
        _safe_log(f"[Sidecar] Audio recorder init failed: {error.message}")

    _safe_log(f"[Sidecar] Final recognition chain: {build_final_recognition_chain(stt_service_type)}")
    _safe_log(f"[Sidecar] Streaming provider: {stt_service_type}")
    _safe_log(f"[Sidecar] Starting HTTP server on port {args.port}")
    _safe_log(f"[Sidecar] WebSocket available at ws://127.0.0.1:{args.port}/ws")
    app.run(host="127.0.0.1", port=args.port, debug=False, threaded=True)


if __name__ == "__main__":
    main()
