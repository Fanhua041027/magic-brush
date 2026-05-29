/**
 * Markdown + LaTeX 渲染工具（带缓存 + 增量渲染优化）
 * 支持多种 LaTeX 分隔符格式 + 代码块语法高亮
 */
import { marked } from 'marked'
import katex from 'katex'
import hljs from 'highlight.js/lib/common'

// ==================== 缓存系统 ====================

const CACHE_SIZE = 100
const renderCache = new Map()
const latexCache = new Map()

function hashString(str) {
    let hash = 0
    for (let i = 0; i < str.length; i++) {
        hash = ((hash << 5) - hash + str.charCodeAt(i)) | 0
    }
    return hash.toString(36)
}

function getFromCache(key) {
    if (renderCache.has(key)) {
        const value = renderCache.get(key)
        renderCache.delete(key)
        renderCache.set(key, value)
        return value
    }
    return null
}

function setToCache(key, value) {
    if (renderCache.size >= CACHE_SIZE) {
        const firstKey = renderCache.keys().next().value
        renderCache.delete(firstKey)
    }
    renderCache.set(key, value)
}

// ==================== LaTeX 渲染 ====================

function renderLatex(latex, displayMode = false) {
    const cacheKey = `${displayMode ? 'D' : 'I'}:${latex}`

    if (latexCache.has(cacheKey)) {
        return latexCache.get(cacheKey)
    }

    let result
    try {
        result = katex.renderToString(latex, {
            displayMode,
            throwOnError: false,
            strict: false,
            trust: true,
            output: 'html'
        })
    } catch (e) {
        result = displayMode ? `$$${latex}$$` : `$${latex}$`
    }

    if (latexCache.size < 500) {
        latexCache.set(cacheKey, result)
    }

    return result
}

// ==================== 预处理/后处理 ====================

function preprocessLatex(text) {
    const placeholders = []
    let idx = 0
    //先把公式全部提取出来，然后换成占位符，然后给markdown渲染器渲染，最后替换公式占位符为真正的公式
    //主要是防止latex语法和markdown冲突导致渲染错误。
    // 1. 块级公式 $$...$$
    text = text.replace(/\$\$([\s\S]+?)\$\$/g, (_, latex) => {
        const ph = `%%LATEX_BLOCK_${idx}%%`
        placeholders.push({ ph, html: renderLatex(latex.trim(), true) })
        idx++
        return ph
    })

    // 2. 块级公式 \[...\]
    text = text.replace(/\\\[([\s\S]+?)\\\]/g, (_, latex) => {
        const ph = `%%LATEX_BLOCK_${idx}%%`
        placeholders.push({ ph, html: renderLatex(latex.trim(), true) })
        idx++
        return ph
    })

    // 3. 行内公式 [...] (AI 常用格式)
    text = text.replace(/\[\s*([^[\]]*(?:\\[a-zA-Z]+|[=+\-*/^_{}])[^[\]]*)\s*\]/g, (match, latex) => {
        if (/^\s*[^\\=+\-*/^_{}]+\s*$/.test(latex)) {
            return match
        }
        const ph = `%%LATEX_INLINE_${idx}%%`
        placeholders.push({ ph, html: renderLatex(latex.trim(), false) })
        idx++
        return ph
    })

    // 4. 行内公式 \(...\)
    text = text.replace(/\\\(([\s\S]+?)\\\)/g, (_, latex) => {
        const ph = `%%LATEX_INLINE_${idx}%%`
        placeholders.push({ ph, html: renderLatex(latex.trim(), false) })
        idx++
        return ph
    })

    // 5. 行内公式 $...$
    text = text.replace(/(?<!\$)\$(?!\$)([^$]+?)(?<!\$)\$(?!\$)/g, (_, latex) => {
        const ph = `%%LATEX_INLINE_${idx}%%`
        placeholders.push({ ph, html: renderLatex(latex.trim(), false) })
        idx++
        return ph
    })

    return { text, placeholders }
}

function postprocessLatex(html, placeholders) {
    for (const { ph, html: latexHtml } of placeholders) {
        html = html.replace(ph, latexHtml)
    }
    return html
}

// ==================== 代码围栏修复 ====================

/**
 * 确保 ``` 标记在独立行上，修复 AI 输出（尤其是音频转录）中代码围栏
 * 与正文粘连导致 marked 将其解析为行内代码的问题
 */
function normalizeCodeFences(text) {
    // Step 1: ``` 前面有非换行内容时，插入换行使其独立成行
    text = text.replace(/([^\n])(```)/g, '$1\n\n$2')
    // Step 2: 开围栏 ```lang 后面如果同行还有代码内容，把代码移到下一行
    // 例如 "```java class Solution {..." → "```java\nclass Solution {..."
    // 避免代码被 marked 当作 info string 导致代码块体为空
    text = text.replace(/^(```\w*)[ \t]+(\S.*)$/gm, '$1\n$2')
    // Step 3: 闭合围栏 ``` 后紧跟非换行内容时，插入换行
    // 排除 ```lang 形式的开围栏（后跟字母/数字/+/-）
    text = text.replace(/(```)(?![\w+-])[ \t]*([^\n])/g, '$1\n\n$2')
    return text
}

// ==================== 主渲染函数 ====================

/**
 * 渲染 Markdown + LaTeX（带缓存）
 */
export function renderMarkdownWithLatex(md) {
    if (!md) return ''

    const cacheKey = hashString(md)
    const cached = getFromCache(cacheKey)
    if (cached) return cached

    // 修复代码围栏格式（AI 音频转录等场景下 ``` 可能不在行首）
    md = normalizeCodeFences(md)

    const { text, placeholders } = preprocessLatex(md)
    let html = marked.parse(text)
    html = postprocessLatex(html, placeholders)

    setToCache(cacheKey, html)

    return html
}

// ==================== 增量渲染器（流式输出优化） ====================

/**
 * 创建增量渲染器实例
 * 流式输出时只渲染已完成的段落，避免重复渲染
 */
export function createIncrementalRenderer() {
    let lastFullText = ''
    let renderedHtml = ''
    let lastCompleteIndex = 0  // 上次完整段落结束位置

    return {
        /**
         * 增量渲染
         * @param {string} fullText 当前完整文本
         * @returns {string} 渲染后的 HTML
         */
        render(fullText) {
            if (!fullText) return ''

            // 如果内容没变或变短了（可能是新对话），重新渲染
            if (fullText.length <= lastFullText.length && fullText !== lastFullText) {
                this.reset()
            }

            // 内容没变，直接返回缓存
            if (fullText === lastFullText) {
                return renderedHtml
            }

            // 找到最后一个完整段落的位置（以双换行或单换行结尾）
            const completeIndex = findLastCompleteParagraph(fullText)

            if (completeIndex > lastCompleteIndex) {
                // 有新的完整段落，渲染新增部分
                const newComplete = fullText.substring(lastCompleteIndex, completeIndex)
                const newHtml = renderMarkdownWithLatex(newComplete)
                renderedHtml += newHtml
                lastCompleteIndex = completeIndex
            }

            // 处理尾部未完成的文本（简单转义显示）
            const tail = fullText.substring(lastCompleteIndex)
            let tailHtml = ''
            if (tail.trim()) {
                    // Render tail with light markdown (paragraphs + line breaks, no LaTeX)
                tailHtml = `<span class="streaming-tail">${renderPartialTail(tail)}</span>`
            }

            lastFullText = fullText

            return renderedHtml + tailHtml
        },

        /**
         * 完成渲染（流结束时调用）
         * 渲染剩余未完成的部分
         */
        finalize(fullText) {
            if (!fullText) return ''

            // 完整渲染整个内容
            const finalHtml = renderMarkdownWithLatex(fullText)
            this.reset()

            return finalHtml
        },

        /**
         * 分离式增量渲染（用于实时面试场景，避免闪烁）
         * 返回稳定部分和流式尾部分离的结果
         * @param {string} fullText 当前完整文本
         * @returns {{ stable: string, tail: string }} 分离的渲染结果
         */
        renderSeparated(fullText) {
            if (!fullText) return { stable: '', tail: '' }

            // 如果内容没变或变短了（可能是新对话），重新渲染
            if (fullText.length <= lastFullText.length && fullText !== lastFullText) {
                this.reset()
            }

            // 找到最后一个完整段落的位置
            const completeIndex = findLastCompleteParagraph(fullText)

            if (completeIndex > lastCompleteIndex) {
                // 有新的完整段落，渲染新增部分
                const newComplete = fullText.substring(lastCompleteIndex, completeIndex)
                const newHtml = renderMarkdownWithLatex(newComplete)
                renderedHtml += newHtml
                lastCompleteIndex = completeIndex
            }

            // 处理尾部未完成的文本
            const tail = fullText.substring(lastCompleteIndex)
            let tailHtml = ''
            if (tail.trim()) {
                tailHtml = `<span class="streaming-tail">${renderPartialTail(tail)}</span>`
            }

            lastFullText = fullText

            return { stable: renderedHtml, tail: tailHtml }
        },

        /**
         * 重置状态（新对话时调用）
         */
        reset() {
            lastFullText = ''
            renderedHtml = ''
            lastCompleteIndex = 0
        },

        /**
         * 获取当前渲染的 HTML（不包含尾部）
         */
        getRenderedHtml() {
            return renderedHtml
        }
    }
}

/**
 * 找到最后一个完整段落的结束位置
 * 完整段落定义：以 \n\n 或 \n- 或 \n# 或 \n数字. 结尾
 * 特别处理：代码块必须完整闭合才算完成
 */
function findLastCompleteParagraph(text) {
    // 首先检查是否有未闭合的代码块
    const codeBlockStarts = (text.match(/```/g) || []).length
    const isInCodeBlock = codeBlockStarts % 2 !== 0  // 奇数个 ``` 表示在代码块中

    if (isInCodeBlock) {
        // 在代码块中，找到最后一个代码块开始之前的位置
        const lastCodeBlockStart = text.lastIndexOf('```')
        // 往前找到这个代码块之前的段落边界
        const textBeforeCodeBlock = text.substring(0, lastCodeBlockStart)
        return findParagraphBoundary(textBeforeCodeBlock)
    }

    return findParagraphBoundary(text)
}

/**
 * 在文本中找到最后一个段落边界
 */
function findParagraphBoundary(text) {
    const patterns = [
        /\n\n/g,           // 空行
        /\n(?=[-*+] )/g,   // 列表项
        /\n(?=#{1,6} )/g,  // 标题
        /\n(?=\d+\. )/g,   // 有序列表
        /\n(?=```)/g,      // 代码块开始
        /```\n/g,          // 代码块结束
        /\n(?=\$\$)/g,     // 块级公式
        /\n(?=>)/g,        // 引用
    ]

    let lastIndex = 0

    for (const pattern of patterns) {
        let match
        while ((match = pattern.exec(text)) !== null) {
            const endPos = match.index + match[0].length
            if (endPos > lastIndex) {
                lastIndex = endPos
            }
        }
    }

    return lastIndex
}

/**
 * 轻量渲染流式尾部文本
 * 做基本的段落和行内格式处理，不做 LaTeX（避免不完整公式报错）
 */
function renderPartialTail(text) {
    let html = escapeHtml(text)
    // Bold: **text**
    html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    // Inline code: `code`
    html = html.replace(/`([^`]+)`/g, '<code>$1</code>')
    // Italic: *text*
    html = html.replace(/(?<!\*)\*(?!\*)(.+?)(?<!\*)\*(?!\*)/g, '<em>$1</em>')
    return html
}

/**
 * HTML 转义（用于流式尾部渲染，将换行转为 <br>）
 */
function escapeHtml(text) {
    return text
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        // 合并超过2个的连续换行为2个
        .replace(/\n{3,}/g, '\n\n')
        .replace(/\n/g, '<br>')
}

/**
 * HTML 转义（用于代码块，保留原始换行和空白）
 */
function escapeCodeHtml(text) {
    return text
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
}

// ==================== 代码块渲染器 ====================

const COPY_ICON_SVG = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>'

marked.use({
    renderer: {
        code({ text, lang }) {
            const langLabel = lang ? lang.split(/\s/)[0].toLowerCase() : ''

            // 语法高亮
            let highlighted
            if (langLabel && hljs.getLanguage(langLabel)) {
                try {
                    highlighted = hljs.highlight(text, { language: langLabel }).value
                } catch {
                    highlighted = escapeCodeHtml(text)
                }
            } else if (text.length < 8000) {
                // 未指定语言时尝试自动检测（限制长度避免性能问题）
                try {
                    const result = hljs.highlightAuto(text)
                    highlighted = result.relevance > 5 ? result.value : escapeCodeHtml(text)
                } catch {
                    highlighted = escapeCodeHtml(text)
                }
            } else {
                highlighted = escapeCodeHtml(text)
            }

            const displayLang = langLabel || 'code'

            return `<div class="code-block-container"><div class="code-block-header"><span class="code-lang-tag">${escapeCodeHtml(displayLang)}</span><button class="code-copy-btn">${COPY_ICON_SVG}<span>复制</span></button></div><div class="code-block-content"><pre><code class="hljs${langLabel ? ` language-${langLabel}` : ''}">${highlighted}</code></pre></div></div>`
        }
    }
})

// ==================== 代码块复制交互 ====================

let copyHandlerInitialized = false

/**
 * 初始化代码块复制按钮的事件委托
 * 在 App.vue 的 onMounted 中调用一次即可
 */
export function initCodeBlockInteractions() {
    if (copyHandlerInitialized) return
    copyHandlerInitialized = true

    document.addEventListener('click', (e) => {
        const btn = e.target.closest('.code-copy-btn')
        if (!btn) return

        e.preventDefault()
        e.stopPropagation()

        const container = btn.closest('.code-block-container')
        if (!container) return

        const codeEl = container.querySelector('code')
        if (!codeEl) return

        const code = codeEl.textContent || ''
        const textEl = btn.querySelector('span')

        navigator.clipboard.writeText(code).then(() => {
            btn.classList.add('copied')
            if (textEl) textEl.textContent = '已复制'
            setTimeout(() => {
                btn.classList.remove('copied')
                if (textEl) textEl.textContent = '复制'
            }, 2000)
        }).catch(() => {
            // 降级方案：使用 execCommand
            try {
                const textarea = document.createElement('textarea')
                textarea.value = code
                textarea.style.cssText = 'position:fixed;left:-9999px'
                document.body.appendChild(textarea)
                textarea.select()
                document.execCommand('copy')
                document.body.removeChild(textarea)
                btn.classList.add('copied')
                if (textEl) textEl.textContent = '已复制'
                setTimeout(() => {
                    btn.classList.remove('copied')
                    if (textEl) textEl.textContent = '复制'
                }, 2000)
            } catch { /* ignore */ }
        })
    })
}

/**
 * 清空缓存
 */
export function clearRenderCache() {
    renderCache.clear()
    latexCache.clear()
}

export default {
    renderMarkdownWithLatex,
    createIncrementalRenderer,
    clearRenderCache,
    initCodeBlockInteractions
}
