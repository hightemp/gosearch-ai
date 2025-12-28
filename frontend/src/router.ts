import type { RouteRecordRaw } from 'vue-router'
import HomePage from './pages/HomePage.vue'
import ChatPage from './pages/ChatPage.vue'

export const routes: RouteRecordRaw[] = [
  { path: '/', name: 'home', component: HomePage },
  { path: '/chat/:chatId', name: 'chat', component: ChatPage }
]

