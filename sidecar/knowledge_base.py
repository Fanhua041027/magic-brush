"""Knowledge Base — TF-IDF 搜索，带缓存和反向索引"""

import math
import os
import re
import time
import threading
from collections import Counter, OrderedDict
from typing import Optional


# ── LRU 缓存 ──────────────────────────────────────────────

class LRUCache:
    """简单的 LRU 缓存"""

    def __init__(self, capacity: int = 128):
        self.capacity = capacity
        self._cache = OrderedDict()
        self._lock = threading.RLock()

    def get(self, key: str) -> Optional[list]:
        with self._lock:
            if key not in self._cache:
                return None
            self._cache.move_to_end(key)
            return self._cache[key]

    def put(self, key: str, value: list):
        with self._lock:
            self._cache[key] = value
            self._cache.move_to_end(key)
            if len(self._cache) > self.capacity:
                self._cache.popitem(last=False)

    def clear(self):
        with self._lock:
            self._cache.clear()

    @property
    def size(self) -> int:
        with self._lock:
            return len(self._cache)


# ── 知识库核心 ──────────────────────────────────────────────

class KnowledgeBase:
    """TF-IDF 本地知识库搜索，带反向索引和查询缓存"""

    MAX_FILES = 500
    MAX_FILE_BYTES = 2 * 1024 * 1024
    MAX_TOTAL_BYTES = 50 * 1024 * 1024

    def __init__(self, kb_path=""):
        self.kb_path = kb_path
        self.chunks = []
        self.idf = {}
        self._inverted_index: dict[str, list[tuple[int, float]]] = {}  # word -> [(chunk_idx, tf), ...]
        self._chunk_norms: list[float] = []  # 预计算的文档归一化因子
        self.ready = False
        self.file_count = 0
        self._query_cache = LRUCache(capacity=256)
        self._state_lock = threading.RLock()
        self._load_time = 0.0
        self._file_signature = None

        if kb_path and os.path.isdir(kb_path):
            self.load(kb_path)

    def load(self, kb_path: str) -> dict:
        """Build a replacement snapshot off-lock, then publish it atomically."""
        candidate = KnowledgeBase()
        result = candidate._load_locked(kb_path)
        with self._state_lock:
            self.kb_path = candidate.kb_path
            self.chunks = candidate.chunks
            self.idf = candidate.idf
            self._inverted_index = candidate._inverted_index
            self._chunk_norms = candidate._chunk_norms
            self.ready = candidate.ready
            self.file_count = candidate.file_count
            self._query_cache = candidate._query_cache
            self._load_time = candidate._load_time
            self._file_signature = candidate._file_signature
        return result

    def _load_locked(self, kb_path: str) -> dict:
        """加载知识库，构建反向索引"""
        start = time.time()
        self.kb_path = kb_path
        self.chunks = []
        self.idf = {}
        self._inverted_index = {}
        self._chunk_norms = []
        self._query_cache.clear()

        if not os.path.isdir(kb_path):
            self.ready = False
            self._file_signature = None
            return {"file_count": 0, "section_count": 0}

        md_files = sorted(f for f in os.listdir(kb_path) if f.endswith('.md'))
        if len(md_files) > self.MAX_FILES:
            raise ValueError(f"知识库文件数量超过限制 ({self.MAX_FILES})")
        self.file_count = len(md_files)

        all_texts = []
        total_bytes = 0
        for fname in md_files:
            fpath = os.path.join(kb_path, fname)
            try:
                size = os.path.getsize(fpath)
                if size > self.MAX_FILE_BYTES:
                    continue
                total_bytes += size
                if total_bytes > self.MAX_TOTAL_BYTES:
                    raise ValueError("知识库总大小超过限制")
                with open(fpath, 'r', encoding='utf-8', errors='ignore') as f:
                    content = f.read()
            except OSError:
                continue
            sections = self._parse_sections(fname.replace('.md', ''), content)
            self.chunks.extend(sections)
            for s in sections:
                all_texts.append(s['content'])

        if not all_texts:
            self.ready = True
            self._file_signature = self._directory_signature(kb_path)
            return {"file_count": self.file_count, "section_count": 0}

        self._compute_idf(all_texts)
        self._build_inverted_index(all_texts)
        self._file_signature = self._directory_signature(kb_path)
        self.ready = True
        self._load_time = time.time() - start

        print(f"[KB] 已加载 {self.file_count} 个文件, {len(self.chunks)} 个章节, "
              f"词典 {len(self.idf)} 词, 耗时 {self._load_time:.2f}s", flush=True)

        return {"file_count": self.file_count, "section_count": len(self.chunks)}

    def load_if_needed(self, kb_path: str) -> bool:
        """Reload when the path or Markdown file fingerprint changes."""
        signature = self._directory_signature(kb_path)
        with self._state_lock:
            current = (
                self.ready
                and self.kb_path == kb_path
                and self._file_signature == signature
            )
        if current:
            return True
        self.load(kb_path)
        with self._state_lock:
            return self.ready

    @staticmethod
    def _directory_signature(kb_path: str):
        """Return a cheap fingerprint for Markdown files in the directory."""
        if not os.path.isdir(kb_path):
            return None
        signature = []
        for fname in sorted(os.listdir(kb_path)):
            if not fname.endswith('.md'):
                continue
            fpath = os.path.join(kb_path, fname)
            try:
                stat = os.stat(fpath)
                signature.append((fname, stat.st_mtime_ns, stat.st_size))
            except OSError:
                signature.append((fname, None, None))
        return tuple(signature)

    @staticmethod
    def _parse_sections(source, content):
        sections = []
        lines = content.split('\n')
        cur_header = "概述"
        cur_body = []
        for line in lines:
            if line.startswith('## ') or line.startswith('### '):
                if cur_body:
                    text = '\n'.join(cur_body).strip()
                    if text:
                        sections.append(dict(source=source, header=cur_header, content=text))
                cur_header = line.lstrip('#').strip()
                cur_body = []
            else:
                cur_body.append(line)
        if cur_body:
            text = '\n'.join(cur_body).strip()
            if text:
                sections.append(dict(source=source, header=cur_header, content=text))
        return sections

    @staticmethod
    def _tokenize(text):
        """分词：中文字符逐字 + 英文单词"""
        text = re.sub(r'([一-鿿])', r' \1 ', text)
        return re.findall(r'[一-鿿\w]+', text.lower())

    def _compute_idf(self, texts):
        """计算 IDF"""
        n = len(texts)
        df = Counter()
        for t in texts:
            for w in set(self._tokenize(t)):
                df[w] += 1
        self.idf = {w: math.log((n + 1) / (c + 1)) + 1 for w, c in df.items()}

    def _build_inverted_index(self, texts: list[str]):
        """构建反向索引：word -> [(chunk_idx, tf_normalized), ...]"""
        self._inverted_index = {}
        self._chunk_norms = [0.0] * len(texts)

        for idx, text in enumerate(texts):
            tokens = self._tokenize(text)
            if not tokens:
                continue
            token_count = len(tokens)
            counter = Counter(tokens)

            # 预计算文档归一化因子
            norm = 0.0
            for word, count in counter.items():
                if word in self.idf:
                    tf = count / token_count
                    idf = self.idf[word]
                    weight = tf * idf
                    norm += weight * weight
                    if word not in self._inverted_index:
                        self._inverted_index[word] = []
                    self._inverted_index[word].append((idx, tf))

            self._chunk_norms[idx] = math.sqrt(norm) if norm > 0 else 1.0

    def search(self, query: str, top_k: int = 5, min_score: float = 0.01) -> list[dict]:
        with self._state_lock:
            ready = self.ready
            chunks = self.chunks
            idf = self.idf
            inverted_index = self._inverted_index
            chunk_norms = self._chunk_norms
            query_cache = self._query_cache
        return self._search_snapshot(
            query,
            top_k,
            min_score,
            ready,
            chunks,
            idf,
            inverted_index,
            chunk_norms,
            query_cache,
        )

    def _search_snapshot(self, query: str, top_k: int, min_score: float,
                         ready: bool, chunks: list, idf: dict,
                         inverted_index: dict, chunk_norms: list,
                         query_cache: LRUCache) -> list[dict]:
        """搜索知识库 — 使用反向索引加速

        Args:
            query: 搜索查询
            top_k: 返回结果数
            min_score: 最低分数阈值

        Returns:
            匹配结果列表
        """
        if not ready or not chunks:
            return []

        # 检查缓存
        cache_key = (query, top_k, min_score)
        cached = query_cache.get(cache_key)
        if cached is not None:
            return cached

        qtokens = self._tokenize(query)
        if not qtokens:
            return []

        # 使用反向索引进行快速评分
        chunk_scores: dict[int, float] = {}
        query_norm = 0.0

        for qt in set(qtokens):
            if qt in idf:
                q_tf = qtokens.count(qt) / len(qtokens)
                term_idf = idf[qt]
                q_weight = q_tf * term_idf
                query_norm += q_weight * q_weight

                if qt in inverted_index:
                    for chunk_idx, doc_tf in inverted_index[qt]:
                        doc_weight = doc_tf * term_idf
                        chunk_scores[chunk_idx] = chunk_scores.get(chunk_idx, 0.0) + q_weight * doc_weight

        query_norm = math.sqrt(query_norm) if query_norm > 0 else 1.0

        # 归一化 + 标题加分
        query_tokens_set = set(qtokens)
        scored = []
        for idx, raw_score in chunk_scores.items():
            norm = chunk_norms[idx] if idx < len(chunk_norms) else 1.0
            score = raw_score / (query_norm * norm) if norm > 0 else 0.0

            # 标题匹配加分
            header_tokens = set(self._tokenize(chunks[idx]['header']))
            if query_tokens_set & header_tokens:
                score *= 1.5

            if score > min_score:
                scored.append((score, idx))

        scored.sort(key=lambda x: x[0], reverse=True)

        results = [
            dict(
                source=chunks[idx]['source'],
                header=chunks[idx]['header'],
                content=chunks[idx]['content'][:1200],
                score=round(sc, 4),
            )
            for sc, idx in scored[:top_k]
        ]

        # 缓存结果
        query_cache.put(cache_key, results)

        return results

    def format_context(self, results, max_chars=2500):
        """格式化搜索结果上下文"""
        parts = []
        total = 0
        for r in results:
            snippet = f"[{r['source']} - {r['header']}]\n{r['content']}"
            if total + len(snippet) > max_chars:
                remain = max_chars - total
                if remain > 200:
                    parts.append(snippet[:remain] + '…')
                break
            parts.append(snippet)
            total += len(snippet)
        return '\n\n'.join(parts)
