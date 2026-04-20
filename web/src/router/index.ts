import { createRouter, createWebHashHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      redirect: '/best50',
    },
    {
      path: '/profile',
      name: 'profile',
      component: () => import('@/views/ProfileView.vue'),
    },
    {
      path: '/best50',
      name: 'best50',
      component: () => import('@/views/Best50View.vue'),
    },
    {
      path: '/songs',
      name: 'songs',
      component: () => import('@/views/SongsView.vue'),
    },
    {
      path: '/records',
      name: 'records',
      component: () => import('@/views/RecordsView.vue'),
    },
    {
      path: '/admin/songs',
      name: 'admin-songs',
      component: () => import('@/views/AdminSongsView.vue'),
      meta: { requiresAdmin: true },
    },
    {
      path: '/about',
      name: 'about',
      component: () => import('@/views/AboutView.vue'),
    },
  ],
})

router.beforeEach((to) => {
  if (to.meta?.requiresAdmin) {
    const userStore = useUserStore()
    if (!userStore.logged_in || !userStore.is_admin) {
      return { path: '/best50' }
    }
  }
  return true
})

export default router
