<template>
  <IconButton
    :icon="ShoppingCart"
    :size="18"
    :title="t('term.upload_list')"
    @click="show = true"
  >
    <span v-if="appStore.uploadList.length > 0" class="badge">{{ appStore.uploadList.length }}</span>
  </IconButton>
  <UploadCartModal v-model:show="show" />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ShoppingCart } from '@lucide/vue';
import { useAppStore } from '@/stores/app'
import IconButton from '@/components/ui/IconButton.vue'
import UploadCartModal from '@/components/business/UploadCartModal.vue'

const { t } = useI18n()
const appStore = useAppStore()
const show = ref(false)
</script>

<style scoped>
.badge {
  position: absolute;
  top: 4px;
  right: 4px;
  background: var(--accent);
  color: #fff;
  font-size: 10px;
  font-weight: 600;
  min-width: 16px;
  height: 16px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 4px;
}
</style>
