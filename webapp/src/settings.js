// Webapp Settings Store
// Fetches /settings.json at startup and provides reactive state.
// The file uses JSONC (JSON with comments) — single-line // comments are stripped before parsing.

import { reactive } from 'vue'

// White-label safe defaults: empty name so "Plik" is never leaked
// if settings.json is missing or fails to load.
export const settings = reactive({
    name: '',
    logo: '',
    theme: 'auto',
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
 * Resolve and apply the theme.
 * - "dark" / "light" / "my-custom" → set directly
 * - "auto" → follow OS preference via prefers-color-scheme
 *
 * Non-"dark" themes lazy-load their CSS from /themes/{name}.css
 * before setting the data-theme attribute (prevents unstyled flash).
 */
const loadedThemes = new Set()

async function applyTheme(value) {
    const mq = window.matchMedia('(prefers-color-scheme: dark)')

    async function resolve() {
        const name = value === 'auto'
            ? (mq.matches ? 'dark' : 'light')
            : value

        // "dark" is the compiled-in default — no external CSS needed.
        // All other themes (including "light") load from /themes/{name}.css.
        if (name !== 'dark') {
            const href = `/themes/${name}.css`
            if (!loadedThemes.has(href)) {
                await injectCSS(href)
                loadedThemes.add(href)
            }
        }

        document.documentElement.dataset.theme = name
        document.documentElement.style.colorScheme = name === 'dark' ? 'dark' : ''
    }

    await resolve()

    // Live-switch when the user toggles OS dark mode (only for "auto")
    if (value === 'auto') {
        mq.addEventListener('change', resolve)
    }
}

/**
 * Load webapp settings from /settings.json before Vue mounts.
 * - Fetches and parses the JSONC file
 * - Merges values into the reactive settings object
 * - Applies theme (before Vue mounts → zero flash)
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

    // Apply theme before anything renders
    applyTheme(settings.theme)

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
