<div align="center">

# ✨ Magic Brush

**AI 智能面试助手 · 截图解题 · 语音转写 · 知识库搜索 · 流式 AI 对话**

<br/>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white&style=for-the-badge">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white&style=for-the-badge">
</picture>
<img src="https://img.shields.io/badge/Vue_3-4FC08D?logo=vue.js&logoColor=white&style=for-the-badge" alt="Vue 3"/>
<img src="https://img.shields.io/badge/Wails_v2-E30613?logo=wails&logoColor=white&style=for-the-badge" alt="Wails v2"/>
<img src="https://img.shields.io/badge/Python_3.11+-3776AB?logo=python&logoColor=white&style=for-the-badge" alt="Python"/>
<img src="https://img.shields.io/badge/DeepSeek-4A6CF7?logo=deepseek&logoColor=white&style=for-the-badge" alt="DeepSeek"/>
<img src="https://img.shields.io/badge/license-MIT-blue?style=for-the-badge" alt="License"/>

<br/>

[📖 简介](#-简介) · [🚀 功能特性](#-功能特性) · [⌨️ 快捷键](#%EF%B8%8F-快捷键) · [🎬 快速开始](#-快速开始) · [🏗️ 架构](#%EF%B8%8F-架构设计) · [🛠️ 技术栈](#%EF%B8%8F-技术栈)

</div>

---

## 📖 简介

**Magic Brush** 是一款专为 **技术面试** 打造的 AI 桌面助手。基于 Wails (Go + Vue 3) 构建，深度融合 **DeepSeek** 大语言模型，提供从面试准备到实战的全链路辅助体验。

> 🎯 **定位**：你的面试副驾驶 — 实时分析问题、生成结构化回答、支持多轮追问与截图识别

### 核心能力一览

| 场景 | 能力 | 快捷操作 |
|:-----|:-----|:---------|
| 💬 **面试问答** | AI 实时分析并生成结构化答案，流式输出逐字呈现 | `F7` 打开对话 |
| 📸 **屏幕解题** | 截图识别编程题/算法题，AI 自动分析并生成代码 | `F8` 截图解题 |
| 🎤 **语音输入** | 按住说话自动转文字，边说边识别 | `左 Alt` 按住 |
| 📚 **知识库** | 本地 Markdown 知识库检索，自动注入对话上下文 | 设置中导入 |
| 🔍 **追问截图** | 多轮对话中再次截图，结合上下文继续追问 | `F6` 追问截图 |
| 🪟 **独立窗口** | 面试面板可弹出为独立窗口，自由拖拽到桌面任意位置 | `F1` / `Ctrl+Alt+Z` |

---

## 🚀 功能特性

### 🤖 AI 辅助面试（三栏布局）

```
┌──────────────────────┬──────────────────────────┬──────────────────┐
│     对话转录         │      AI 智能体            │     输入与控制    │
│                      │                          │                  │
│  面试官 ──────────   │  你的问题 ───────────    │  💡 解题思路     │
│  请问 RAG 是什么？   │  请解释 RAG 的原理       │  💻 代码优化     │
│                      │                          │  ⭐ STAR 回答    │
│  我 ────────────     │  AI 回答 ────────────    │                  │
│  RAG 是检索增强...   │  分析 → 检索 → 生成      │  [输入框...]  📷 │
│                      │  结构化回答...            │                  │
│                      │                          │  🎤 录音中  发送  │
├──────────────────────┴──────────────────────────┴──────────────────┤
│  ⏱ 计时器 · 简历已加载 · 背景透明度 · 文字透明度 · 历史对话      │
└────────────────────────────────────────────────────────────────────┘
```

- **三栏分屏**：对话转录、AI 智能体、输入与控制，信息一目了然
- **流式输出**：AI 回答逐字显示，实时可见，可随时停止
- **历史对话**：自动保存所有对话，支持加载和删除
- **透明度调节**：背景透明度和文字透明度独立调节（0~100%）
- **自由拖动**：长按顶栏可任意拖动位置，窗口大小可调
- **独立窗口**：一键弹出为独立桌面窗口，不受主窗口限制

### 📸 截图解题
| 功能 | 说明 |
|:-----|:------|
| 🖼️ 一键截图 | 按 `F8` 截图，AI 实时分析解答 |
| 🔍 视觉识别 | 支持代码题、算法题、图文题识别 |
| 💬 追问对话 | 解题后自动弹出追问框，支持多轮对话 |
| 📷 追问截图 | 追问中按 `F6` 截图继续提问 |
| ⏹️ 停止思考 | 随时点击"停止"按钮终止 AI 生成 |

### 🎙️ 语音转写 (STT)
| 功能 | 说明 |
|:-----|:------|
| 🎤 按住说话 | `左 Alt` 按住录音，松开自动识别 |
| 🔄 流式转写 | 边说边识别，实时显示识别结果 |
| 🌐 多服务支持 | 千问云端 / 千问本地 / 本地 Whisper 三种模式 |
| 🎛️ 音频输入 | 支持麦克风和立体声混音切换 |
| ⚡ CUDA 加速 | 本地 Whisper 支持 GPU 加速推理 |

### 📚 知识库搜索
| 功能 | 说明 |
|:-----|:------|
| 🔎 TF-IDF 检索 | 本地 Markdown 知识库相关性匹配 |
| 📁 Markdown 格式 | 支持目录导入，自动解析章节结构 |
| 🤖 自动注入 | 对话时自动检索并注入上下文 |

### 🛡️ 隐身防御 (Ghost Mode)
| 功能 | 说明 |
|:-----|:------|
| 🚫 防截屏 | `SetWindowDisplayAffinity` 防止被系统截屏捕获 |
| 👻 隐藏任务栏 | `WS_EX_TOOLWINDOW` 隐藏任务栏图标 |
| 🖥️ 无边框 | 完全无边框窗口，沉浸式体验 |
| 🔒 隐私保护 | 悬浮窗默认不进录屏，保护面试隐私 |

### 💡 其他亮点
- **📋 简历集成** — 上传简历后 AI 自动参考你的背景回答问题
- **🔑 多模型支持** — OpenAI 兼容接口，支持 DeepSeek / 千问 / OpenRouter 等
- **🎨 深色主题** — 护眼深色模式，支持主题切换
- **💾 本地存储** — 配置和对话历史全部本地保存

---

## ⌨️ 快捷键

<div align="center">

| 动作 | Windows | macOS | 说明 |
|:-----|:--------|:------|:-----|
| 🪟 **弹出独立面试窗** | `F1` / `Ctrl + Alt + Z` | — | 最大化窗口+打开面试面板 |
| 💬 **AI 辅助面试** | `F7` | — | 打开三栏面试面板 |
| 📸 **截图解题** | `F8` | `⌘ + 1` | 截图并发送给 AI |
| 📤 **发送解题** | `Ctrl + J` | `⌘ + J` | 发送当前截图解题 |
| 👁️ **显示/隐藏** | `F9` | `⌘ + 2` | 切换窗口可见性 |
| 🎤 **语音输入** | `左 Alt` (按住) | — | 按住说话，松开识别 |
| 🖱️ **鼠标穿透** | `F10` | `⌘ + 3` | 切换鼠标穿透模式 |
| ❌ **删除截图** | `Ctrl + D` | `⌘ + D` | 删除最后一张截图 |

</div>

---

## 🎬 快速开始

### 前置条件

| 依赖 | 版本 | 用途 |
|:-----|:-----|:------|
| **Go** | ≥ 1.24 | Wails 后端编译 |
| **Node.js** | ≥ 20 | Vue 3 前端构建 |
| **Python** | ≥ 3.11 | 语音转写 sidecar |
| **NVIDIA GPU** | 可选 | faster-whisper 加速 |

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
# 构建生产版本（Windows）
wails build -skipbindings

# 构建产物位于 build/bin/ShenbiMaliang.exe
```

### ⚠️ 首次使用

1. 启动后在**设置**中配置 **API Key**（DeepSeek / OpenAI 兼容接口）
2. 可选：上传简历 → AI 在回答时会参考你的个人背景
3. 可选：导入知识库（Markdown 格式目录） → 自动注入面试相关知识

---

## 🏗️ 架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                     Magic Brush Desktop App                      │
│                       Wails (Go + Vue 3)                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────────────────────┐    ┌─────────────────────────────┐  │
│  │      Go Backend         │    │      Vue 3 Frontend         │  │
│  │      (app/ + pkg/)      │    │                             │  │
│  │                         │    │  ┌─────────┐ ┌───────────┐  │  │
│  │  ┌───────────────────┐  │    │  │ Solve   │ │ Chat      │  │  │
│  │  │ LLM Provider      │  │    │  │ Panel   │ │ Dialog    │  │  │
│  │  │ (OpenAI API)      │  │    │  └─────────┘ └───────────┘  │  │
│  │  └───────────────────┘  │    │                             │  │
│  │  ┌───────────────────┐  │    │  ┌─────────┐ ┌───────────┐  │  │
│  │  │ Screen Capture    │  │    │  │ STT     │ │ Tutorial  │  │  │
│  │  │ + Ghost Mode      │  │    │  │ Voice   │ │ Wizard    │  │  │
│  │  └───────────────────┘  │    │  └─────────┘ └───────────┘  │  │
│  │  ┌───────────────────┐  │    │                             │  │
│  │  │ Shortcut Manager  │  │    │  ┌─────────────────────┐    │  │
│  │  │ (Global Hotkeys)  │  │    │  │ Settings Modal      │    │  │
│  │  └───────────────────┘  │    │  └─────────────────────┘    │  │
│  │  ┌───────────────────┐  │    │                             │  │
│  │  │ Config Manager    │  │    │  ┌─────────────────────┐    │  │
│  │  │ (AES Encrypted)   │  │    │  │ Standalone Panel   │    │  │
│  │  └───────────────────┘  │    │  └─────────────────────┘    │  │
│  └────────────────────────┘    └─────────────────────────────┘  │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    Python Sidecar                          │  │
│  │  ┌───────────────┐    ┌────────────────┐    ┌──────────┐  │  │
│  │  │ STT Service   │    │ KB Search     │    │ WebSocket│  │  │
│  │  │ (Whisper/     │    │ (TF-IDF)      │    │ Stream   │  │  │
│  │  │  Qwen ASR)    │    │               │    │          │  │  │
│  │  └───────────────┘    └────────────────┘    └──────────┘  │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🛠️ 技术栈

<div align="center">

| 层级 | 技术 | 版本 | 说明 |
|:-----|:-----|:-----|:------|
| 🖥️ **桌面框架** | [Wails](https://wails.io) | v2 | Go + Vue 3 跨平台桌面应用 |
| ⚙️ **后端** | [Go](https://go.dev) | 1.24+ | 高性能并发，系统级 API 调用 |
| 🎨 **前端** | [Vue 3](https://vuejs.org) + [Pinia](https://pinia.vuejs.org) | 3.x | 响应式 UI + 状态管理 |
| 🧠 **LLM** | [DeepSeek](https://deepseek.com) / OpenAI API | — | 智能对话与解题 |
| 🎤 **语音** | [faster-whisper](https://github.com/SYSTRAN/faster-whisper) | — | 离线语音识别 |
| 📚 **知识库** | TF-IDF | — | 本地 Markdown 检索 |
| 🔌 **Sidecar** | [Flask](https://flask.palletsprojects.com) + Python | 3.11+ | STT & KB HTTP 服务 |
| 🔐 **安全** | AES-256-GCM | — | 配置加密存储 |

</div>

---

## 📁 项目结构

```
magic-brush/
├── main.go                  # 应用入口（支持 --standalone-interview 参数）
├── wails.json               # Wails 配置文件
│
├── app/                     # Go 后端应用逻辑
│   ├── app.go              # 应用主入口 + Startup/Shutdown
│   ├── chat.go             # AI 对话（流式 + 非流式）
│   ├── ai.go               # STT 绑定 + 截图解题
│   ├── solve.go            # 解题引擎逻辑
│   ├── screen.go           # 屏幕截图绑定
│   ├── shortcut.go         # 快捷键处理
│   ├── inject.go           # 上下文注入
│   └── window.go           # 窗口操作
│
├── pkg/                    # Go 核心包
│   ├── llm/               # LLM Provider 抽象层
│   ├── config/            # 配置管理（AES 加密）
│   ├── shortcut/          # 全局快捷键系统
│   ├── screen/            # 截图服务
│   ├── solution/          # 解题引擎
│   ├── sidecar/           # Python Sidecar 进程管理
│   ├── platform/          # Windows/macOS 平台 API
│   └── tools/             # 工具函数
│
├── frontend/              # Vue 3 前端
│   ├── src/
│   │   ├── components/    # Vue 组件（20+ 个组件）
│   │   │   ├── ChatDialog.vue           # AI 辅助面试三栏面板
│   │   │   ├── StandaloneInterviewPanel.vue  # 独立面试窗口
│   │   │   ├── TutorialWizard.vue       # 入門教程向导
│   │   │   └── ...                      # 其他组件
│   │   ├── stores/        # Pinia 状态管理
│   │   │   ├── chat.js    # 聊天状态（含多会话管理）
│   │   │   ├── settings.js # 设置状态
│   │   │   └── ...
│   │   ├── services/      # API 服务层
│   │   └── utils/         # 工具函数（Markdown 渲染等）
│   └── wailsjs/           # Wails 自动生成的 JS 绑定
│
├── sidecar/               # Python Sidecar
│   ├── main.py            # Flask HTTP 服务 + WebSocket
│   ├── transcribe.py      # Whisper 语音转写
│   ├── audio.py           # 音频录制
│   ├── knowledge_base.py  # TF-IDF 知识库
│   ├── qwen_stt.py        # 千问云端 STT
│   └── qwen_asr_local.py  # 千问本地 ASR
│
└── build/                 # 构建输出
    └── bin/
        └── ShenbiMaliang.exe  # 可执行文件
```

---

## 🤝 贡献

欢迎任何形式的贡献！无论是新功能、Bug 修复还是文档改进。

1. **Fork** 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'feat: Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 **Pull Request**

### 开发指引

```bash
# 前端热重载开发
cd frontend && npm run dev

# Go 后端测试
go test ./...

# 构建
wails build -skipbindings
```

---

## 📄 许可证

本项目基于 **MIT 许可证** 开源 — 详见 [LICENSE](LICENSE) 文件。

---

<div align="center">

### ⭐ 如果觉得有用，请点个 Star 支持！⭐

<br/>

[![Star History Chart](https://api.star-history.com/svg?repos=Fanhua041027/magic-brush&type=Date&theme=dark)](https://star-history.com/#Fanhua041027/magic-brush&Date)

<br/>

**Made with ❤️ for interview warriors**

</div>
