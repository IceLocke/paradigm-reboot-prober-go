<template>
  <transition name="fade">
    <div v-if="visible" class="version-banner">
      <div class="banner-header">
        <div class="banner-title-row">
          <Sparkles :size="16" class="banner-icon" />
          <span class="banner-title">{{ t('announce.new_version', { version: latestVersion }) }}</span>
        </div>
        <button class="banner-close" @click="confirmDismiss">
          <span class="dismiss-hint">{{ t('announce.dismiss') }}</span>
          <X :size="14" />
        </button>
      </div>
      <div class="banner-body">
        <div class="song-scroll">
          <ChartCoverCard
            v-for="chart in displayCharts"
            :key="chart.id"
            :cover="chart.cover"
            :title="chart.title"
            :difficulty="chart.difficulty"
            :level="chart.level"
            @click="openUpload(chart)"
          />
        </div>
      </div>
    </div>
  </transition>

  <QuickUploadModal
    v-model:show="showUpload"
    :title="uploadTarget.title"
    :difficulty="uploadTarget.difficulty"
    :level="uploadTarget.level"
    :chart-id="uploadTarget.chartId"
  />

  <n-modal
    :show="showConfirm"
    preset="card"
    :title="t('common.confirm')"
    style="width: 400px; max-width: 95vw;"
    :bordered="false"
    :auto-focus="false"
    @update:show="showConfirm = $event"
  >
    <div class="confirm-body">
      <p>{{ t('announce.dismiss_confirm') }}</p>
      <div class="confirm-actions">
        <button type="button" class="btn btn--secondary" @click="showConfirm = false">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn--primary" @click="dismiss">{{ t('common.confirm') }}</button>
      </div>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NModal } from 'naive-ui'
import { Sparkles, X } from '@lucide/vue'

import { useAppStore } from '@/stores/app'
import type { ChartInfo, Difficulty } from '@/api/types'
import ChartCoverCard from '@/components/business/ChartCoverCard.vue'
import QuickUploadModal from '@/components/business/QuickUploadModal.vue'

const { t } = useI18n()
const appStore = useAppStore()

function compareVersion(a: string, b: string): number {
  const pa = a.split('.').map(Number)
  const pb = b.split('.').map(Number)
  const len = Math.max(pa.length, pb.length)
  for (let i = 0; i < len; i++) {
    const na = pa[i] ?? 0
    const nb = pb[i] ?? 0
    if (na !== nb) return na - nb
  }
  return 0
}

const latestVersion = computed(() => {
  if (!appStore.charts || appStore.charts.length === 0) return ''
  let best = appStore.charts[0].version
  for (const c of appStore.charts) {
    if (compareVersion(c.version, best) > 0) best = c.version
  }
  return best
})

const displayCharts = computed(() => {
  if (!appStore.charts || !latestVersion.value) return []

  const versionCharts = appStore.charts.filter(
    (c) => c.version === latestVersion.value
  )

  // Group by song_id
  const songMap = new Map<number, ChartInfo[]>()
  for (const c of versionCharts) {
    const arr = songMap.get(c.song_id) ?? []
    arr.push(c)
    songMap.set(c.song_id, arr)
  }

  // For each song: take massive; if reboot exists, also include it
  const result: ChartInfo[] = []
  for (const [, charts] of songMap) {
    const massive = charts.find((c) => c.difficulty === 'massive')
    const reboot = charts.find((c) => c.difficulty === 'reboot')
    if (massive) result.push(massive)
    if (reboot) result.push(reboot)
  }

  // Sort: reboot first (more noteworthy), then by song_id for stability
  result.sort((a, b) => {
    const diffOrder: Record<string, number> = { reboot: 0, massive: 1 }
    const da = diffOrder[a.difficulty] ?? 2
    const db = diffOrder[b.difficulty] ?? 2
    if (da !== db) return da - db
    return a.song_id - b.song_id
  })

  return result
})

const visible = computed(() => {
  return (
    latestVersion.value !== '' &&
    displayCharts.value.length > 0 &&
    appStore.dismissedVersion !== latestVersion.value
  )
})

const showUpload = ref(false)
const showConfirm = ref(false)
const uploadTarget = ref({ title: '', difficulty: 'detected' as Difficulty, level: 0, chartId: 0 })

function openUpload(chart: ChartInfo) {
  uploadTarget.value = {
    title: chart.title,
    difficulty: chart.difficulty,
    level: chart.level,
    chartId: chart.id,
  }
  showUpload.value = true
}

function confirmDismiss() {
  showConfirm.value = true
}

function dismiss() {
  appStore.dismissedVersion = latestVersion.value
  showConfirm.value = false
}
</script>

<style scoped>
.version-banner {
  background: linear-gradient(180deg, rgba(59, 130, 246, 0.06) 0%, var(--bg-card) 60%);
  border: 1px solid var(--border);
  border-radius: 10px;
  margin-bottom: var(--space-6);
  overflow: hidden;
}

.banner-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-5);
  border-bottom: 1px solid var(--border);
}

.banner-title-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.banner-icon {
  color: var(--accent);
  flex-shrink: 0;
}

.banner-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--text-primary);
}

.banner-close {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  border-radius: 6px;
  padding: 4px 8px;
  transition: background var(--transition-fast), color var(--transition-fast);
  flex-shrink: 0;
}
.dismiss-hint {
  font-size: var(--text-xs);
  color: var(--text-muted);
}
@media (hover: hover) {
  .banner-close:hover {
    background: rgba(255, 255, 255, 0.06);
    color: var(--text-secondary);
  }
}

.banner-body {
  padding: var(--space-4) var(--space-5);
}

.song-scroll {
  display: flex;
  gap: var(--space-4);
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: thin;
  scrollbar-color: var(--border) transparent;
  padding-bottom: var(--space-1);
}
.song-scroll::-webkit-scrollbar {
  height: 4px;
}
.song-scroll::-webkit-scrollbar-track {
  background: transparent;
}
.song-scroll::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 2px;
}

/* Confirm modal */
.confirm-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding-top: var(--space-3);
}
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 10px 16px;
  font-weight: 500;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: background var(--transition-base);
  font-family: inherit;
  font-size: var(--text-base);
  min-height: 44px;
}
.btn--primary { background: var(--accent); color: #fff; }
.btn--secondary { background: transparent; border: 1px solid var(--border); color: var(--text-primary); }
@media (hover: hover) {
  .btn--primary:hover:not(:disabled) { background: var(--accent-hover); }
  .btn--secondary:hover { border-color: var(--border-hover); background: var(--bg-tertiary); }
}

/* Responsive */
@media (max-width: 639px) {
  .banner-header {
    padding: var(--space-3) var(--space-4);
  }
  .banner-body {
    padding: var(--space-3) var(--space-4);
  }
}
</style>
