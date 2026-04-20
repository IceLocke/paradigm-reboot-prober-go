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

      <!-- Right: editor (single scroll container) -->
      <div class="admin-editor-wrap">
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
  // Group charts by song id first…
  const grouped = new Map<number, ChartInfo[]>()
  for (const c of charts) {
    const arr = grouped.get(c.song_id)
    if (arr) arr.push(c)
    else grouped.set(c.song_id, [c])
  }

  // …then pick a “canonical” chart per song to source title/artist/etc. from.
  // MASSIVE is preferred because `override_*` fields on other difficulties may
  // re-title or re-attribute the chart for that specific difficulty — MASSIVE
  // generally carries the “main” metadata. Fall back in descending order when
  // MASSIVE is absent.
  const PREFERRED: Difficulty[] = ['massive', 'invaded', 'detected', 'reboot']
  const pickCanonical = (group: ChartInfo[]): ChartInfo => {
    for (const d of PREFERRED) {
      const hit = group.find((c) => c.difficulty === d)
      if (hit) return hit
    }
    return group[0]
  }

  const out: SongListItem[] = []
  for (const [id, group] of grouped) {
    const canonical = pickCanonical(group)
    const diffs: Difficulty[] = []
    for (const c of group) {
      if (!diffs.includes(c.difficulty)) diffs.push(c.difficulty)
    }
    out.push({
      id,
      title: canonical.title,
      artist: canonical.artist,
      version: canonical.version,
      album: canonical.album,
      genre: canonical.genre,
      wiki_id: canonical.wiki_id,
      cover: canonical.cover,
      b15: canonical.b15,
      diff_count: group.length,
      diffs,
    })
  }
  return out.sort((a, b) => a.id - b.id)
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
  max-height: calc(100vh - var(--app-header-height));
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
  .admin-page {
    max-height: none;
  }
  .admin-layout {
    grid-template-columns: 1fr;
    flex: initial;
    min-height: 0;
  }
}

.admin-sidebar {
  min-height: 0;
  /* Grid items default to min-width: auto which lets long content push the
     column wider than its track. Reset to 0 so the 340px track is respected. */
  min-width: 0;
  display: flex;
  flex-direction: column;
}

/* Single scroll container for the editor (desktop) */
.admin-editor-wrap {
  min-height: 0;
  min-width: 0;
  overflow-y: auto;
  padding-right: 4px;
}
@media (max-width: 1023px) {
  .admin-editor-wrap {
    overflow-y: visible;
    padding-right: 0;
  }
}

.editor-empty {
  min-height: 280px;
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
