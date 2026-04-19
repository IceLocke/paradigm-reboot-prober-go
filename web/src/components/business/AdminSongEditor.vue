<template>
  <div class="editor">
    <div v-if="loading" class="loading">
      <n-spin size="medium" />
    </div>
    <template v-else>
      <!-- Header -->
      <div class="editor-header">
        <div class="title-wrap">
          <h3>
            {{ mode === 'create' ? t('admin.create_song') : t('admin.edit_song') }}
            <span v-if="dirty" class="unsaved">· {{ t('admin.unsaved') }}</span>
          </h3>
          <span v-if="editingId != null" class="mono id-tag">#{{ editingId }}</span>
        </div>
        <div class="actions">
          <BaseButton variant="ghost" size="sm" @click="onReset" :text="t('admin.reset')" />
          <BaseButton variant="secondary" size="sm" @click="onDiscard">
            <X :size="14" />
            {{ t('common.close') }}
          </BaseButton>
          <BaseButton size="sm" :disabled="submitting || !dirty" @click="onSubmit">
            <Save :size="14" />
            {{ mode === 'create' ? t('admin.create_song') : t('admin.save_song') }}
          </BaseButton>
        </div>
      </div>

      <!-- General section -->
      <BaseCard>
        <template #header>
          <h4>{{ t('admin.general') }}</h4>
        </template>
        <div class="form-grid">
          <BaseInput v-model="form.title" :label="t('term.title') + ' *'" />
          <BaseInput v-model="form.artist" :label="t('term.artist') + ' *'" />
          <BaseInput v-model="form.wiki_id" :label="'wiki_id *'" />
          <BaseInput v-model="form.version" :label="t('term.version')" />
          <BaseInput v-model="form.album" :label="t('term.album')" />
          <BaseInput v-model="form.genre" :label="t('term.genre')" />
          <BaseInput v-model="form.bpm" label="BPM" />
          <BaseInput v-model="form.length" :label="t('term.length')" placeholder="2:30" />
          <BaseInput v-model="form.illustrator" :label="t('term.illustrator')" />
          <BaseInput v-model="form.cover" :label="t('term.cover')" placeholder="Cover_xxx.jpg" />
          <div class="form-field">
            <label class="field-label">{{ t('term.season') }}</label>
            <n-radio-group v-model:value="form.b15">
              <span class="radio-row">
                <n-radio :value="false">{{ t('admin.b15_off') }}</n-radio>
                <n-radio :value="true">{{ t('admin.b15_on') }}</n-radio>
              </span>
            </n-radio-group>
          </div>
          <div v-if="form.cover" class="cover-preview">
            <img :src="coverUrl(form.cover)" :alt="form.title" />
          </div>
        </div>
      </BaseCard>

      <!-- Charts section -->
      <BaseCard>
        <template #header>
          <div class="charts-header">
            <h4>{{ t('admin.charts') }}</h4>
            <BaseButton variant="secondary" size="sm" @click="addChart">
              <Plus :size="14" />
              {{ t('admin.add_chart') }}
            </BaseButton>
          </div>
        </template>
        <div v-if="form.charts.length === 0" class="no-charts">
          <n-empty :description="t('admin.require_chart')" size="small" />
        </div>
        <div v-else class="charts-list">
          <div v-for="(chart, i) in form.charts" :key="i" class="chart-card">
            <div class="chart-head">
              <DifficultyBadge :difficulty="chart.difficulty" :level="chart.level" />
              <button class="remove-btn" @click="removeChart(i)" :title="t('admin.remove_chart')">
                <Trash2 :size="14" />
              </button>
            </div>
            <div class="chart-grid">
              <div class="form-field">
                <label class="field-label">{{ t('term.difficulty') }} *</label>
                <n-select
                  :value="chart.difficulty"
                  :options="diffOptions"
                  size="small"
                  @update:value="(v: Difficulty) => (chart.difficulty = v)"
                />
              </div>
              <div class="form-field">
                <label class="field-label">{{ t('term.level') }} *</label>
                <n-input-number
                  v-model:value="chart.level"
                  :min="0"
                  :max="20"
                  :step="0.1"
                  :precision="1"
                  size="small"
                />
              </div>
              <div class="form-field">
                <label class="field-label">{{ t('term.notes') }} *</label>
                <n-input-number
                  v-model:value="chart.notes"
                  :min="0"
                  :step="1"
                  size="small"
                />
              </div>
              <div class="form-field chart-span-2">
                <label class="field-label">{{ t('term.level_design') }}</label>
                <input
                  v-model="chart.level_design"
                  class="plain-input"
                />
              </div>
            </div>
            <details class="override-section">
              <summary class="override-summary">
                <span>{{ t('admin.override') }}</span>
                <span v-if="hasOverride(chart)" class="override-dot"></span>
              </summary>
              <p class="override-hint">{{ t('admin.override_hint') }}</p>
              <div class="chart-grid">
                <div class="form-field">
                  <label class="field-label">{{ t('admin.override_title') }}</label>
                  <input
                    :value="chart.override_title ?? ''"
                    class="plain-input"
                    :placeholder="form.title"
                    @input="setOverride(i, 'override_title', ($event.target as HTMLInputElement).value)"
                  />
                </div>
                <div class="form-field">
                  <label class="field-label">{{ t('admin.override_artist') }}</label>
                  <input
                    :value="chart.override_artist ?? ''"
                    class="plain-input"
                    :placeholder="form.artist"
                    @input="setOverride(i, 'override_artist', ($event.target as HTMLInputElement).value)"
                  />
                </div>
                <div class="form-field">
                  <label class="field-label">{{ t('admin.override_version') }}</label>
                  <input
                    :value="chart.override_version ?? ''"
                    class="plain-input"
                    :placeholder="form.version"
                    @input="setOverride(i, 'override_version', ($event.target as HTMLInputElement).value)"
                  />
                </div>
                <div class="form-field">
                  <label class="field-label">{{ t('admin.override_cover') }}</label>
                  <input
                    :value="chart.override_cover ?? ''"
                    class="plain-input"
                    :placeholder="form.cover"
                    @input="setOverride(i, 'override_cover', ($event.target as HTMLInputElement).value)"
                  />
                </div>
              </div>
            </details>
          </div>
        </div>
      </BaseCard>
    </template>

    <ConfirmModal
      v-model:show="confirmDiscard"
      :message="t('admin.confirm_discard')"
      @confirm="doDiscard"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { NSpin, NSelect, NInputNumber, NRadioGroup, NRadio, NEmpty } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { Save, X, Plus, Trash2 } from '@lucide/vue'

import BaseCard from '@/components/ui/BaseCard.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import ConfirmModal from '@/components/ui/ConfirmModal.vue'
import DifficultyBadge from '@/components/business/DifficultyBadge.vue'

import { createSong, updateSong } from '@/api/song'
import { USE_MOCK } from '@/api/mock'
import { toastSuccess, toastError } from '@/utils/toast'
import { DIFFICULTY_ORDER, sortByDifficulty } from '@/utils/difficulty'
import type {
  Song, Chart, ChartInfo, Difficulty,
  CreateSongRequest, UpdateSongRequest, ChartInput,
} from '@/api/types'

export type EditorMode = 'edit' | 'create'

const { t } = useI18n()

const props = defineProps<{
  mode: EditorMode
  song: Song | null
  loading: boolean
}>()

const emit = defineEmits<{
  saved: [charts: ChartInfo[]]
  discard: []
}>()

// ---- form state ----
// Chart editor state mirrors `ChartInput` (the API request DTO) plus a local-only
// `id` for tracking existing rows. Override fields follow the DTO's optional shape.
type ChartForm = ChartInput & { id?: number }

// Song editor state mirrors `CreateSongRequest` (all form-editable fields) but
// keeps every text field as a required string for easier v-model binding; empty
// strings are treated as “unset” and filtered out (or kept as empty) when
// building the outbound payload.
type SongForm = {
  id?: number
  title: string
  artist: string
  wiki_id: string
  bpm: string
  cover: string
  illustrator: string
  version: string
  album: string
  genre: string
  length: string
  b15: boolean
  charts: ChartForm[]
}

const emptyChart = (difficulty: Difficulty = 'detected'): ChartForm => ({
  difficulty,
  level: 1,
  notes: 0,
  level_design: '',
})

const emptyForm = (): SongForm => ({
  title: '',
  artist: '',
  wiki_id: '',
  bpm: '',
  cover: '',
  illustrator: '',
  version: '',
  album: '',
  genre: '',
  length: '',
  b15: false,
  charts: [emptyChart()],
})

const form = reactive<SongForm>(emptyForm())
const original = ref<string>('')
const submitting = ref(false)
const confirmDiscard = ref(false)

const editingId = computed(() => form.id ?? null)

const fromSong = (song: Song): SongForm => ({
  id: song.id,
  title: song.title ?? '',
  artist: song.artist ?? '',
  wiki_id: song.wiki_id ?? '',
  bpm: song.bpm ?? '',
  cover: song.cover ?? '',
  illustrator: song.illustrator ?? '',
  version: song.version ?? '',
  album: song.album ?? '',
  genre: song.genre ?? '',
  length: song.length ?? '',
  b15: song.b15 ?? false,
  charts: (song.charts ?? []).map<ChartForm>((c: Chart) => ({
    id: c.id,
    difficulty: c.difficulty,
    level: c.level ?? 0,
    notes: c.notes ?? 0,
    level_design: c.level_design ?? '',
    override_title: c.override_title ?? undefined,
    override_artist: c.override_artist ?? undefined,
    override_version: c.override_version ?? undefined,
    override_cover: c.override_cover ?? undefined,
  })),
})

const resetFromSong = () => {
  if (props.mode === 'create') {
    Object.assign(form, emptyForm())
  } else if (props.song) {
    const next = fromSong(props.song)
    next.charts = sortByDifficulty(next.charts)
    Object.assign(form, next)
  }
  original.value = JSON.stringify(form)
}

watch(
  () => [props.song, props.mode] as const,
  () => resetFromSong(),
  { immediate: true },
)

const dirty = computed(() => JSON.stringify(form) !== original.value)

// ---- chart helpers ----
const diffOptions: SelectOption[] = [
  { label: 'DETECTED', value: 'detected' },
  { label: 'INVADED', value: 'invaded' },
  { label: 'MASSIVE', value: 'massive' },
  { label: 'REBOOT', value: 'reboot' },
]

const nextDifficulty = (): Difficulty => {
  const used = new Set(form.charts.map((c) => c.difficulty))
  for (const d of DIFFICULTY_ORDER) if (!used.has(d)) return d
  return 'detected'
}

const addChart = () => {
  form.charts.push(emptyChart(nextDifficulty()))
}

const removeChart = (i: number) => {
  form.charts.splice(i, 1)
}

type OverrideKey = 'override_title' | 'override_artist' | 'override_version' | 'override_cover'

const hasOverride = (c: ChartForm) =>
  !!(c.override_title || c.override_artist || c.override_version || c.override_cover)

const setOverride = (i: number, key: OverrideKey, v: string) => {
  form.charts[i][key] = v === '' ? undefined : v
}

// ---- validation ----
const validate = (): string | null => {
  if (!form.title.trim()) return t('admin.require_title')
  if (!form.artist.trim()) return t('admin.require_artist')
  if (!form.wiki_id.trim()) return t('admin.require_wiki_id')
  if (form.charts.length === 0) return t('admin.require_chart')
  const diffSeen = new Set<string>()
  for (const c of form.charts) {
    if (diffSeen.has(c.difficulty)) return t('admin.duplicate_difficulty')
    diffSeen.add(c.difficulty)
  }
  return null
}

// ---- build payload ----
const buildCharts = (): ChartInput[] =>
  form.charts.map<ChartInput>((c) => {
    const base: ChartInput = {
      difficulty: c.difficulty,
      level: Number(c.level) || 0,
      notes: Number(c.notes) || 0,
      level_design: c.level_design ?? '',
    }
    if (c.override_title) base.override_title = c.override_title
    if (c.override_artist) base.override_artist = c.override_artist
    if (c.override_version) base.override_version = c.override_version
    if (c.override_cover) base.override_cover = c.override_cover
    return base
  })

const buildCreatePayload = (): CreateSongRequest => ({
  title: form.title,
  artist: form.artist,
  wiki_id: form.wiki_id,
  bpm: form.bpm,
  cover: form.cover,
  illustrator: form.illustrator,
  version: form.version,
  album: form.album,
  genre: form.genre,
  length: form.length,
  b15: form.b15,
  charts: buildCharts(),
})

const buildUpdatePayload = (id: number): UpdateSongRequest => ({
  ...buildCreatePayload(),
  id,
})

// ---- submit ----
const onSubmit = async () => {
  const err = validate()
  if (err) {
    toastError(err)
    return
  }
  submitting.value = true
  try {
    let resultCharts: ChartInfo[] = []
    if (props.mode === 'create') {
      if (!USE_MOCK) {
        const res = await createSong(buildCreatePayload())
        resultCharts = res.data
      }
      toastSuccess('admin.create_success')
    } else {
      if (form.id == null) return
      if (!USE_MOCK) {
        const res = await updateSong(buildUpdatePayload(form.id))
        resultCharts = res.data
      }
      toastSuccess('admin.save_success')
    }
    emit('saved', resultCharts)
    // refresh baseline so dirty=false
    original.value = JSON.stringify(form)
  } catch (e) {
    toastError(props.mode === 'create' ? 'admin.create_failed' : 'admin.save_failed', e)
  } finally {
    submitting.value = false
  }
}

const onReset = () => { resetFromSong() }

const onDiscard = () => {
  if (dirty.value) confirmDiscard.value = true
  else doDiscard()
}
const doDiscard = () => {
  emit('discard')
}

// ---- keyboard shortcut: Cmd/Ctrl+S ----
const onKey = (e: KeyboardEvent) => {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 's') {
    e.preventDefault()
    onSubmit()
  }
}
window.addEventListener('keydown', onKey)
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))

// ---- utils ----
const coverUrl = (cover: string) => {
  if (!cover) return ''
  if (cover.startsWith('http')) return cover
  return `/cover/${cover}`
}
</script>

<style scoped>
.editor {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.loading {
  display: flex;
  justify-content: center;
  padding: var(--space-10);
}

.editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--space-3);
  flex-wrap: wrap;
  position: sticky;
  top: 0;
  z-index: 2;
  background: var(--bg-primary);
  padding: var(--space-2) 0;
  margin: calc(var(--space-2) * -1) 0 0 0;
}
.editor-header h3 {
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--text-primary);
}
.title-wrap {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  flex-wrap: wrap;
}
.unsaved {
  color: var(--diff-reboot);
  font-size: var(--text-sm);
  font-weight: 500;
}
.id-tag {
  color: var(--text-muted);
  font-size: 12px;
}
.mono { font-family: var(--font-mono); }

.actions {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--space-3);
  align-items: start;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.field-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
}
.radio-row {
  display: flex;
  gap: var(--space-4);
  padding-top: 8px;
}

.cover-preview img {
  width: 96px;
  height: 96px;
  border-radius: 8px;
  object-fit: cover;
  background: var(--bg-tertiary);
}

.charts-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.charts-header h4 {
  font-size: var(--text-base);
  font-weight: 600;
  color: var(--text-primary);
}
.no-charts {
  display: flex;
  justify-content: center;
  padding: var(--space-6);
}

.charts-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.chart-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: var(--space-3);
}
.chart-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-3);
}
.remove-btn {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 6px;
  border-radius: 6px;
  display: flex;
  transition: color var(--transition-fast), background var(--transition-fast);
}
@media (hover: hover) {
  .remove-btn:hover {
    color: var(--color-danger);
    background: rgba(239, 68, 68, 0.1);
  }
}

.chart-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: var(--space-3);
  align-items: end;
}
.chart-span-2 {
  grid-column: span 2;
}

.plain-input {
  padding: 6px 10px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 14px;
  outline: none;
  transition: border-color var(--transition-base);
  min-height: 34px;
  font-family: inherit;
}
.plain-input:focus { border-color: var(--accent); }

.override-section {
  margin-top: var(--space-3);
  border-top: 1px dashed var(--border);
  padding-top: var(--space-3);
}
.override-summary {
  cursor: pointer;
  font-size: 13px;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-weight: 500;
  list-style: none;
}
.override-summary::-webkit-details-marker { display: none; }
.override-summary::before {
  content: '›';
  font-size: 16px;
  transition: transform var(--transition-fast);
}
.override-section[open] .override-summary::before {
  transform: rotate(90deg);
}
.override-dot {
  width: 6px;
  height: 6px;
  background: var(--accent);
  border-radius: 50%;
}
.override-hint {
  font-size: 11px;
  color: var(--text-muted);
  margin: var(--space-2) 0 var(--space-3);
}
</style>
