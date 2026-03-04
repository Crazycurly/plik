import { test, expect } from './fixtures.js'

/**
 * Helper: intercept settings.json with the given themes config.
 */
async function withThemes(page, themes, extra = {}) {
    await page.route('**/settings.json', async (route) => {
        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
                name: 'Plik',
                theme: 'auto',
                themes,
                ...extra,
            }),
        })
    })
}



test.describe('Theme Picker — visibility', () => {
    test('picker is visible when multiple themes are available (default: all built-ins)', async ({ page }) => {
        await withThemes(page, [])  // empty = all built-ins
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#theme-picker-toggle')
        await expect(picker).toBeVisible()
    })

    test('picker is hidden when only one theme is configured', async ({ page }) => {
        await withThemes(page, ['dark'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#theme-picker-toggle')
        await expect(picker).toHaveCount(0)
    })

    test('picker is hidden when themes is an empty-equivalent single entry', async ({ page }) => {
        await withThemes(page, ['auto'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#theme-picker-toggle')
        await expect(picker).toHaveCount(0)
    })

    test('picker shows when exactly two themes configured', async ({ page }) => {
        await withThemes(page, ['dark', 'light'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#theme-picker-toggle')
        await expect(picker).toBeVisible()
    })
})

test.describe('Theme Picker — dropdown', () => {
    test('clicking opens dropdown with theme options', async ({ page }) => {
        await withThemes(page, ['dark', 'light', 'nord'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Dropdown should not be visible initially
        await expect(page.locator('#theme-option-dark')).toHaveCount(0)

        // Click the picker toggle
        await page.locator('#theme-picker-toggle').click()

        // Dropdown should now show the configured themes
        await expect(page.locator('#theme-option-dark')).toBeVisible()
        await expect(page.locator('#theme-option-light')).toBeVisible()
        await expect(page.locator('#theme-option-nord')).toBeVisible()

        // Themes NOT in the list should not appear
        await expect(page.locator('#theme-option-matrix')).toHaveCount(0)
    })

    test('clicking outside closes the dropdown', async ({ page }) => {
        await withThemes(page, [])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#theme-picker-toggle').click()
        await expect(page.locator('#theme-option-dark')).toBeVisible()

        // Click outside (on the body)
        await page.locator('body').click({ position: { x: 10, y: 300 } })

        // Dropdown should close
        await expect(page.locator('#theme-option-dark')).toHaveCount(0)
    })

    test('selecting a theme closes the dropdown', async ({ page }) => {
        await withThemes(page, [])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#theme-picker-toggle').click()
        await page.locator('#theme-option-light').click()

        // Dropdown should close after selection
        await expect(page.locator('#theme-option-light')).toHaveCount(0)
    })
})

test.describe('Theme Picker — theme application', () => {
    test('selecting a theme applies it via data-theme', async ({ page }) => {
        await withThemes(page, [])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#theme-picker-toggle').click()
        await page.locator('#theme-option-light').click()

        const theme = await page.evaluate(() => document.documentElement.dataset.theme)
        expect(theme).toBe('light')
    })

    test('selected theme injects the CSS file', async ({ page }) => {
        await withThemes(page, ['dark', 'light', 'nord'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#theme-picker-toggle').click()
        const nordBtn = page.locator('#theme-option-nord')
        await expect(nordBtn).toBeVisible()
        await nordBtn.click()

        // Wait for dropdown to close (confirms click was processed)
        await expect(nordBtn).toHaveCount(0)

        // Use auto-retrying assertion for data-theme
        await expect(page.locator('html')).toHaveAttribute('data-theme', 'nord')

        const link = page.locator('link[href*="nord.css"]')
        await expect(link).toHaveCount(1)
    })

    test('active theme shows checkmark', async ({ page }) => {
        await withThemes(page, [])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Open picker and select nord
        await page.locator('#theme-picker-toggle').click()
        await page.locator('#theme-option-nord').click()

        // Reopen picker — nord should have the accent color (checkmark)
        await page.locator('#theme-picker-toggle').click()
        const nordOption = page.locator('#theme-option-nord')
        await expect(nordOption).toHaveClass(/text-accent-400/)
    })
})

test.describe('Theme Picker — localStorage persistence', () => {
    test('selected theme persists across page reload', async ({ page }) => {
        await withThemes(page, [])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Select light theme
        await page.locator('#theme-picker-toggle').click()
        await page.locator('#theme-option-light').click()

        // Verify localStorage was set
        const stored = await page.evaluate(() => localStorage.getItem('plik-theme'))
        expect(stored).toBe('light')

        // Reload the page (route intercept persists)
        await page.reload({ waitUntil: 'networkidle' })

        // Theme should still be light
        const theme = await page.evaluate(() => document.documentElement.dataset.theme)
        expect(theme).toBe('light')
    })

    test('localStorage theme overrides settings.json default', async ({ page }) => {
        // Pre-seed localStorage before any navigation
        await page.addInitScript(() => localStorage.setItem('plik-theme', 'nord'))

        await withThemes(page, [])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // data-theme should be 'nord' from localStorage despite settings.json default being 'auto'
        await expect(page.locator('html')).toHaveAttribute('data-theme', 'nord')
    })
})

test.describe('Theme Picker — custom theme objects', () => {
    test('supports object entries with custom labels', async ({ page }) => {
        await withThemes(page, [
            'dark',
            { name: 'nord', label: 'Arctic Night' },
        ])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#theme-picker-toggle').click()

        // The custom label should appear
        const nordOption = page.locator('#theme-option-nord')
        await expect(nordOption).toBeVisible()
        await expect(nordOption).toContainText('Arctic Night')
    })
})

test.describe('Theme Picker — layout', () => {
    test('has "Theme" text label in the button', async ({ page }) => {
        await withThemes(page, [])
        // Use wider viewport so the desktop nav is visible
        await page.setViewportSize({ width: 1280, height: 720 })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#theme-picker-toggle')
        await expect(picker).toBeVisible()
        await expect(picker).toContainText('Theme')
    })
})
