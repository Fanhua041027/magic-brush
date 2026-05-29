"""Knowledge Base search module — ported from Interview Assistant."""

import math
import os
import re
from collections import Counter


class KnowledgeBase:
    """TF-IDF based local knowledge base search for Markdown documents."""

    def __init__(self, kb_path=""):
        self.kb_path = kb_path
        self.chunks = []
        self.idf = {}
        self.ready = False
        if kb_path and os.path.isdir(kb_path):
            self.load(kb_path)

    def load(self, kb_path: str) -> dict:
        """Load knowledge base from a directory of .md files."""
        self.kb_path = kb_path
        self.chunks = []
        self.idf = {}
        all_texts = []
        for fname in sorted(os.listdir(kb_path)):
            if not fname.endswith('.md'):
                continue
            fpath = os.path.join(kb_path, fname)
            try:
                with open(fpath, 'r', encoding='utf-8') as f:
                    content = f.read()
            except Exception:
                continue
            sections = self._parse_sections(fname.replace('.md', ''), content)
            self.chunks.extend(sections)
            all_texts.extend(s['content'] for s in sections)
        self._compute_idf(all_texts)
        self.ready = True
        result = {"file_count": len(all_texts), "section_count": len(self.chunks)}
        return result

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
        text = re.sub(r'([一-鿿])', r' \1 ', text)
        return re.findall(r'[一-鿿\w]+', text.lower())

    def _compute_idf(self, texts):
        n = len(texts)
        df = Counter()
        for t in texts:
            for w in set(self._tokenize(t)):
                df[w] += 1
        self.idf = {w: math.log((n + 1) / (c + 1)) + 1 for w, c in df.items()}

    def search(self, query, top_k=5):
        if not self.ready or not self.chunks:
            return []
        qtokens = self._tokenize(query)
        scored = []
        for ch in self.chunks:
            ctokens = self._tokenize(ch['content'])
            ccnt = Counter(ctokens)
            score = 0.0
            for qt in qtokens:
                if qt in self.idf:
                    tf = ccnt.get(qt, 0) / max(len(ctokens), 1)
                    score += tf * self.idf[qt]
            if any(qt in self._tokenize(ch['header']) for qt in qtokens):
                score *= 2
            if score > 0:
                scored.append((score, ch))
        scored.sort(key=lambda x: x[0], reverse=True)
        return [
            dict(source=s['source'], header=s['header'], content=s['content'][:1200], score=round(sc, 4))
            for sc, s in scored[:top_k]
        ]

    def format_context(self, results, max_chars=2500):
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
