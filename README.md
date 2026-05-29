# Magic Brush

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev)
[![Vue 3](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vue.js)](https://vuejs.org)
[![Wails v2](https://img.shields.io/badge/Wails-v2-E30613)](https://wails.io)
[![Python](https://img.shields.io/badge/Python-3.11+-3776AB?logo=python)](https://python.org)

**Magic Brush** 是一个融合了截图解题 + 语音转写 + 知识库搜索的 AI 桌面助手，基于 Wails (Go+Vue) 构建。

---

## 功能

- **截图解题** — 截图即问，AI 实时分析解答
- **语音转写 (STT)** — 按住录音，语音输入问题（基于 faster-whisper）
- **知识库搜索** — 本地知识库 TF-IDF 检索，辅助回答
- **隐身防御** — Ghost Mode 防截屏、隐藏任务栏
- **多轮对话** — 支持上下文延续
- **全局快捷键** — 截图、发送、切换窗口等
- **多模型支持** — OpenAI 兼容接口，支持 DeepSeek/千问/OpenRouter 等

## 快速开始

### 前置条件

- Go 1.24+
- Node.js 20+
- Python 3.11+（用于语音转写 sidecar）
- NVIDIA GPU（可选，推荐用于 faster-whisper 加速）

### 运行

```bash
git clone https://github.com/Fanhua041027/magic-brush.git
cd magic-brush

# 安装 Go 前端依赖
cd frontend && npm install && cd ..

# 安装 Python sidecar 依赖
cd sidecar && pip install -r requirements.txt && cd ..

# 开发模式启动
wails dev -skipbindings
```

## 快捷键

| 动作 | Windows | macOS |
|------|---------|-------|
| 截图 | F8 | Cmd+1 |
| 发送解题 | Ctrl+J | Cmd+J |
| 显示/隐藏 | F9 | Cmd+2 |
| 语音输入 | 左Alt (按住说话) | - |
| AI 对话 | F7 | - |
| 鼠标穿透 | F10 | Cmd+3 |

## 功能特性

### 知识库集成
- 对话时自动搜索知识库
- 基于 TF-IDF 算法的相关性匹配
- 支持 Markdown 文件格式

### 简历集成
- 支持上传简历文件（PDF/Word）
- AI 对话时自动加载简历内容
- 根据简历信息回答相关问题
- 状态栏显示简历加载状态

### 截图追问
- 截图解题后自动弹出追问对话框
- 支持语音输入追问内容
- 保持上下文关联
- 支持多轮对话

## 架构

```
Wails Desktop App (Go + Vue 3)
  ├── Go Backend (app/ + pkg/)
  │   ├── LLM / Solver / Screen Capture
  │   ├── Shortcut Manager
  │   ├── Sidecar Manager (spawns Python)
  │   └── Ghost Mode / Window Control
  ├── Python Sidecar (sidecar/)
  │   ├── STT (faster-whisper audio transcription)
  │   └── KB Search (TF-IDF)
  └── Vue 3 Frontend
      ├── Solve Panel / History
      ├── STT Voice Input
      ├── KB Search Panel
      └── Settings
```

## 技术栈

- **桌面框架**: Wails v2
- **后端**: Go 1.24+
- **前端**: Vue 3 + Pinia + Vite
- **语音转写**: Python + faster-whisper
- **知识库**: TF-IDF 本地检索
- **LLM**: OpenAI 兼容 API

## License

MIT
