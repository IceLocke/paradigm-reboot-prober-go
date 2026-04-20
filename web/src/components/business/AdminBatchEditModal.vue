<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('admin.batch_edit')"
    :style="modalStyle"
    :bordered="false"
    content-style="display: flex; flex-direction: column; overflow: hidden;"
    @update:show="$emit('update:show', $event)"
  >
    <p class="hint">{{ t('admin.batch_edit_hint') }}</p>

    <div class="sel-summary">
      <strong>{{ t('admin.selection', { count: songIds.length }) }}</strong>
      <div class="sel-chips">
        <span v-for="id in previewIds" :key="id" class="chip">
          <span class="mono">#{{ id }}</span>
          <span class="chip-title">{{ songsIndex.get(id)?.title ?? '' }}</span>
        </span>
        <span v-if="songIds.length > previewIds.length" class="chip chip--more">
          +{{ songIds.length - previewIds.length }}
        </span>
      </div>
    </div>

    <div class="fields">
      <div v-for="f in fields" :key="f.key" class="field-row">
        <label class="enable-cb">
          <input type="checkbox" :checked="f.enabled.value" @change="f.enabled.value = !f.enabled.value" />
          <span>{{ f.label }}</span>
        </label>
        <div class="field-input">
          <template v-if="f.type === 'text'">
            <input
              :value="(f.value.value as string)"
              class="plain-input"
              :disabled="!f.enabled.value"
              :placeholder="f.placeholder"
              @input="f.value.value = ($event.target as HTMLInputElement).value"
            />
          </template>
          <template v-else-if="f.type === 'boolean'">
            <n-radio-group
              :value="(f.value.value as boolean)"
              :disabled="!f.enabled.value"
              @update:value="(v: unknown) => (f.value.value = v as boolean)"
            >
              <span class="radio-row">
                <n-radio :value="false">{{ t('admin.b15_off') }}</n-radio>
                <n-radio :value="true">{{ t('admin.b15_on') }}</n-radio>
              </span>
            </n-radio-group>
          </template>
        </div>
      </div>
    </div>

    <div v-if="progress.running" class="progress">
      <span>{{ progress.done }} / {{ progress.total }}</span>
      <div class="bar">
        <div class="bar-inner" :style="{ width: `${(progress.done / Math.max(progress.total, 1)) * 100}%` }"></div>
      </div>
      <span v-if="progress.failed > 0" class="fail-text">
        {{ progress.failed }} failed
      </span>
    </div>

    <template #footer>
      <div class="footer">
        <BaseButton variant="secondary" @click="$emit('update:show', false)" :disabled="progress.running" :text="t('common.cancel')" />
        <BaseButton :disabled="songIds.length === 0 || !anyEnabled || progress.running" @click="onApply">
          {{ t('admin.apply') }}
        </BaseButton>
      </div>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NModal, NRadioGroup, NRadio } from 'naive-ui'
import type { Ref } from 'vue'

import BaseButton from '@/components/ui/BaseButton.vue'
import { getSingleSongInfo, updateSong } from '@/api/song'
import { USE_MOCK } from '@/api/mock'
import { toastSuccess, toastError } from '@/utils/toast'
import { useBreakpoint } from '@/composables/useBreakpoint'
import type { SongListItem } from '@/views/AdminSongsView.vue'
import type { Song, ChartInput, UpdateSongRequest } from '@/api/types'

const { t } = useI18n()
const { isMobile } = useBreakpoint()

const show = defineModel<boolean>('show', { required: true })

const props = defineProps<{
  songIds: number[]
  songsIndex: Map<number, SongListItem>
}>()

const emit = defineEmits<{
  applied: []
}>()

const modalStyle = computed(() =>
  (isMobile.value
    ? 'width: 95vw; max-width: 95vw;'
    : 'width: 560px; max-width: 90vw;') +
  'max-height: 90vh;'
)

const previewIds = computed(() => props.songIds.slice(0, 10))

// ---- Field definitions ----
// Only keys that map to “textual” / “boolean” fields on UpdateSongRequest can be
// batch-edited. We key the field by `UpdateSongRequest` so any typo is caught.
type TextKey = Extract<
  keyof UpdateSongRequest,
  'version' | 'album' | 'genre' | 'illustrator' | 'bpm' | 'length' | 'cover'
>
type BoolKey = Extract<keyof UpdateSongRequest, 'b15'>

interface TextField {
  key: TextKey
  label: string
  type: 'text'
  placeholder?: string
  enabled: Ref<boolean>
  value: Ref<string>
}
interface BoolField {
  key: BoolKey
  label: string
  type: 'boolean'
  enabled: Ref<boolean>
  value: Ref<boolean>
}
type FieldDef = TextField | BoolField

const text = (key: TextKey, label: string, placeholder = ''): TextField => ({
  key, label, type: 'text', placeholder,
  enabled: ref(false), value: ref(''),
})
const bool = (key: BoolKey, label: string): BoolField => ({
  key, label, type: 'boolean',
  enabled: ref(false), value: ref(false),
})

const fields: FieldDef[] = [
  text('version', t('term.version'), '1.0.0'),
  text('album', t('term.album')),
  text('genre', t('term.genre')),
  text('illustrator', t('term.illustrator')),
  text('bpm', 'BPM', '180'),
  text('length', t('term.length'), '2:30'),
  text('cover', t('term.cover'), 'Cover_xxx.jpg'),
  bool('b15', t('term.season')),
]

const anyEnabled = computed(() => fields.some((f) => f.enabled.value))

// ---- Progress ----
const progress = reactive({
  running: false,
  total: 0,
  done: 0,
  failed: 0,
})

// Build a strongly-typed patch containing just the enabled overrides.
type SongPatch = Partial<Pick<UpdateSongRequest, TextKey | BoolKey>>

const collectOverrides = (): SongPatch => {
  const patch: SongPatch = {}
  for (const f of fields) {
    if (!f.enabled.value) continue
    if (f.type === 'text') patch[f.key] = f.value.value
    else patch[f.key] = f.value.value
  }
  return patch
}

// Rebuild charts array from the existing song as typed ChartInput[] so
// we never need an `as unknown as` cast.
const chartsFromSong = (song: Song): ChartInput[] =>
  (song.charts ?? []).map<ChartInput>((c) => {
    const out: ChartInput = {
      difficulty: c.difficulty,
      level: c.level,
      notes: c.notes,
      level_design: c.level_design ?? '',
    }
    if (c.override_title) out.override_title = c.override_title
    if (c.override_artist) out.override_artist = c.override_artist
    if (c.override_version) out.override_version = c.override_version
    if (c.override_cover) out.override_cover = c.override_cover
    return out
  })

// ---- Apply ----
const onApply = async () => {
  if (props.songIds.length === 0 || !anyEnabled.value) return

  progress.running = true
  progress.total = props.songIds.length
  progress.done = 0
  progress.failed = 0

  const overrides = collectOverrides()

  for (const id of props.songIds) {
    try {
      // Fetch full song so charts can be included unchanged
      let song: Song | null = null
      if (!USE_MOCK) {
        const res = await getSingleSongInfo(id)
        song = res.data
      }
      if (!song) {
        // fallback: use listing info (won’t have charts — skip safely)
        const listItem = props.songsIndex.get(id)
        if (!listItem) throw new Error('song not found')
        progress.failed++
        progress.done++
        continue
      }

      const payload: UpdateSongRequest = {
        id: song.id,
        title: song.title,
        artist: song.artist,
        wiki_id: song.wiki_id,
        bpm: song.bpm,
        cover: song.cover,
        illustrator: song.illustrator,
        version: song.version,
        album: song.album,
        genre: song.genre,
        length: song.length,
        b15: song.b15,
        charts: chartsFromSong(song),
        ...overrides,
      }

      if (!USE_MOCK) {
        await updateSong(payload)
      }
    } catch {
      progress.failed++
    } finally {
      progress.done++
    }
  }

  progress.running = false
  if (progress.failed === 0) {
    toastSuccess(t('admin.batch_edit_applied', { count: props.songIds.length }))
  } else {
    toastError(t('admin.batch_edit_failed', { failed: progress.failed, count: props.songIds.length }))
  }
  emit('applied')
  show.value = false
}
</script>

<style scoped>
.hint {
  color: var(--text-muted);
  font-size: 13px;
  margin-bottom: var(--space-3);
}

.sel-summary {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: var(--space-3);
  margin-bottom: var(--space-4);
}
.sel-summary strong {
  display: block;
  color: var(--text-primary);
  font-size: 13px;
  margin-bottom: var(--space-2);
}
.sel-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--bg-tertiary);
  border-radius: 4px;
  padding: 2px 8px;
  font-size: 11px;
  color: var(--text-secondary);
  max-width: 220px;
}
.chip--more {
  color: var(--accent);
  background: var(--accent-muted);
}
.chip-title {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 160px;
}
.mono { font-family: var(--font-mono); }

.fields {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-height: 360px;
  overflow-y: auto;
  padding-right: 4px;
}

.field-row {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: var(--space-3);
  align-items: center;
  padding: var(--space-2) var(--space-3);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 6px;
}
@media (max-width: 639px) {
  .field-row {
    grid-template-columns: 1fr;
  }
}
.enable-cb {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: 13px;
  color: var(--text-primary);
  cursor: pointer;
}
.enable-cb input {
  width: 16px;
  height: 16px;
  accent-color: var(--accent);
}

.plain-input {
  width: 100%;
  padding: 6px 10px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 14px;
  outline: none;
  min-height: 34px;
  font-family: inherit;
}
.plain-input:focus { border-color: var(--accent); }
.plain-input:disabled { opacity: 0.5; cursor: not-allowed; }

.radio-row {
  display: flex;
  gap: var(--space-4);
}

.progress {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-top: var(--space-3);
  font-size: 12px;
  color: var(--text-secondary);
  font-family: var(--font-mono);
}
.bar {
  flex: 1;
  height: 6px;
  background: var(--bg-secondary);
  border-radius: 3px;
  overflow: hidden;
}
.bar-inner {
  height: 100%;
  background: var(--accent);
  transition: width var(--transition-base);
}
.fail-text {
  color: var(--color-danger);
}

.footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
