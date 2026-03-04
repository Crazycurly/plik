import { test, expect } from './fixtures.js'

test.describe('Customization — settings.json', () => {
    test('default settings: logo text and page title are "Plik"', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Logo text should be "Plik" (from the default settings.json)
        const logo = page.locator('.plik-logo-text').first()
        await expect(logo).toHaveText('Plik')

        // Page title should be "Plik"
        await expect(page).toHaveTitle('Plik')
    })

    test('custom name via settings.json override', async ({ page }) => {
        // Intercept /settings.json and return custom settings
        await page.route('**/settings.json', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    name: 'MyFileShare',
                    backgroundImage: '',
                    backgroundColor: '',
                    overlayOpacity: 0.2,
                    customCSS: '',
                    customJS: '',
                }),
            })
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Logo text should be the custom name
        const logo = page.locator('.plik-logo-text').first()
        await expect(logo).toHaveText('MyFileShare')

        // Page title should match
        await expect(page).toHaveTitle('MyFileShare')
    })

    test('missing settings.json falls back to empty name (white-label safe)', async ({ page }) => {
        // Intercept settings.json with a 404
        await page.route('**/settings.json', async (route) => {
            await route.fulfill({ status: 404, body: '' })
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Logo text should be empty (never leaks "Plik")
        const logo = page.locator('.plik-logo-text').first()
        await expect(logo).toHaveText('')

        // Page title should be empty
        await expect(page).toHaveTitle('')
    })
})

test.describe('Customization — custom CSS', () => {
    test('custom CSS is injected when customCSS is set', async ({ page }) => {
        // Intercept settings.json to enable custom CSS
        await page.route('**/settings.json', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    name: 'Plik',
                    customCSS: '/css/test-custom.css',
                    customJS: '',
                }),
            })
        })

        // Serve the custom CSS that sets a distinctive background color
        await page.route('**/css/test-custom.css', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'text/css',
                body: 'body { background-color: rgb(255, 0, 255) !important; }',
            })
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Verify the custom CSS was applied
        const bgColor = await page.evaluate(() => {
            return window.getComputedStyle(document.body).backgroundColor
        })
        expect(bgColor).toBe('rgb(255, 0, 255)')
    })

    test('no CSS injected when customCSS is empty', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // No custom stylesheet should be injected (only the app's own styles)
        const customLinks = await page.evaluate(() => {
            return Array.from(document.querySelectorAll('link[rel="stylesheet"]'))
                .filter(link => link.href.includes('custom'))
                .length
        })
        expect(customLinks).toBe(0)
    })
})

test.describe('Customization — custom JS', () => {
    test('custom JS is injected when customJS is set', async ({ page }) => {
        // Intercept settings.json to enable custom JS
        await page.route('**/settings.json', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    name: 'Plik',
                    customCSS: '',
                    customJS: '/js/test-custom.js',
                }),
            })
        })

        // Serve a custom JS that sets a global marker
        await page.route('**/js/test-custom.js', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/javascript',
                body: 'window.__CUSTOM_JS_LOADED__ = true;',
            })
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Verify the custom script was executed
        const loaded = await page.evaluate(() => window.__CUSTOM_JS_LOADED__)
        expect(loaded).toBe(true)
    })
})
