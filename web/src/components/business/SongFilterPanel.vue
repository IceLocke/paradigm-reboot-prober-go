<template>
  <div>
    <!-- Advanced Filters Toggle -->
    <button class="adv-filter-toggle" @click="show = !show">
      <ChevronRight :size="14" :class="['adv-filter-chevron', { open: show }]" />
      <span>{{ t('term.filters') }}</span>
    </button>

    <!-- Advanced Filters Panel -->
    <transition name="fade">
      <div v-if="show" class="adv-filters">
        <div class="adv-filters-grid">
          <!-- Level range -->
          <div class="filter-group">
            <label class="filter-label">{{ t('term.level_range') }}</label>
            <div class="level-range-inputs">
              <input
                v-model.number="levelMin"
                type="number"
                step="0.1"
                class="level-input"
                :placeholder="t('term.min_level')"
              />
              <span class="range-sep">~</span>
              <input
                v-model.number="levelMax"
                type="number"
                step="0.1"
                class="level-input"
                :placeholder="t('term.max_level')"
              />
            </div>
          </div>

          <!-- Version select -->
          <div class="filter-group">
            <label class="filter-label">{{ t('term.version') }}</label>
            <n-select
              v-model:value="versionSelect"
              :options="versionOptions"
              :placeholder="t('term.version_select')"
              clearable
              multiple
            />
          </div>

          <!-- Album select -->
          <div class="filter-group">
            <label class="filter-label">{{ t('term.album') }}</label>
            <n-select
              v-model:value="albumSelect"
              :options="albumOptions"
              :placeholder="t('term.album_select')"
              clearable
              multiple
            />
          </div>

          <!-- Group by -->
          <div class="filter-group">
            <label class="filter-label">{{ t('term.group_by') }}</label>
            <n-select
              v-model:value="groupBy"
              :options="groupByOptions"
            />
          </div>
        </div>

        <!-- B50 filter (full-width at bottom) -->
        <div class="adv-filters-bottom">
          <button
            :class="['b50-btn', { active: b50Filter }]"
            @click="$emit('toggle-b50')"
          >
            <Star :size="14" />
            {{ t('term.in_b50') }}
          </button>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NSelect } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { ChevronRight, Star } from '@lucide/vue'

const { t } = useI18n()

defineProps<{
  versionOptions: SelectOption[]
  albumOptions: SelectOption[]
  groupByOptions: SelectOption[]
  b50Filter: boolean
  b50Loading: boolean
}>()

defineEmits<{
  'toggle-b50': []
}>()

const show = defineModel<boolean>('show', { required: true })
const levelMin = defineModel<number | null>('levelMin', { required: true })
const levelMax = defineModel<number | null>('levelMax', { required: true })
const versionSelect = defineModel<string[] | null>('versionSelect', { required: true })
const albumSelect = defineModel<string[] | null>('albumSelect', { required: true })
const groupBy = defineModel<'level' | 'version' | 'album'>('groupBy', { required: true })
</script>

<style scoped>
/* Advanced filter toggle */
.adv-filter-toggle {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  padding: var(--space-1) 0;
  margin-bottom: var(--space-2);
  font-family: inherit;
  transition: color var(--transition-fast);
}
@media (hover: hover) {
  .adv-filter-toggle:hover { color: var(--text-primary); }
}
.adv-filter-chevron {
  transition: transform var(--transition-fast);
  flex-shrink: 0;
}
.adv-filter-chevron.open {
  transform: rotate(90deg);
}

/* Advanced filters panel */
.adv-filters {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: var(--space-4);
}

.adv-filters-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: var(--space-4);
  align-items: end;
}

.filter-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.filter-label {
  font-size: var(--text-xs);
  color: var(--text-muted);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.level-range-inputs {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.level-input {
  width: 80px;
  min-height: 34px;
  padding: 0 var(--space-2);
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 16px;
  font-family: var(--font-mono);
  outline: none;
  transition: border-color var(--transition-fast);
}
.level-input:focus {
  border-color: var(--accent);
}
.level-input::placeholder {
  color: var(--text-muted);
  font-family: var(--font-sans);
}

.range-sep {
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.adv-filters-bottom {
  margin-top: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--border);
}

.b50-btn {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: 7px 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  transition: background var(--transition-fast), color var(--transition-fast), border-color var(--transition-fast);
  font-family: inherit;
  white-space: nowrap;
  min-height: 34px;
}
.b50-btn.active {
  background: var(--accent-muted);
  border-color: var(--accent);
  color: var(--accent);
}
@media (hover: hover) {
  .b50-btn:not(.active):hover {
    border-color: var(--border-hover);
    color: var(--text-primary);
  }
}
</style>
