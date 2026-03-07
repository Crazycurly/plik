import { createApp } from 'vue'
import App from './App.vue'
import router from './router.js'
import { loadConfig } from './config.js'
import { loadSettings } from './settings.js'
import { checkSession } from './authStore.js'
import './style.css'

const app = createApp(App)

// Load server config, webapp settings, and check auth session before installing the router.
// The router must be installed AFTER config loads because navigation guards
// rely on config values (e.g. feature_authentication for forced-auth redirect).
// Settings must load before mount to avoid flicker (name, background, custom CSS/JS).
Promise.all([loadConfig(), loadSettings(), checkSession()]).then(() => {
    app.use(router)
    app.mount('#app')
})
