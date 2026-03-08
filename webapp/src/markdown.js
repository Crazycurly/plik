import { marked } from 'marked'
import DOMPurify from 'dompurify'

/**
 * Custom renderer that converts ```mermaid code blocks into
 * <div class="mermaid"> containers (instead of <pre><code>).
 * Mermaid.run() will later transform them into SVGs.
 */
const renderer = {
    code({ text, lang }) {
        if (lang === 'mermaid') {
            return `<div class="mermaid">${text}</div>\n`
        }
        // Fall through to default marked behaviour
        return false
    },
}

marked.use({ renderer })

/**
 * Render Markdown text to sanitized HTML.
 *
 * Uses DOMPurify to prevent XSS from user-supplied content
 * (e.g. upload comments rendered via v-html).
 *
 * @param {string} text - Raw Markdown text
 * @returns {string} Sanitized HTML string
 */
export function renderMarkdown(text) {
    if (!text) return ''
    const html = marked.parse(text, { breaks: true })
    return DOMPurify.sanitize(html)
}

/**
 * Lazy-load Mermaid and render all `.mermaid` divs inside a container.
 *
 * Call this AFTER Vue has injected rendered HTML via v-html + nextTick
 * so the DOM nodes actually exist.
 *
 * @param {HTMLElement} container - Parent element containing .mermaid divs
 */
export async function initMermaidInElement(container) {
    if (!container) return
    const nodes = container.querySelectorAll('.mermaid:not([data-processed])')
    if (!nodes.length) return

    const { default: mermaid } = await import('mermaid')
    mermaid.initialize({
        startOnLoad: false,
        theme: 'dark',
        securityLevel: 'strict',
    })
    await mermaid.run({ nodes })
}
