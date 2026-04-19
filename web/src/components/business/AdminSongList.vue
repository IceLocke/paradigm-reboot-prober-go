<template>
  <div class="song-list-panel">
    <!-- Search -->
    <div class="search-box">
      <Search :size="14" />
      <input
        ref="searchInput"
        v-model="search"
        class="search-input"
        :placeholder="t('admin.search_placeholder')"
      />
      <button v-if="search" class="clear-btn" @click="search = ''">
        <X :size="14" />
      </button>
    </div>

    <!-- Filter: version + b15 -->
    <div class="list-filters">
      <n-select
        v-model:value="versionFilter"
        :options="versionOptions"
        :placeholder="t('admin.filter_version')"
        size="small"
        clearable
        class="filter-select"
      />
      <div class="b15-toggle">
        <button
          :class="['tog-btn', { active: b15Filter === null }]"
          @click="b15Filter = null"
        >{{ t('common.all') }}</button>
        <button
          :class="['tog-btn', { active: b15Filter === true }]"
          @click="b15Filter = true"
        >{{ t('admin.filter_b15') }}</button>
        <button
          :class="['tog-btn', { active: b15Filter === false }]"
          @click="b15Filter = false"
        >{{ t('admin.filter_b35') }}</button>
      </div>
    </div>

    <!-- Selection actions -->
    <div class="selection-bar">
      <span class="sel-count">
        {{ t('admin.selection', { count: selectedIds.length }) }}
      </span>
      <button class="link-btn" @click="$emit('select-all-filtered', filteredSongs.map(s => s.id))">
        {{ t('admin.select_all') }}
      </button>
      <button v-if="selectedIds.length > 0" class="link-btn" @click="$emit('clear-selection')">
        {{ t('admin.clear_selection') }}
      </button>
    </div>

    <!-- List -->
    <div class="song-list" :style="{ '--total': filteredSongs.length }">
      <div v-if="filteredSongs.length === 0" class="empty">
        <n-empty :description="t('common.no_data')" size="small" />
      </div>
      <button
        v-for="s in pagedSongs"
        :key="s.id"
        :class="['song-item', { 'is-selected': selectedId === s.id }]"
        @click="$emit('select', s.id)"
      >
        <span class="check-wrap" @click.stop="$emit('toggle-select', s.id)">
          <span :class="['checkbox', { checked: selectedIds.includes(s.id) }]">
            <Check v-if="selectedIds.includes(s.id)" :size="12" />
          </span>
        </span>
        <img
          v-if="s.cover"
          :src="coverUrl(s.cover)"
          :alt="s.title"
          class="song-cover"
          loading="lazy"
        />
        <div v-else class="song-cover song-cover--placeholder">?</div>
        <div class="song-meta">
          <div class="row-1">
            <span class="song-title" :title="s.title">{{ s.title }}</span>
            <span v-if="s.b15" class="b15-tag">B15</span>
          </div>
          <div class="row-2">
            <span class="song-artist" :title="s.artist">{{ s.artist }}</span>
          </div>
          <div class="row-3">
            <span class="id-tag">#{{ s.id }}</span>
            <span v-if="s.version" class="ver-tag">{{ s.version }}</span>
            <span class="diff-count">{{ s.diff_count }} {{ t('term.difficulty') }}</span>
          </div>
        </div>
      </button>
    </div>

    <!-- Load more / pagination -->
    <div v-if="filteredSongs.length > pageSize * pageIndex" class="load-more">
      <button class="more-btn" @click="pageIndex++">
        {{ filteredSongs.length - pageSize * pageIndex }} {{ t('common.all') }} +
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NSelect, NEmpty } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { Search, Check, X } from '@lucide/vue'
import type { SongListItem } from '@/views/AdminSongsView.vue'

const { t } = useI18n()

const props = defineProps<{
  songs: SongListItem[]
  selectedId: number | null
  selectedIds: number[]
}>()

defineEmits<{
  select: [id: number]
  'toggle-select': [id: number]
  'select-all-filtered': [ids: number[]]
  'clear-selection': []
}>()

const search = ref('')
const versionFilter = ref<string | null>(null)
const b15Filter = ref<boolean | null>(null)

const versionOptions = computed<SelectOption[]>(() => {
  const set = new Set<string>()
  for (const s of props.songs) if (s.version) set.add(s.version)
  return Array.from(set).sort().map((v) => ({ label: v, value: v }))
})

const filteredSongs = computed(() => {
  const q = search.value.trim().toLowerCase()
  return props.songs.filter((s) => {
    if (versionFilter.value && s.version !== versionFilter.value) return false
    if (b15Filter.value !== null && s.b15 !== b15Filter.value) return false
    if (!q) return true
    if (String(s.id) === q) return true
    return (
      s.title.toLowerCase().includes(q)
      || s.artist.toLowerCase().includes(q)
      || (s.wiki_id?.toLowerCase().includes(q) ?? false)
      || (s.album?.toLowerCase().includes(q) ?? false)
      || (s.genre?.toLowerCase().includes(q) ?? false)
    )
  })
})

// simple incremental render to keep DOM light
const pageSize = 80
const pageIndex = ref(1)
const pagedSongs = computed(() => filteredSongs.value.slice(0, pageSize * pageIndex.value))
watch(filteredSongs, () => { pageIndex.value = 1 })

const coverUrl = (cover: string) => {
  if (!cover) return ''
  if (cover.startsWith('http')) return cover
  return `/cover/${cover}`
}
</script>

<style scoped>
.song-list-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: var(--space-3);
  min-height: 0;
  height: 100%;
  max-height: calc(100vh - var(--app-header-height) - var(--space-6));
  overflow: hidden;
}

.search-box {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0 var(--space-3);
  min-height: 40px;
  transition: border-color var(--transition-base);
}
.search-box:focus-within { border-color: var(--accent); }
.search-input {
  border: none;
  background: none;
  color: var(--text-primary);
  font-size: 16px;
  outline: none;
  flex: 1;
  width: 100%;
  font-family: inherit;
}
.search-input::placeholder { color: var(--text-muted); }
.clear-btn {
  border: none;
  background: none;
  color: var(--text-muted);
  cursor: pointer;
  display: flex;
  padding: 4px;
  border-radius: 4px;
}
@media (hover: hover) { .clear-btn:hover { color: var(--text-primary); } }

.list-filters {
  display: flex;
  gap: var(--space-2);
  align-items: center;
  flex-wrap: wrap;
}
.filter-select { flex: 1; min-width: 140px; }

.b15-toggle {
  display: flex;
  background: var(--bg-secondary);
  border-radius: 6px;
  padding: 2px;
}
.tog-btn {
  padding: 4px 10px;
  background: transparent;
  border: none;
  color: var(--text-muted);
  font-size: 12px;
  font-family: inherit;
  cursor: pointer;
  border-radius: 4px;
  transition: color var(--transition-fast), background var(--transition-fast);
}
.tog-btn.active {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}
@media (hover: hover) {
  .tog-btn:not(.active):hover { color: var(--text-secondary); }
}

.selection-bar {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  font-size: 12px;
  color: var(--text-muted);
  padding: 0 var(--space-1);
}
.sel-count {
  font-family: var(--font-mono);
}
.link-btn {
  background: none;
  border: none;
  color: var(--accent);
  cursor: pointer;
  font-size: 12px;
  font-family: inherit;
  padding: 2px 0;
}
@media (hover: hover) {
  .link-btn:hover { opacity: 0.8; }
}

.song-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-right: 2px;
}
.empty {
  display: flex;
  justify-content: center;
  padding: var(--space-6);
}

.song-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2);
  background: transparent;
  border: 1px solid transparent;
  border-radius: 6px;
  cursor: pointer;
  text-align: left;
  font-family: inherit;
  color: var(--text-primary);
  transition: background var(--transition-fast), border-color var(--transition-fast);
  min-height: 56px;
}
@media (hover: hover) {
  .song-item:hover {
    background: rgba(255, 255, 255, 0.04);
  }
}
.song-item.is-selected {
  background: var(--accent-muted);
  border-color: var(--accent);
}

.check-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
}
.checkbox {
  width: 16px;
  height: 16px;
  border: 1.5px solid var(--border-hover);
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  transition: background var(--transition-fast), border-color var(--transition-fast);
}
.checkbox.checked {
  background: var(--accent);
  border-color: var(--accent);
}

.song-cover {
  width: 40px;
  height: 40px;
  object-fit: cover;
  border-radius: 4px;
  flex-shrink: 0;
  background: var(--bg-tertiary);
}
.song-cover--placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  font-size: 12px;
}

.song-meta {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.row-1 {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.song-title {
  font-weight: 500;
  font-size: 13px;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}
.b15-tag {
  font-family: var(--font-mono);
  font-size: 10px;
  padding: 1px 5px;
  background: rgba(249, 115, 22, 0.15);
  color: var(--diff-reboot);
  border-radius: 3px;
  flex-shrink: 0;
}
.row-2 {
  font-size: 11px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.row-3 {
  display: flex;
  gap: var(--space-2);
  font-size: 10px;
  color: var(--text-muted);
  font-family: var(--font-mono);
}
.id-tag, .ver-tag, .diff-count { white-space: nowrap; }

.load-more {
  display: flex;
  justify-content: center;
}
.more-btn {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  color: var(--text-secondary);
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
}
@media (hover: hover) {
  .more-btn:hover {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }
}
</style>
