import { createApp } from 'vue'
import { createPinia } from 'pinia'
import vuetify from './plugins/vuetify'
import router from './router'
import App from './App.vue'

if (import.meta.env.VITE_DESKTOP === 'true') {
  document.documentElement.classList.add('desktop-client')
  void import('./desktop.css')
}

const app = createApp(App)
console.info(import.meta.env.VITE_COMMIT_SHA)
app.use(createPinia())
app.use(router)
app.use(vuetify)

app.mount('#app')
