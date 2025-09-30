import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import router from './router'
import './style.css'
import App from './App.vue'
import pl from './locales/pl.json'
import { register as registerServiceWorker } from './registerServiceWorker'

const i18n = createI18n({
  locale: 'pl',
  fallbackLocale: 'pl',
  messages: { pl }
})

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(i18n)
app.mount('#app')

// Register service worker for PWA
if (import.meta.env.PROD) {
  registerServiceWorker()
}