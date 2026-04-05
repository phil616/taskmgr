<template>
  <v-layout>
    <v-navigation-drawer v-model="drawer" :rail="rail">
      <v-list-item
        prepend-icon="mdi-timer-cog-outline"
        title="任务管理器"
        class="py-3"
        @click="rail = !rail"
      />
      <v-divider />

      <v-list density="compact" nav color="primary">
        <v-list-item
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          :prepend-icon="item.icon"
          :title="item.title"
          rounded="lg"
          class="my-1"
        />
      </v-list>

      <template v-slot:append>
        <v-list density="compact" nav>
          <v-list-item
            prepend-icon="mdi-cog"
            title="设置"
            to="/settings"
            rounded="lg"
          />
        </v-list>
      </template>
    </v-navigation-drawer>

    <v-app-bar elevation="0" border="b">
      <v-app-bar-nav-icon @click="drawer = !drawer" class="d-md-none" />
      <v-app-bar-title class="text-body-1 font-weight-medium">
        {{ currentTitle }}
      </v-app-bar-title>

      <template v-slot:append>
        <v-btn icon to="/notifications" class="mr-1">
          <v-badge
            :content="notificationStore.unreadCount"
            :model-value="notificationStore.unreadCount > 0"
            color="error"
          >
            <v-icon>mdi-bell-outline</v-icon>
          </v-badge>
        </v-btn>

        <v-menu>
          <template v-slot:activator="{ props }">
            <v-btn icon v-bind="props">
              <v-avatar color="primary" size="32">
                <span class="text-body-2 font-weight-bold text-white">
                  {{ auth.user?.display_name?.charAt(0) || 'A' }}
                </span>
              </v-avatar>
            </v-btn>
          </template>
          <v-list density="compact" min-width="160">
            <v-list-item prepend-icon="mdi-cog" title="设置" to="/settings" />
            <v-divider />
            <v-list-item prepend-icon="mdi-logout" title="退出登录" @click="handleLogout" />
          </v-list>
        </v-menu>
      </template>
    </v-app-bar>

    <v-main>
      <v-container fluid class="pa-4 pa-md-6">
        <router-view v-slot="{ Component }">
          <transition name="route-fade" mode="out-in">
            <component :is="Component" :key="route.path" />
          </transition>
        </router-view>
      </v-container>
    </v-main>
  </v-layout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDisplay } from 'vuetify'
import { useAuthStore } from '@/stores/auth'
import { useNotificationStore } from '@/stores/notifications'

const route = useRoute()
const router = useRouter()
const { mdAndUp } = useDisplay()
const auth = useAuthStore()
const notificationStore = useNotificationStore()

const drawer = ref(true)
const rail = ref(false)

// 小屏幕默认收起抽屉，同时启动通知轮询
onMounted(() => {
  drawer.value = mdAndUp.value
  notificationStore.startPolling()
})

const navItems = [
  { title: '仪表盘', icon: 'mdi-view-dashboard', to: '/dashboard' },
  { title: '计时单元', icon: 'mdi-timer', to: '/units' },
  { title: '项目管理', icon: 'mdi-folder-outline', to: '/projects' },
  { title: '待办事项', icon: 'mdi-checkbox-marked-outline', to: '/todos' },
  { title: '日程管理', icon: 'mdi-calendar-month', to: '/schedule' },
  { title: '预算管理', icon: 'mdi-wallet', to: '/budget' },
  { title: '密钥管理', icon: 'mdi-key-variant', to: '/secrets' },
  { title: '通知中心', icon: 'mdi-bell-outline', to: '/notifications' },
]

const titleMap: Record<string, string> = {
  dashboard: '仪表盘',
  units: '计时单元',
  'unit-detail': '单元详情',
  projects: '项目管理',
  'project-detail': '项目详情',
  todos: '待办事项',
  schedule: '日程管理',
  budget: '预算管理',
  secrets: '密钥管理',
  notifications: '通知中心',
  settings: '系统设置',
}

const currentTitle = computed(() => {
  return titleMap[route.name as string] || '任务管理器'
})

async function handleLogout() {
  await auth.logout()
  router.push('/login')
}

onUnmounted(() => {
  notificationStore.stopPolling()
})
</script>

<style scoped>
.route-fade-enter-active,
.route-fade-leave-active {
  transition: opacity 0.12s ease;
}

.route-fade-enter-from,
.route-fade-leave-to {
  opacity: 0;
}
</style>
