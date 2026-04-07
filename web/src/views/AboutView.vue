<template>
  <div class="page-container">
    <div class="about-content">
      <div class="about-hero">
        <h1 class="about-title">{{ t('about.title') }}</h1>
        <p class="about-subtitle">{{ t('about.subtitle') }}</p>
      </div>

      <BaseCard>
        <div class="about-text">
          <p>{{ t('about.content_1') }}</p>
          <p style="margin-top: var(--space-4)">{{ t('about.content_2') }}</p>
        </div>
      </BaseCard>

      <div class="about-links">
        <a
          href="https://github.com/IceLocke/paradigm-reboot-prober-go"
          target="_blank"
          rel="noopener noreferrer"
          class="github-link"
        >
          <!-- GitHub Icon -->
          <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
          </svg>
          {{ t('about.github') }}
        </a>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseCard from '@/components/ui/BaseCard.vue'

const { t } = useI18n()
</script>

<style scoped>
.about-content {
  max-width: 640px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  padding-top: var(--space-8);
}
.about-hero {
  text-align: center;
}
.about-title {
  font-size: var(--text-3xl);
  font-weight: 700;
  margin-bottom: var(--space-2);
}
.about-subtitle {
  font-size: var(--text-lg);
  color: var(--text-muted);
}
.about-text p {
  font-size: var(--text-base);
  color: var(--text-secondary);
  line-height: 1.8;
}
.about-links {
  display: flex;
  justify-content: center;
}
.github-link {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: 10px 20px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text-primary);
  font-weight: 500;
  text-decoration: none;
  transition: border-color var(--transition-base);
}
@media (hover: hover) {
  .github-link:hover {
    border-color: var(--border-hover);
    color: var(--text-primary);
  }
}
</style>
