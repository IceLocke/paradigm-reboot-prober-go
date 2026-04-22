<template>
  <div class="page-container">
    <div class="health-content">
      <div class="health-hero">
        <h1 class="health-title">{{ t('health.title') }}</h1>
        <p class="health-subtitle">{{ t('health.subtitle') }}</p>
      </div>

      <BaseCard>
        <div class="health-status">
          <div class="status-indicator">
            <span :class="['status-dot', `status-dot--${statusKind}`]" />
            <span class="status-label">{{ statusLabel }}</span>
          </div>

          <dl class="status-details">
            <div class="status-row">
              <dt>{{ t('health.endpoint') }}</dt>
              <dd class="status-endpoint">
                <a :href="HEALTH_URL" target="_blank" rel="noopener noreferrer">{{ HEALTH_URL }}</a>
              </dd>
            </div>
            <div class="status-row">
              <dt>{{ t('health.http_status') }}</dt>
              <dd>{{ httpStatus ?? '—' }}</dd>
            </div>
            <div class="status-row">
              <dt>{{ t('health.latency') }}</dt>
              <dd>{{ latencyMs != null ? `${latencyMs} ms` : '—' }}</dd>
            </div>
            <div class="status-row">
              <dt>{{ t('health.checked_at') }}</dt>
              <dd>{{ checkedAt ? checkedAt.toLocaleString() : '—' }}</dd>
            </div>
            <div v-if="responseText" class="status-row status-row--block">
              <dt>{{ t('health.response') }}</dt>
              <dd><pre class="response-body">{{ responseText }}</pre></dd>
            </div>
            <div v-if="errorMessage" class="status-row status-row--block">
              <dt>{{ t('health.error') }}</dt>
              <dd class="error-text">{{ errorMessage }}</dd>
            </div>
          </dl>

          <div class="status-actions">
            <BaseButton
              variant="secondary"
              :disabled="loading"
              @click="check"
            >
              {{ loading ? t('common.loading') : t('common.refresh') }}
            </BaseButton>
          </div>
        </div>
      </BaseCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseCard from '@/components/ui/BaseCard.vue'
import BaseButton from '@/components/ui/BaseButton.vue'

const { t } = useI18n()

const HEALTH_URL = 'https://api.prp.icel.site/healthz'

const loading = ref(false)
const httpStatus = ref<number | null>(null)
const latencyMs = ref<number | null>(null)
const checkedAt = ref<Date | null>(null)
const responseText = ref<string>('')
const errorMessage = ref<string>('')

type StatusKind = 'idle' | 'healthy' | 'unhealthy' | 'unreachable'
const statusKind = computed<StatusKind>(() => {
  if (loading.value && checkedAt.value === null) return 'idle'
  if (errorMessage.value) return 'unreachable'
  if (httpStatus.value !== null && httpStatus.value >= 200 && httpStatus.value < 300) return 'healthy'
  if (httpStatus.value !== null) return 'unhealthy'
  return 'idle'
})

const statusLabel = computed(() => {
  switch (statusKind.value) {
    case 'healthy':
      return t('health.status.healthy')
    case 'unhealthy':
      return t('health.status.unhealthy')
    case 'unreachable':
      return t('health.status.unreachable')
    default:
      return t('health.status.checking')
  }
})

async function check() {
  loading.value = true
  errorMessage.value = ''
  responseText.value = ''
  httpStatus.value = null
  latencyMs.value = null

  const start = performance.now()
  try {
    const resp = await fetch(HEALTH_URL, { method: 'GET', cache: 'no-store' })
    latencyMs.value = Math.round(performance.now() - start)
    httpStatus.value = resp.status
    const text = await resp.text()
    responseText.value = text.trim().length > 0 ? text : t('health.empty_body')
  } catch (err) {
    latencyMs.value = Math.round(performance.now() - start)
    errorMessage.value = err instanceof Error ? err.message : String(err)
  } finally {
    checkedAt.value = new Date()
    loading.value = false
  }
}

onMounted(() => {
  void check()
})
</script>

<style scoped>
.health-content {
  max-width: 640px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  padding-top: var(--space-8);
}
.health-hero {
  text-align: center;
}
.health-title {
  font-size: var(--text-3xl);
  font-weight: 700;
  margin-bottom: var(--space-2);
}
.health-subtitle {
  font-size: var(--text-lg);
  color: var(--text-muted);
}

.health-status {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.status-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--text-muted);
  box-shadow: 0 0 0 4px rgba(255, 255, 255, 0.04);
}
.status-dot--idle {
  background: var(--text-muted);
  animation: pulse 1.4s ease-in-out infinite;
}
.status-dot--healthy {
  background: #22c55e;
  box-shadow: 0 0 0 4px rgba(34, 197, 94, 0.15);
}
.status-dot--unhealthy {
  background: #f59e0b;
  box-shadow: 0 0 0 4px rgba(245, 158, 11, 0.15);
}
.status-dot--unreachable {
  background: var(--color-danger, #ef4444);
  box-shadow: 0 0 0 4px rgba(239, 68, 68, 0.15);
}
.status-label {
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--text-primary);
}

.status-details {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin: 0;
}
.status-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: var(--space-4);
  font-size: var(--text-base);
}
.status-row dt {
  color: var(--text-muted);
  font-weight: 500;
  flex-shrink: 0;
}
.status-row dd {
  margin: 0;
  color: var(--text-primary);
  text-align: right;
  word-break: break-all;
}
.status-row--block {
  flex-direction: column;
  align-items: stretch;
  gap: var(--space-2);
}
.status-row--block dd {
  text-align: left;
}
.status-endpoint a {
  color: var(--accent);
  text-decoration: none;
}
@media (hover: hover) {
  .status-endpoint a:hover {
    text-decoration: underline;
  }
}
.response-body {
  margin: 0;
  padding: var(--space-3);
  background: var(--bg-tertiary, rgba(255, 255, 255, 0.04));
  border: 1px solid var(--border);
  border-radius: 6px;
  font-family: var(--font-mono, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace);
  font-size: var(--text-sm);
  color: var(--text-secondary);
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 240px;
  overflow: auto;
}
.error-text {
  color: var(--color-danger, #ef4444);
}

.status-actions {
  display: flex;
  justify-content: flex-end;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
</style>
