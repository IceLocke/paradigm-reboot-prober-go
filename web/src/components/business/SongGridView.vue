<template>
  <div class="grid-view">
    <div v-if="groups.length > 0" class="level-groups">
      <div
        v-for="group in groups"
        :key="group.key"
        :class="['level-row', { 'level-row--collapsed': collapsedLevels.has(group.key) }]"
      >
        <button class="level-label" @click="$emit('toggle-level', group.key)">
          <ChevronRight :size="12" :class="['level-chevron', { open: !collapsedLevels.has(group.key) }]" />
          <span class="level-value">{{ group.key }}</span>
          <span class="level-count">({{ t('term.chart_count', { count: group.charts.length }) }})</span>
        </button>
        <div v-if="!collapsedLevels.has(group.key)" class="level-cards">
          <SongGridCard
            v-for="chart in group.charts"
            :key="chart.id"
            :chart="chart"
            @click="$emit('click-chart', chart.song_id)"
            @add-to-cart="$emit('add-to-cart', chart)"
            @quick-upload="$emit('quick-upload', chart)"
          />
        </div>
      </div>
    </div>
    <div v-else class="empty-state">
      <n-empty :description="t('common.no_data')" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NEmpty } from 'naive-ui'
import { ChevronRight } from '@lucide/vue'
import type { ChartInfo } from '@/api/types'
import type { ChartGroup } from '@/composables/useChartGroups'
import SongGridCard from '@/components/business/SongGridCard.vue'

const { t } = useI18n()

defineProps<{
  groups: ChartGroup[]
  collapsedLevels: Set<string>
}>()

defineEmits<{
  'toggle-level': [key: string]
  'click-chart': [songId: number]
  'add-to-cart': [chart: ChartInfo]
  'quick-upload': [chart: ChartInfo]
}>()
</script>

<style scoped>
.grid-view {
  margin-top: 0;
}

.level-groups {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.level-row {
  display: flex;
  gap: var(--space-3);
  align-items: stretch;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: var(--space-3);
  transition: width var(--transition-base);
}

.level-row--collapsed {
  padding: var(--space-1) var(--space-3);
}

.level-label {
  display: flex;
  align-items: baseline;
  gap: var(--space-1);
  flex-shrink: 0;
  width: 140px;
  min-width: 140px;
  padding: var(--space-1) 0;
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-primary);
  font-family: inherit;
  transition: color var(--transition-fast);
  overflow: hidden;
}
@media (hover: hover) {
  .level-label:hover { color: var(--accent); }
}

.level-chevron {
  color: var(--text-muted);
  flex-shrink: 0;
  transition: transform var(--transition-fast);
}
.level-chevron.open {
  transform: rotate(90deg);
}

.level-value {
  font-family: var(--font-mono);
  font-size: var(--text-lg);
  font-weight: 700;
  white-space: nowrap;
}

.level-count {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 400;
  white-space: nowrap;
  flex-shrink: 0;
}

.level-cards {
  display: flex;
  gap: var(--space-3);
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: thin;
  scrollbar-color: var(--border) transparent;
  padding-bottom: var(--space-1);
  flex: 1;
  min-width: 0;
}
.level-cards::-webkit-scrollbar {
  height: 4px;
}
.level-cards::-webkit-scrollbar-track {
  background: transparent;
}
.level-cards::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 2px;
}

/* Cards in level rows should have fixed width */
.level-cards :deep(.song-grid-card) {
  flex-shrink: 0;
  width: 130px;
}

@media (max-width: 639px) {
  .level-row {
    flex-wrap: wrap;
    gap: var(--space-2);
    padding: var(--space-2);
  }
  .level-label {
    width: 100%;
    min-width: 0;
  }
  .level-value {
    font-size: var(--text-base);
  }
  .level-cards {
    width: 100%;
    gap: var(--space-2);
  }
  .level-cards :deep(.song-grid-card) {
    width: 105px;
  }
}

.empty-state {
  padding: var(--space-10) 0;
  display: flex;
  justify-content: center;
}
</style>
