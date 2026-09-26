import { describe, expect, it } from 'vitest'
import { renderMarkdownWithLatex } from './markdown-latex'

describe('renderMarkdownWithLatex', () => {
  it('removes scripts, HTML tags, and event handlers', () => {
    const html = renderMarkdownWithLatex('<img src=x onerror=alert(1)><script>alert(1)</script>')
    expect(html).not.toContain('<script')
    expect(html).not.toContain('<img')
  })

  it('removes dangerous link protocols', () => {
    const html = renderMarkdownWithLatex('[click](javascript:alert(1))')
    expect(html).not.toContain('javascript:')
  })

  it('preserves generated markdown, code, and math markup', () => {
    const html = renderMarkdownWithLatex('**bold**\n\n```js\nconst x = 1\n```\n\n$x^2$')
    expect(html).toContain('<strong>bold</strong>')
    expect(html).toContain('code-block-container')
    expect(html).toContain('katex')
  })
})
