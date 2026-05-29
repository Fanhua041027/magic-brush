"""AI-Assistant Sidecar: HTTP server providing STT + KB search services."""

import argparse
import json
import os
import sys
import threading

from flask import Flask, jsonify, request

from audio import AudioRecorder
from transcribe import Transcriber
from knowledge_base import KnowledgeBase

app = Flask(__name__)

# Global state
recorder: AudioRecorder | None = None
transcriber: Transcriber | None = None
kb: KnowledgeBase | None = None
recording_lock = threading.Lock()
is_recording = False


# ── Health ──────────────────────────────────────────────────────────────

@app.route("/api/health", methods=["GET"])
def health():
    return jsonify({"status": "ok", "stt_ready": transcriber is not None})


# ── STT (Speech-to-Text) ───────────────────────────────────────────────

@app.route("/api/stt/status", methods=["GET"])
def stt_status():
    return jsonify({"recording": is_recording})


@app.route("/api/stt/start", methods=["POST"])
def stt_start():
    global is_recording
    if recorder is None:
        return jsonify({"error": "Audio recorder not initialized"}), 500
    with recording_lock:
        if is_recording:
            return jsonify({"error": "Already recording"}), 409
        recorder.start_recording()
        is_recording = True
    return jsonify({"status": "recording"})


@app.route("/api/stt/stop", methods=["POST"])
def stt_stop():
    global is_recording
    if recorder is None or transcriber is None:
        return jsonify({"error": "STT not initialized"}), 500
    with recording_lock:
        if not is_recording:
            return jsonify({"error": "Not recording"}), 409
        audio = recorder.stop_recording()
        is_recording = False
    if audio.size == 0:
        return jsonify({"text": ""})
    try:
        text = transcriber.transcribe(audio)
        return jsonify({"text": text})
    except Exception as e:
        try:
            transcriber.switch_to_cpu()
            text = transcriber.transcribe(audio)
            return jsonify({"text": text})
        except Exception as e2:
            return jsonify({"error": f"Transcription failed: {e2}"}), 500


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

@app.route("/api/kb/search", methods=["POST"])
def kb_search():
    global kb
    if kb is None or not kb.ready:
        return jsonify({"results": [], "error": "KB not loaded"}), 400
    data = request.get_json(silent=True) or {}
    query = data.get("query", "")
    top_k = data.get("top_k", 5)
    if not query:
        return jsonify({"results": []})
    results = kb.search(query, top_k=top_k)
    return jsonify({"results": results})


@app.route("/api/kb/load", methods=["POST"])
def kb_load():
    global kb
    data = request.get_json(silent=True) or {}
    path = data.get("path", "")
    if not path or not os.path.isdir(path):
        return jsonify({"error": f"Invalid path: {path}"}), 400
    try:
        if kb is None:
            kb = KnowledgeBase()
        result = kb.load(path)
        return jsonify({"status": "ok", **result})
    except Exception as e:
        return jsonify({"error": str(e)}), 500


# ── Main ───────────────────────────────────────────────────────────────

def main():
    global recorder, transcriber

    parser = argparse.ArgumentParser(description="AI-Assistant Sidecar")
    parser.add_argument("--port", type=int, default=18765, help="HTTP port")
    parser.add_argument("--model", default="base", help="Whisper model size")
    parser.add_argument("--device", default="auto", help="Device: auto/cuda/cpu")
    args = parser.parse_args()

    print(f"[Sidecar] Initializing STT (model={args.model}, device={args.device})...")
    try:
        transcriber = Transcriber(model_name=args.model, device=args.device)
        print(f"[Sidecar] Transcriber ready (device={transcriber.device})")
    except Exception as e:
        print(f"[Sidecar] Transcriber init failed: {e}")

    try:
        recorder = AudioRecorder()
        print("[Sidecar] Audio recorder ready")
    except Exception as e:
        print(f"[Sidecar] Audio recorder init failed: {e}")

    print(f"[Sidecar] Starting HTTP server on port {args.port}")
    app.run(host="127.0.0.1", port=args.port, debug=False)


if __name__ == "__main__":
    main()
