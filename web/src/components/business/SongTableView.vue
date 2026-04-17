<template>
  <div>
    <div class="table-wrapper">
      <n-data-table
        :columns="columns"
        :data="data"
        :bordered="false"
        :scroll-x="650"
        size="small"
        striped
        @update:sorter="(s: DataTableSortState | DataTableSortState[] | null) => $emit('update:sorter', s)"
      />
    </div>

    <!-- Pagination -->
    <div class="pagination-row">
      <n-pagination
        v-model:page="pageIndex"
        :page-size="pageSize"
        :item-count="filteredCount"
        :page-slot="5"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NDataTable, NPagination } from 'naive-ui'
import type { DataTableColumns, DataTableSortState } from 'naive-ui'
import { Plus, Upload } from '@lucide/vue'
import type { ChartInfo } from '@/api/types'
import DifficultyBadge from '@/components/business/DifficultyBadge.vue'

const { t } = useI18n()

defineProps<{
  data: ChartInfo[]
  filteredCount: number
  pageSize: number
}>()

const emit = defineEmits<{
  'click-title': [songId: number]
  'add-to-cart': [chart: ChartInfo]
  'quick-upload': [chart: ChartInfo]
  'update:sorter': [sorter: DataTableSortState | DataTableSortState[] | null]
}>()

const pageIndex = defineModel<number>('pageIndex', { required: true })

const columns = computed<DataTableColumns<ChartInfo>>(() => [
  {
    title: t('term.title'),
    key: 'title',
    minWidth: 150,
    ellipsis: {
      tooltip: {
        zIndex: 99,
        'width': 'trigger',
      },
    },
    sorter: true,
    fixed: 'left',
    render(row) {
      return h('a', {
        class: 'link-text',
        onClick: () => emit('click-title', row.song_id),
      }, row.title)
    },
  },
  {
    title: t('term.version'),
    key: 'version',
    width: 90,
    sorter: true,
  },
  {
    title: t('term.season'),
    key: 'b15',
    width: 90,
    render(row) {
      return h('span', {
        class: row.b15 ? 'version-badge version-badge--new' : 'version-badge version-badge--old',
      }, row.b15 ? t('term.current') : t('term.past'))
    },
  },
  {
    title: t('term.difficulty'),
    key: 'difficulty',
    width: 90,
    render(row) {
      return h(DifficultyBadge, { difficulty: row.difficulty, short: false })
    },
  },
  {
    title: t('term.level'),
    key: 'level',
    width: 80,
    sorter: true,
    render(row) {
      return h('span', { class: 'mono' }, row.level.toFixed(1))
    },
  },
  {
    title: t('term.fitting_level'),
    key: 'fitting_level',
    width: 90,
    sorter: true,
    render(row) {
      return h('span', { class: 'mono' }, row.fitting_level != null ? row.fitting_level.toFixed(1) : '-')
    },
  },
  {
    title: '',
    key: 'actions',
    width: 80,
    render(row) {
      return h('div', { class: 'action-btns' }, [
        h('button', {
          class: 'action-btn',
          title: t('message.add_to_upload_list'),
          onClick: () => emit('add-to-cart', row),
        }, [
          h(Plus, { size: 14 }),
        ]),
        h('button', {
          class: 'action-btn',
          title: t('message.quick_upload'),
          onClick: () => emit('quick-upload', row),
        }, [
          h(Upload, { size: 14 }),
        ]),
      ])
    },
  },
])
</script>

<style scoped>
/* Table view */
.table-wrapper {
  -webkit-overflow-scrolling: touch;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
  padding: var(--space-3);
}

/* Pagination */
.pagination-row {
  display: flex;
  justify-content: center;
  margin-top: var(--space-5);
}

/* Table styles */
:deep(.link-text) {
  color: var(--accent);
  cursor: pointer;
  text-decoration: none;
  font-size: var(--text-sm);
}
:deep(.link-text:hover) { text-decoration: underline; }
:deep(.mono) { font-family: var(--font-mono); font-size: var(--text-sm); }

:deep(.version-badge) {
  display: inline-flex;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}
:deep(.version-badge--new) { background: var(--accent-muted); color: var(--accent); }
:deep(.version-badge--old) { background: rgba(161,161,170,0.1); color: var(--text-secondary); }

:deep(.action-btns) {
  display: flex;
  gap: 2px;
}
:deep(.action-btn) {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 6px;
  transition: background var(--transition-fast), color var(--transition-fast);
}
@media (hover: hover) {
  :deep(.action-btn:hover) { background: rgba(255,255,255,0.06); color: var(--text-primary); }
}
</style>
