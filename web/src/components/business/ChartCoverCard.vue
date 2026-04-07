<template>
  <div class="chart-cover-card" @click="$emit('click')">
    <div
      class="cover-frame"
      :style="{ borderColor: diffColors[difficulty] }"
    >
      <img
        :src="coverUrl"
        :alt="title"
        class="cover-img"
        loading="lazy"
      />
    </div>
    <div class="card-meta">
      <span class="card-title" :title="title">{{ title }}</span>
      <DifficultyBadge :difficulty="difficulty" :level="level" :short="true" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Difficulty } from '@/api/types'
import DifficultyBadge from '@/components/business/DifficultyBadge.vue'

const props = defineProps<{
  cover: string
  title: string
  difficulty: Difficulty
  level: number
}>()

defineEmits<{ click: [] }>()

const diffColors: Record<Difficulty, string> = {
  detected: '#3b82f6',
  invaded: '#ef4444',
  massive: '#a855f7',
  reboot: '#f97316',
}

const coverUrl = computed(() => {
  if (!props.cover) return ''
  if (props.cover.startsWith('http')) return props.cover
  return `/cover/${props.cover}`
})
</script>

<style scoped>
.chart-cover-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
  flex-shrink: 0;
  width: 110px;
  cursor: pointer;
}

.cover-frame {
  width: 96px;
  height: 96px;
  border: 2.5px solid;
  border-radius: 8px;
  overflow: hidden;
  flex-shrink: 0;
  transition: box-shadow var(--transition-fast);
}
@media (hover: hover) {
  .cover-frame:hover {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  }
}

.cover-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.card-meta {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  width: 100%;
}

.card-title {
  font-size: var(--text-xs);
  color: var(--text-primary);
  text-align: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  width: 100%;
  line-height: 1.4;
}

@media (max-width: 639px) {
  .chart-cover-card {
    width: 92px;
  }
  .cover-frame {
    width: 80px;
    height: 80px;
  }
}
</style>
