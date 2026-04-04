import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      redirect: '/best50',
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
      path: '/about',
      name: 'about',
      component: () => import('@/views/AboutView.vue'),
    },
  ],
})

export default router
