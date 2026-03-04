// Webapp Settings Store
// Fetches /settings.json at startup and provides reactive state.
// The file uses JSONC (JSON with comments) — single-line // comments are stripped before parsing.

import { reactive } from 'vue'

// White-label safe defaults: empty name so "Plik" is never leaked
// if settings.json is missing or fails to load.
export const settings = reactive({
    name: '',
    logo: '',
    backgroundImage: '',
    backgroundColor: '',
    overlayOpacity: 0,
    customCSS: '',
    customJS: '',
})

/**
 * Strip single-line // comments from JSONC text.
 * Ignores // inside quoted strings.
 */
function stripJSONCComments(text) {
    return text.replace(/("(?:[^"\\]|\\.)*")|\/\/[^\n]*/g, (match, quoted) => {
        return quoted || ''
    })
}

/**
 * Dynamically inject a <link rel="stylesheet"> and wait for it to load.
 */
function injectCSS(href) {
    return new Promise((resolve) => {
        const link = document.createElement('link')
        link.rel = 'stylesheet'
        link.href = href
        link.onload = resolve
        link.onerror = resolve // Don't block on failure
        document.head.appendChild(link)
    })
}

/**
 * Dynamically inject a <script> and wait for it to load.
 */
function injectJS(src) {
    return new Promise((resolve) => {
        const script = document.createElement('script')
        script.src = src
        script.onload = resolve
        script.onerror = resolve // Don't block on failure
        document.head.appendChild(script)
    })
}

/**
 * Load webapp settings from /settings.json before Vue mounts.
 * - Fetches and parses the JSONC file
 * - Merges values into the reactive settings object
 * - Sets document.title from settings.name
 * - Conditionally injects custom CSS/JS if paths are configured
 */
export async function loadSettings() {
    try {
        const resp = await fetch('/settings.json')
        if (resp.ok) {
            const text = await resp.text()
            const data = JSON.parse(stripJSONCComments(text))
            Object.assign(settings, data)
        }
    } catch {
        // Silently fall back to defaults
    }

    // Set page title (stays empty if no custom settings — white-label safe)
    document.title = settings.name

    // Conditionally inject custom assets (before Vue mounts)
    const injections = []
    if (settings.customCSS) {
        injections.push(injectCSS(settings.customCSS))
    }
    if (settings.customJS) {
        injections.push(injectJS(settings.customJS))
    }
    if (injections.length) {
        await Promise.all(injections)
    }
}
