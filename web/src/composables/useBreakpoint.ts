import { ref, onMounted, onUnmounted } from 'vue'

export function useBreakpoint() {
  const width = ref(window.innerWidth)
  const isMobile = ref(window.innerWidth < 768)
  const isDesktop = ref(window.innerWidth >= 768)

  const update = () => {
    width.value = window.innerWidth
    isMobile.value = window.innerWidth < 768
    isDesktop.value = window.innerWidth >= 768
  }

  onMounted(() => window.addEventListener('resize', update))
  onUnmounted(() => window.removeEventListener('resize', update))

  return { width, isMobile, isDesktop }
}
