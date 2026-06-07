"""Helpers for STT provider normalization and fallback ordering."""

from __future__ import annotations


_ALIASES = {
    "": "qwen_cloud",
    "qwen": "qwen_cloud",
}


def normalize_stt_service(service: str | None) -> str:
    normalized = (service or "").strip().lower()
    return _ALIASES.get(normalized, normalized)


def build_final_recognition_chain(selected_service: str | None) -> list[str]:
    service = normalize_stt_service(selected_service)
    chain = ["qwen3_asr_flash", service]

    if service == "qwen3_asr_flash":
        chain.extend(["qwen_cloud", "local_whisper", "qwen_local"])
    elif service == "qwen_local":
        chain.extend(["qwen_cloud", "local_whisper"])
    elif service == "local_whisper":
        chain.extend(["qwen_cloud", "qwen_local"])
    else:
        chain.extend(["local_whisper", "qwen_local"])

    deduped: list[str] = []
    for item in chain:
        if item and item not in deduped:
            deduped.append(item)
    return deduped
