<template>
  <n-modal
    :show="show"
    preset="card"
    :title="song?.title ?? ''"
    :style="modalStyle"
    content-style="display: flex; flex-direction: column; overflow: hidden;"
    :bordered="false"
    @update:show="$emit('update:show', $event)"
  >
    <div v-if="song" class="song-detail">
      <div class="detail-grid">
        <div class="cover-section">
          <img
            v-if="song.cover"
            :src="coverUrl"
            :alt="song.title"
            class="cover-img"
            loading="lazy"
          />
          <div v-else class="cover-placeholder">No Cover</div>
        </div>
        <div class="info-section">
          <div class="info-grid">
            <div class="info-item">
              <span class="info-label">{{ t('term.artist') }}</span>
              <span class="info-value">{{ song.artist }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">BPM</span>
              <span class="info-value mono">{{ song.bpm }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">{{ t('term.illustrator') }}</span>
              <span class="info-value">{{ song.illustrator }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">{{ t('term.version') }}</span>
              <span class="info-value">{{ song.version }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">{{ t('term.album') }}</span>
              <span class="info-value">{{ song.album || '-' }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">{{ t('term.genre') }}</span>
              <span class="info-value">{{ song.genre || '-' }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">{{ t('term.length') }}</span>
              <span class="info-value mono">{{ song.length || '-' }}</span>
            </div>
          </div>

          <div class="charts-section">
            <span class="info-label">{{ t('term.difficulty') }}</span>
            <div class="charts-list">
              <div v-for="chart in sortedCharts" :key="chart.id" class="chart-row">
                <DifficultyBadge :difficulty="chart.difficulty" :level="chart.level" />
                <span class="chart-designer">{{ chart.level_design }}</span>
                <span v-if="chart.notes" class="chart-notes mono">{{ chart.notes }} {{ t('term.notes') }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div v-else class="loading-state">
      <n-spin size="medium" />
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NSpin, NModal } from 'naive-ui'
import type { Song } from '@/api/types'
import DifficultyBadge from './DifficultyBadge.vue'
import { useBreakpoint } from '@/composables/useBreakpoint'
import { sortByDifficulty } from '@/utils/difficulty'

const { t } = useI18n()
const { isMobile } = useBreakpoint()

const show = defineModel<boolean>('show', { required: true })
const props = defineProps<{
  song: Song | null
}>()

const sortedCharts = computed(() => sortByDifficulty(props.song?.charts ?? []))

const modalStyle = computed(() =>
  (isMobile.value
    ? 'width: 95vw; max-width: 90vw;'
    : 'width: 700px; max-width: 90vw;') +
  'max-height: 90vh;'
)

const coverUrl = computed(() => {
  if (!props.song?.cover) return ''
  if (props.song.cover.startsWith('http')) return props.song.cover
  return `/cover/${props.song.cover}`
})
</script>

<style scoped>
.song-detail {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  color: var(--text-primary);
}
.detail-grid {
  display: flex;
  gap: var(--space-6);
}
@media (max-width: 639px) {
  .detail-grid { flex-direction: column; gap: var(--space-4); }
}
.cover-section {
  flex-shrink: 0;
  width: 180px;
}
@media (max-width: 639px) {
  .cover-section { width: 120px; margin: 0 auto; }
}
.cover-img {
  width: 100%;
  aspect-ratio: 1;
  object-fit: cover;
  border-radius: 8px;
  background: var(--bg-tertiary);
}
.cover-placeholder {
  width: 100%;
  aspect-ratio: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-tertiary);
  border-radius: 8px;
  color: var(--text-muted);
  font-size: var(--text-sm);
}
.info-section { flex: 1; min-width: 0; }
.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3);
  margin-bottom: var(--space-5);
}
@media (max-width: 479px) {
  .info-grid { grid-template-columns: 1fr; }
}
.info-item { display: flex; flex-direction: column; gap: 2px; }
.info-label {
  font-size: var(--text-xs);
  color: var(--text-muted);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
.info-value {
  font-size: var(--text-base);
  color: var(--text-primary);
}
.charts-section { margin-top: var(--space-3); }
.charts-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
.chart-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  background: var(--bg-secondary);
  border-radius: 6px;
}
.chart-designer {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  flex: 1;
}
.chart-notes {
  font-size: var(--text-xs);
  color: var(--text-muted);
}
.loading-state {
  display: flex;
  justify-content: center;
  padding: var(--space-10);
}
.mono { font-family: var(--font-mono); }
</style>
