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
 * `mermaid.initialize()` runs once (guarded by `mermaidReady`).
 * The theme is chosen dynamically from <html>'s colorScheme at init time.
 *
 * @param {HTMLElement} container - Parent element containing .mermaid divs
 */
let mermaidReady = false

export async function initMermaidInElement(container) {
    if (!container) return
    const nodes = container.querySelectorAll('.mermaid:not([data-processed])')
    if (!nodes.length) return

    const { default: mermaid } = await import('mermaid')
    if (!mermaidReady) {
        const isDark = getComputedStyle(document.documentElement).colorScheme !== 'light'
        mermaid.initialize({
            startOnLoad: false,
            theme: isDark ? 'dark' : 'default',
            securityLevel: 'strict',
        })
        mermaidReady = true
    }
    await mermaid.run({ nodes })
}
