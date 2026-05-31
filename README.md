<div align="center">

# ✨ Magic Brush

**AI 智能桌面助手 — 截图解题 · 语音转写 · 知识库搜索**

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Vue 3](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vue.js&logoColor=white)](https://vuejs.org)
[![Wails v2](https://img.shields.io/badge/Wails-v2-E30613?logo=wails&logoColor=white)](https://wails.io)
[![Python](https://img.shields.io/badge/Python-3.11+-3776AB?logo=python&logoColor=white)](https://python.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[![GitHub Stars](https://img.shields.io/github/stars/Fanhua041027/magic-brush?style=social)](https://github.com/Fanhua041027/magic-brush/stargazers)
[![GitHub Forks](https://img.shields.io/github/forks/Fanhua041027/magic-brush?style=social)](https://github.com/Fanhua041027/magic-brush/network/members)

<br/>

[功能特性](#-功能特性) · [快速开始](#-快速开始) · [快捷键](#-快捷键) · [架构设计](#-架构设计) · [技术栈](#-技术栈)

</div>

---

## 📖 简介

Magic Brush 是一款融合了 **截图解题**、**语音转写**、**知识库搜索** 的 AI 桌面助手。基于 Wails (Go + Vue 3) 构建，支持全局快捷键、隐身防御、多轮对话等功能，是面试辅助、学习解题的利器。

<div align="center">
  <img src="https://img.shields.io/badge/平台-Windows%20%7C%20macOS-lightgrey?style=for-the-badge" alt="Platform">
</div>

---

## 🚀 功能特性

### 📸 截图解题
| 功能 | 说明 |
|:-----|:-----|
| 🖼️ 一键截图 | 按 F8 截图，AI 实时分析解答 |
| 🔍 图片识别 | 支持视觉模型，自动识别截图内容 |
| 💬 追问对话 | 解题后自动弹出追问框，支持多轮对话 |
| 📷 追问截图 | 追问中按 F6/F8 截图，继续提问 |
| ⏹️ 停止思考 | 模型思考时可点击"停止"按钮终止 |

### 🎙️ 语音转写 (STT)
| 功能 | 说明 |
|:-----|:-----|
| 🎤 按住说话 | 左 Alt 键按住录音，松开自动识别 |
| 🔄 流式转写 | 边说边识别，实时显示 |
| 🌐 多服务支持 | 本地 Whisper / 千问云端 / 千问本地 |
| 🎛️ 音频输入 | 支持麦克风和立体声混音切换 |
| ⚡ CUDA 加速 | 本地 Whisper 支持 GPU 加速 |

### 📚 知识库搜索
| 功能 | 说明 |
|:-----|:-----|
| 🔎 TF-IDF 检索 | 本地知识库相关性匹配 |
| 📁 Markdown 格式 | 支持 Markdown 文件导入 |
| 🤖 自动注入 | 对话时自动搜索并注入上下文 |

### 🛡️ 隐身防御 (Ghost Mode)
| 功能 | 说明 |
|:-----|:-----|
| 🚫 防截屏 | SetWindowDisplayAffinity 防止被捕获 |
| 👻 隐藏任务栏 | WS_EX_TOOLWINDOW 隐藏任务栏图标 |
| 🖥️ 无边框 | FramelessWindowHint 无边框窗口 |

### 💡 其他功能
- **全局快捷键** — 截图、发送、切换窗口等
- **多模型支持** — OpenAI 兼容接口，支持 DeepSeek / 千问 / OpenRouter 等
- **简历集成** — AI 对话时自动加载简历内容
- **对话历史** — 本地存储，支持导出/清除
- **WebSocket** — 实时通信，自动重连

---

## ⌨️ 快捷键

<div align="center">

| 动作 | Windows | macOS |
|:-----|:--------|:------|
| 📸 截图 | `F8` | `⌘ + 1` |
| 📤 发送解题 | `Ctrl + J` | `⌘ + J` |
| 👁️ 显示/隐藏 | `F9` | `⌘ + 2` |
| 🎤 语音输入 | `左 Alt` (按住) | - |
| 💬 AI 对话 | `F7` | - |
| 🖱️ 鼠标穿透 | `F10` | `⌘ + 3` |
| ❌ 删除截图 | `Ctrl + D` | `⌘ + D` |

</div>

---

## 🏃 快速开始

### 前置条件

- **Go** 1.24+
- **Node.js** 20+
- **Python** 3.11+（用于语音转写 sidecar）
- **NVIDIA GPU**（可选，推荐用于 faster-whisper 加速）

### 安装运行

```bash
# 1. 克隆仓库
git clone https://github.com/Fanhua041027/magic-brush.git
cd magic-brush

# 2. 安装前端依赖
cd frontend && npm install && cd ..

# 3. 安装 Python sidecar 依赖
cd sidecar && pip install -r requirements.txt && cd ..

# 4. 开发模式启动
wails dev -skipbindings
```

### 构建打包

```bash
# 构建生产版本
wails build

# 构建产物位于 build/bin/ 目录
```

---

## 🏗️ 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                    Wails Desktop App                        │
│                     (Go + Vue 3)                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────┐    ┌─────────────────────────────┐ │
│  │    Go Backend        │    │    Vue 3 Frontend           │ │
│  │    (app/ + pkg/)     │    │                             │ │
│  │                      │    │  ┌─────────┐ ┌───────────┐  │ │
│  │  ┌───────────────┐   │    │  │ Solve   │ │ History   │  │ │
│  │  │ LLM Provider  │   │    │  │ Panel   │ │ Panel     │  │ │
│  │  │ (OpenAI API)  │   │    │  └─────────┘ └───────────┘  │ │
│  │  └───────────────┘   │    │                             │ │
│  │  ┌───────────────┐   │    │  ┌─────────┐ ┌───────────┐  │ │
│  │  │ Screen Capture│   │    │  │ STT     │ │ KB Search │  │ │
│  │  │ + Ghost Mode  │   │    │  │ Voice   │ │ Panel     │  │ │
│  │  └───────────────┘   │    │  └─────────┘ └───────────┘  │ │
│  │  ┌───────────────┐   │    │                             │ │
│  │  │ Shortcut Mgr  │   │    │  ┌─────────────────────┐    │ │
│  │  │ (Global Keys) │   │    │  │ Settings Modal      │    │ │
│  │  └───────────────┘   │    │  └─────────────────────┘    │ │
│  └─────────────────────┘    └─────────────────────────────┘ │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────┐    │
│  │              Python Sidecar (sidecar/)               │    │
│  │  ┌───────────────┐    ┌───────────────────────────┐  │    │
│  │  │ STT Service   │    │ KB Search (TF-IDF)        │  │    │
│  │  │ (Whisper/     │    │                           │  │    │
│  │  │  Qwen ASR)    │    │                           │  │    │
│  │  └───────────────┘    └───────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

---

## 🛠️ 技术栈

<div align="center">

| 层级 | 技术 | 说明 |
|:-----|:-----|:-----|
| **桌面框架** | Wails v2 | Go + Vue 3 跨平台桌面应用 |
| **后端** | Go 1.24+ | 高性能并发处理 |
| **前端** | Vue 3 + Pinia + Vite | 响应式 UI，状态管理 |
| **语音转写** | Python + faster-whisper | 离线语音识别 |
| **知识库** | TF-IDF | 本地知识库检索 |
| **LLM** | OpenAI 兼容 API | 支持多种模型 |

</div>

---

## 📁 项目结构

```
magic-brush/
├── app/                    # Go 后端应用逻辑
│   ├── app.go             # 应用主入口
│   ├── chat.go            # AI 对话功能
│   ├── solve.go           # 截图解题逻辑
│   └── screen.go          # 屏幕截图
├── pkg/                    # Go 包
│   ├── llm/               # LLM Provider
│   ├── solution/          # 解题引擎
│   ├── screen/            # 截图服务
│   ├── shortcut/          # 快捷键管理
│   └── sidecar/           # Python Sidecar 管理
├── frontend/              # Vue 3 前端
│   ├── src/
│   │   ├── components/    # Vue 组件
│   │   ├── stores/        # Pinia 状态管理
│   │   └── services/      # 服务层
│   └── wailsjs/           # Wails JS 绑定
├── sidecar/               # Python Sidecar
│   ├── main.py            # HTTP 服务入口
│   ├── transcribe.py      # Whisper 语音转写
│   └── knowledge_base.py  # 知识库搜索
└── build/                 # 构建产物
```

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！详见 [贡献指南](CONTRIBUTING.md)。

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'feat: Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

---

## 📄 许可证

本项目基于 MIT 许可证开源 - 详见 [LICENSE](LICENSE) 文件

---

<div align="center">

**⭐ 如果觉得有用，请点个 Star 支持一下！⭐**

[![Star History Chart](https://api.star-history.com/svg?repos=Fanhua041027/magic-brush&type=Date)](https://star-history.com/#Fanhua041027/magic-brush&Date)

</div>
