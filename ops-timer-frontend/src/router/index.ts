import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/Login.vue'),
      meta: { requiresAuth: false },
    },
    {
      path: '/',
      component: () => import('@/views/Layout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          redirect: '/dashboard',
        },
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('@/views/Dashboard.vue'),
        },
        {
          path: 'units',
          name: 'units',
          component: () => import('@/views/Units/UnitList.vue'),
        },
        {
          path: 'units/:id',
          name: 'unit-detail',
          component: () => import('@/views/Units/UnitDetail.vue'),
        },
        {
          path: 'projects',
          name: 'projects',
          component: () => import('@/views/Projects/ProjectList.vue'),
        },
        {
          path: 'projects/:id',
          name: 'project-detail',
          component: () => import('@/views/Projects/ProjectDetail.vue'),
        },
        {
          path: 'todos',
          name: 'todos',
          component: () => import('@/views/Todos/TodoList.vue'),
        },
        {
          path: 'schedule',
          name: 'schedule',
          component: () => import('@/views/Schedule/ScheduleCalendar.vue'),
        },
        {
          path: 'budget',
          name: 'budget',
          component: () => import('@/views/Budget/BudgetPage.vue'),
        },
        {
          path: 'secrets',
          name: 'secrets',
          component: () => import('@/views/Secrets/SecretList.vue'),
        },
        {
          path: 'notifications',
          name: 'notifications',
          component: () => import('@/views/Notifications.vue'),
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/Settings/SettingsPage.vue'),
        },
      ],
    },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()

  // 用 matched 检查整条路由链，只要链中有任意一条显式标记 requiresAuth: false，就视为公开路由
  const isPublicRoute = to.matched.some(record => record.meta.requiresAuth === false)

  if (!isPublicRoute && !auth.isLoggedIn) {
    // 保存目标路径，登录后可跳回
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  if (to.name === 'login' && auth.isLoggedIn) {
    return { name: 'dashboard' }
  }
})

export default router
