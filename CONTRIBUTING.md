# 贡献指南

感谢你对 Magic Brush 项目的关注！我们欢迎任何形式的贡献。

## 如何贡献

### 报告问题

如果你发现了 bug 或有功能建议，请：

1. 点击 [Issues](https://github.com/Fanhua041027/magic-brush/issues) 页面
2. 点击 "New Issue" 按钮
3. 选择合适的模板（Bug 报告或功能请求）
4. 填写详细信息

### 提交代码

1. **Fork 本仓库**
   ```bash
   # 点击页面右上角的 Fork 按钮
   ```

2. **克隆你的 Fork**
   ```bash
   git clone https://github.com/你的用户名/magic-brush.git
   cd magic-brush
   ```

3. **创建特性分支**
   ```bash
   git checkout -b feature/你的特性名称
   ```

4. **进行修改并提交**
   ```bash
   git add .
   git commit -m "feat: 添加你的特性描述"
   ```

5. **推送到你的 Fork**
   ```bash
   git push origin feature/你的特性名称
   ```

6. **创建 Pull Request**
   - 点击 "New Pull Request" 按钮
   - 填写详细的描述
   - 等待审核

### 提交规范

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

- `feat:` 新功能
- `fix:` 修复 bug
- `docs:` 文档更新
- `style:` 代码格式（不影响功能）
- `refactor:` 重构
- `perf:` 性能优化
- `test:` 测试相关
- `chore:` 构建/工具相关

示例：
```
feat: 添加语音识别功能
fix: 修复截图时的内存泄漏
docs: 更新 README 文档
```

## 开发环境

### 前置条件

- Go 1.24+
- Node.js 20+
- Python 3.11+
- NVIDIA GPU（可选，用于 Whisper 加速）

### 设置开发环境

```bash
# 1. 克隆仓库
git clone https://github.com/Fanhua041027/magic-brush.git
cd magic-brush

# 2. 安装 Go 依赖
go mod tidy

# 3. 安装前端依赖
cd frontend && npm install && cd ..

# 4. 安装 Python 依赖
cd sidecar && pip install -r requirements.txt && cd ..

# 5. 启动开发服务器
wails dev
```

### 项目结构

```
magic-brush/
├── app/                    # Go 后端
├── pkg/                    # Go 包
├── frontend/              # Vue 3 前端
├── sidecar/               # Python Sidecar
└── build/                 # 构建产物
```

## 代码规范

### Go 代码

- 使用 `gofmt` 格式化代码
- 遵循 [Effective Go](https://go.dev/doc/effective_go) 指南
- 添加必要的注释

### Vue 代码

- 使用 ESLint 和 Prettier
- 遵循 [Vue 风格指南](https://vuejs.org/style-guide/)
- 使用 Composition API

### Python 代码

- 使用 Black 格式化
- 遵循 PEP 8 规范
- 添加类型注解

## 行为准则

请遵守我们的 [行为准则](CODE_OF_CONDUCT.md)。

## 联系方式

- Issues: [GitHub Issues](https://github.com/Fanhua041027/magic-brush/issues)
- Discussions: [GitHub Discussions](https://github.com/Fanhua041027/magic-brush/discussions)

## 许可证

本项目基于 CC BY-NC 4.0 许可证 - 详见 [LICENSE](LICENSE) 文件

---

感谢你的贡献！🎉
