import { describe, it, expect, beforeEach, afterAll } from 'vitest'
import { resolveAutoLanguage, settings } from '../settings.js'

// ── resolveAutoLanguage ──
// Maps the browser's navigator.language to a supported locale.
// Of note: Chinese is disambiguated by script/region — Traditional (Hant / TW / HK / MO)
// resolves to zh_TW, everything else Chinese to Simplified zh.

const originalLanguage = Object.getOwnPropertyDescriptor(navigator, 'language')

function setNavigatorLanguage(value) {
    Object.defineProperty(navigator, 'language', { value, configurable: true })
}

beforeEach(() => {
    // Default: all built-in languages available (zh and zh_TW both present).
    settings.languages = ['*']
})

afterAll(() => {
    if (originalLanguage) {
        Object.defineProperty(navigator, 'language', originalLanguage)
    }
})

describe('resolveAutoLanguage', () => {
    it('routes Traditional Chinese tags to zh_TW', () => {
        for (const tag of ['zh-TW', 'zh-HK', 'zh-MO', 'zh-Hant', 'zh-Hant-TW', 'zh-hant', 'zh-Hant-HK']) {
            setNavigatorLanguage(tag)
            expect(resolveAutoLanguage()).toBe('zh_TW')
        }
    })

    it('routes Simplified / generic Chinese tags to zh', () => {
        for (const tag of ['zh-CN', 'zh-SG', 'zh', 'zh-Hans', 'zh-Hans-CN']) {
            setNavigatorLanguage(tag)
            expect(resolveAutoLanguage()).toBe('zh')
        }
    })

    it('falls back to zh when zh_TW is not an available language', () => {
        settings.languages = ['en', 'zh']
        setNavigatorLanguage('zh-TW')
        expect(resolveAutoLanguage()).toBe('zh')
    })

    it('matches other languages by their base code', () => {
        setNavigatorLanguage('fr-FR')
        expect(resolveAutoLanguage()).toBe('fr')
        setNavigatorLanguage('de')
        expect(resolveAutoLanguage()).toBe('de')
        setNavigatorLanguage('en-US')
        expect(resolveAutoLanguage()).toBe('en')
    })

    it('falls back to en for unsupported languages', () => {
        setNavigatorLanguage('xx-YY')
        expect(resolveAutoLanguage()).toBe('en')
    })
})
