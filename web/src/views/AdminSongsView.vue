<template>
  <div class="page-container admin-page">
    <div class="page-header">
      <h2>{{ t('admin.songs_panel') }}</h2>
      <div class="page-actions">
        <BaseButton variant="secondary" size="sm" @click="onNewSong">
          <Plus :size="14" />
          {{ t('admin.new_song') }}
        </BaseButton>
        <BaseButton
          variant="secondary"
          size="sm"
          :disabled="selectedIds.length === 0"
          @click="showBatchEdit = true"
        >
          <Layers :size="14" />
          {{ t('admin.batch_edit') }}
          <span v-if="selectedIds.length > 0" class="sel-badge">{{ selectedIds.length }}</span>
        </BaseButton>
        <IconButton :icon="RefreshCw" :size="18" :title="t('common.refresh')" @click="loadCharts" />
      </div>
    </div>

    <div class="admin-layout">
      <!-- Left: song list -->
      <div class="admin-sidebar">
        <AdminSongList
          :songs="songList"
          :selected-id="editingSongId"
          :selected-ids="selectedIds"
          @select="onSelectSong"
          @toggle-select="onToggleSelect"
          @select-all-filtered="onSelectAllFiltered"
          @clear-selection="selectedIds = []"
        />
      </div>

      <!-- Right: editor -->
      <div class="admin-editor">
        <div v-if="!editorMode" class="editor-empty">
          <Music2 :size="48" :stroke-width="1" />
          <p>{{ t('admin.no_song_selected') }}</p>
        </div>
        <AdminSongEditor
          v-else
          :key="editorKey"
          :mode="editorMode"
          :song="editingSong"
          :loading="editorLoading"
          @saved="onSaved"
          @discard="onDiscardEditor"
        />
      </div>
    </div>

    <AdminBatchEditModal
      v-model:show="showBatchEdit"
      :song-ids="selectedIds"
      :songs-index="songsById"
      @applied="onBatchApplied"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, Layers, Music2, RefreshCw } from '@lucide/vue'

import { useAppStore } from '@/stores/app'
import { getAllCharts, getSingleSongInfo } from '@/api/song'
import { USE_MOCK, getMockCharts } from '@/api/mock'
import type { ChartInfo, Song, Chart, Difficulty } from '@/api/types'

import BaseButton from '@/components/ui/BaseButton.vue'
import IconButton from '@/components/ui/IconButton.vue'
import AdminSongList from '@/components/business/AdminSongList.vue'
import AdminSongEditor, { type EditorMode } from '@/components/business/AdminSongEditor.vue'
import AdminBatchEditModal from '@/components/business/AdminBatchEditModal.vue'

import { toastSuccess, toastError } from '@/utils/toast'

const { t } = useI18n()
const appStore = useAppStore()

// -------- Song list derived from appStore.charts --------
export interface SongListItem {
  id: number
  title: string
  artist: string
  version: string
  album: string
  genre: string
  wiki_id: string
  cover: string
  b15: boolean
  diff_count: number
  diffs: Difficulty[]
}

const songList = computed<SongListItem[]>(() => {
  const charts = appStore.charts ?? []
  const byId = new Map<number, SongListItem>()
  for (const c of charts) {
    const existing = byId.get(c.song_id)
    if (existing) {
      existing.diff_count += 1
      if (!existing.diffs.includes(c.difficulty)) existing.diffs.push(c.difficulty)
    } else {
      byId.set(c.song_id, {
        id: c.song_id,
        title: c.title,
        artist: c.artist,
        version: c.version,
        album: c.album,
        genre: c.genre,
        wiki_id: c.wiki_id,
        cover: c.cover,
        b15: c.b15,
        diff_count: 1,
        diffs: [c.difficulty],
      })
    }
  }
  return Array.from(byId.values()).sort((a, b) => a.id - b.id)
})

const songsById = computed(() => {
  const m = new Map<number, SongListItem>()
  for (const s of songList.value) m.set(s.id, s)
  return m
})

// -------- Selection --------
const selectedIds = ref<number[]>([])
const onToggleSelect = (id: number) => {
  const idx = selectedIds.value.indexOf(id)
  if (idx >= 0) selectedIds.value.splice(idx, 1)
  else selectedIds.value.push(id)
}
const onSelectAllFiltered = (ids: number[]) => {
  // Merge
  const set = new Set(selectedIds.value)
  for (const id of ids) set.add(id)
  selectedIds.value = Array.from(set)
}

// -------- Editor --------
type LocalEditorMode = EditorMode | null
const editorMode = ref<LocalEditorMode>(null)
const editingSongId = ref<number | null>(null)
const editingSong = ref<Song | null>(null)
const editorLoading = ref(false)
const editorKey = ref(0)
const showBatchEdit = ref(false)

const onSelectSong = async (id: number) => {
  editorMode.value = 'edit'
  editingSongId.value = id
  editingSong.value = null
  editorLoading.value = true
  editorKey.value++
  try {
    if (USE_MOCK) {
      editingSong.value = buildMockSong(id)
    } else {
      const res = await getSingleSongInfo(id)
      editingSong.value = res.data
    }
  } catch (err) {
    toastError('message.get_song_failed', err)
    editorMode.value = null
  } finally {
    editorLoading.value = false
  }
}

const onNewSong = () => {
  editorMode.value = 'create'
  editingSongId.value = null
  editingSong.value = null
  editorKey.value++
}

const onDiscardEditor = () => {
  editorMode.value = null
  editingSongId.value = null
  editingSong.value = null
}

const onSaved = async (updatedCharts: ChartInfo[]) => {
  // Update appStore.charts with the returned charts
  if (appStore.charts && updatedCharts && updatedCharts.length > 0) {
    const affectedSongIds = new Set(updatedCharts.map((c) => c.song_id))
    appStore.charts = [
      ...appStore.charts.filter((c) => !affectedSongIds.has(c.song_id)),
      ...updatedCharts,
    ]
  } else {
    await loadCharts(true)
  }
  // If a new song was created, select it in the editor
  if (editorMode.value === 'create' && updatedCharts?.length) {
    const newId = updatedCharts[0].song_id
    onSelectSong(newId)
  }
}

const onBatchApplied = async () => {
  selectedIds.value = []
  await loadCharts(true)
  // Refresh current edit view if needed
  if (editingSongId.value != null) {
    onSelectSong(editingSongId.value)
  }
}

// -------- Load charts --------
const loadCharts = async (silent = false) => {
  if (USE_MOCK) {
    appStore.charts = getMockCharts()
    return
  }
  try {
    const res = await getAllCharts()
    appStore.charts = res.data
    if (!silent) toastSuccess('message.get_charts_success')
  } catch (err) {
    toastError('message.get_charts_failed', err)
  }
}

onMounted(() => {
  if (!appStore.charts) loadCharts(true)
})

// -------- Mock song builder --------
function buildMockSong(id: number): Song {
  const charts = (appStore.charts ?? []).filter((c) => c.song_id === id)
  const first = charts[0]
  return {
    id,
    title: first?.title ?? '',
    artist: first?.artist ?? '',
    bpm: first?.bpm ?? '',
    cover: first?.cover ?? '',
    illustrator: first?.illustrator ?? '',
    version: first?.version ?? '',
    album: first?.album ?? '',
    genre: first?.genre ?? '',
    length: first?.length ?? '',
    b15: first?.b15 ?? false,
    wiki_id: first?.wiki_id ?? '',
    created_at: '',
    updated_at: '',
    charts: charts.map<Chart>((c) => ({
      id: c.id,
      song_id: c.song_id,
      difficulty: c.difficulty,
      level: c.level,
      fitting_level: c.fitting_level,
      level_design: c.level_design,
      notes: c.notes,
      created_at: '',
      updated_at: '',
    })),
  }
}
</script>

<style scoped>
.admin-page {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
}

.page-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.sel-badge {
  background: var(--accent-muted);
  color: var(--accent);
  border-radius: 999px;
  padding: 1px 8px;
  font-size: 11px;
  font-weight: 600;
  font-family: var(--font-mono);
  margin-left: 2px;
}

.admin-layout {
  display: grid;
  grid-template-columns: 340px 1fr;
  gap: var(--space-4);
  flex: 1;
  min-height: 0;
  margin-top: var(--space-3);
}

@media (max-width: 1023px) {
  .admin-layout {
    grid-template-columns: 1fr;
  }
}

.admin-sidebar {
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.admin-editor {
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.editor-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
  color: var(--text-muted);
  background: var(--bg-card);
  border: 1px dashed var(--border);
  border-radius: 10px;
  padding: var(--space-10);
  text-align: center;
}
</style>
