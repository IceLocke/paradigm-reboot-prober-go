import { computed } from 'vue'
import { createDiscreteApi, darkTheme } from 'naive-ui'
import type { ConfigProviderProps } from 'naive-ui'
import { themeOverrides } from '@/config/naive-theme'

/**
 * Discrete API — provides Naive UI's message / notification / loadingBar
 * outside of Vue component setup() context (e.g. in Axios interceptors,
 * Pinia actions, plain utility functions).
 *
 * Using createDiscreteApi means we do NOT need <n-message-provider> or
 * <n-notification-provider> in App.vue.  Theme overrides are kept in sync
 * via configProviderProps.
 *
 * @see https://www.naiveui.com/en-US/os-theme/docs/discrete-api
 */
const configProviderPropsRef = computed<ConfigProviderProps>(() => ({
  theme: darkTheme,
  themeOverrides,
}))

const { message, notification, loadingBar } = createDiscreteApi(
  ['message', 'notification', 'loadingBar'],
  { configProviderProps: configProviderPropsRef },
)

export { message, notification, loadingBar }
