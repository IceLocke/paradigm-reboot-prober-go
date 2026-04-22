<template>
  <div class="song-grid-card" @click="$emit('click')">
    <div
      class="cover-frame"
      :style="{ borderColor: diffColors[chart.difficulty] }"
    >
      <img
        :src="coverUrl"
        :alt="chart.title"
        class="cover-img"
        loading="lazy"
      />
      <div class="cover-actions">
        <button
          class="cover-action-btn"
          :title="t('message.add_to_upload_list')"
          @click.stop="$emit('add-to-cart')"
        >
          <Plus :size="13" />
        </button>
        <button
          class="cover-action-btn"
          :title="t('message.quick_upload')"
          @click.stop="$emit('quick-upload')"
        >
          <Upload :size="13" />
        </button>
      </div>
    </div>
    <div class="card-info">
      <span class="card-title" :title="chart.title">{{ chart.title }}</span>
      <span :class="['card-version', { 'card-version--new': chart.b15 }]">v{{ chart.version }}{{ chart.b15 ? ' ✦' : '' }}</span>
      <span class="card-badge-row">
        <DifficultyBadge :difficulty="chart.difficulty" :level="chart.level" :short="true" />
        <span
          v-if="chart.fitting_level != null"
          class="card-fitting"
          :title="t('term.fitting_level') + ' ' + chart.fitting_level.toFixed(1)"
        >≈{{ chart.fitting_level.toFixed(1) }}</span>
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, Upload } from '@lucide/vue'
import type { ChartInfo, Difficulty } from '@/api/types'
import DifficultyBadge from '@/components/business/DifficultyBadge.vue'
import { coverThumbUrl } from '@/utils/cover'

const { t } = useI18n()

const props = defineProps<{
  chart: ChartInfo
}>()

defineEmits<{
  click: []
  'add-to-cart': []
  'quick-upload': []
}>()

const diffColors: Record<Difficulty, string> = {
  detected: '#3b82f6',
  invaded: '#ef4444',
  massive: '#a855f7',
  reboot: '#f97316',
}

const coverUrl = computed(() => {
  if (!props.chart.cover) return ''
  if (props.chart.cover.startsWith('http')) return props.chart.cover
  // Use thumbnail for grid view to save bandwidth
  return coverThumbUrl(props.chart.cover)
})
</script>

<style scoped>
.song-grid-card {
  display: flex;
  flex-direction: column;
  padding: var(--space-1);
  cursor: pointer;
}

.cover-frame {
  position: relative;
  width: 100%;
  aspect-ratio: 1;
  border: 2.5px solid;
  border-radius: 8px;
  overflow: hidden;
  flex-shrink: 0;
}

.cover-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

/* Action buttons overlay on cover */
.cover-actions {
  position: absolute;
  bottom: 4px;
  right: 4px;
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity var(--transition-fast);
}

/* Desktop: show on card hover */
@media (hover: hover) {
  .song-grid-card:hover .cover-actions {
    opacity: 1;
  }
}

/* Mobile: always visible */
@media (hover: none) {
  .cover-actions {
    opacity: 1;
  }
}

.cover-action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  background: rgba(0, 0, 0, 0.75);
  border: none;
  color: #e4e4e7;
  cursor: pointer;
  border-radius: 4px;
  transition: background var(--transition-fast);
}
@media (hover: hover) {
  .cover-action-btn:hover {
    background: rgba(0, 0, 0, 0.85);
  }
}

.card-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-top: var(--space-2);
  min-width: 0;
}

.card-title {
  font-size: var(--text-xs);
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.4;
}

.card-version {
  font-size: 11px;
  font-family: var(--font-mono);
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.3;
  margin-bottom: 2px;
}

.card-version--new {
  color: var(--accent);
}

.card-badge-row {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  flex-wrap: wrap;
  min-width: 0;
}

.card-fitting {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-muted);
  white-space: nowrap;
  cursor: help;
}
</style>
