<template>
  <article class="nexus-config-doctor">
    <panel-header :title="$t('doctor.title')">
      <template #action>
        <status-badge :label="statusLabel" :tone="statusTone" />
      </template>
    </panel-header>

    <div class="nexus-config-doctor__summary">
      <span>{{ report?.summary ?? $t('doctor.idle') }}</span>
      <v-btn
        color="primary"
        density="comfortable"
        prepend-icon="lucide:activity"
        :loading="loading"
        size="small"
        variant="tonal"
        @click="runDoctor"
      >
        {{ $t('doctor.run') }}
      </v-btn>
    </div>

    <v-skeleton-loader
      v-if="loading && !report"
      class="nexus-config-doctor__skeleton"
      type="list-item-three-line, list-item-three-line"
    />

    <div v-else-if="visibleItems.length" class="nexus-config-doctor__items">
      <section
        v-for="item in visibleItems"
        :key="item.id"
        class="nexus-config-doctor__item"
      >
        <status-badge :label="severityLabel(item.severity)" :tone="severityTone(item.severity)" />
        <div class="nexus-config-doctor__copy">
          <strong>{{ item.title }}</strong>
          <span>{{ item.message }}</span>
          <small v-if="item.action">{{ item.action }}</small>
        </div>
      </section>
    </div>

    <empty-state
      v-else
      compact
      icon="lucide:activity"
      :title="$t('doctor.noReport')"
    />
  </article>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import EmptyState from '@/components/nexus/primitives/EmptyState.vue'
import PanelHeader from '@/components/nexus/primitives/PanelHeader.vue'
import StatusBadge from '@/components/nexus/primitives/StatusBadge.vue'
import HttpUtils from '@/plugins/httputil'
import type { DoctorReport, DoctorSeverity } from '@/types/doctor'

const { t } = useI18n()
const loading = ref(false)
const report = ref<DoctorReport>()

const visibleItems = computed(() => report.value?.items?.slice(0, 7) ?? [])

const severityTone = (severity: DoctorSeverity) => {
  if (severity === 'error') return 'error'
  if (severity === 'warn') return 'warning'
  return 'success'
}

const severityLabel = (severity: DoctorSeverity) => {
  if (severity === 'error') return t('doctor.error')
  if (severity === 'warn') return t('doctor.warn')
  return t('doctor.ok')
}

const statusTone = computed(() => {
  if (!report.value) return 'neutral'
  return severityTone(report.value.status)
})

const statusLabel = computed(() => {
  if (!report.value) return t('doctor.notRun')
  return severityLabel(report.value.status)
})

const runDoctor = async () => {
  loading.value = true
  const msg = await HttpUtils.post('api/doctor/run', {})
  if (msg.success) {
    report.value = msg.obj as DoctorReport
  }
  loading.value = false
}
</script>

<style scoped>
.nexus-config-doctor {
  background: var(--nexus-surface-1);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-lg);
  display: grid;
  gap: var(--nexus-gap-3);
  min-width: 0;
  padding: var(--nexus-gap-4);
}

.nexus-config-doctor__summary {
  align-items: center;
  display: flex;
  gap: var(--nexus-gap-2);
  justify-content: space-between;
  min-width: 0;
}

.nexus-config-doctor__summary span {
  color: var(--nexus-text-secondary);
  font-size: 0.82rem;
  line-height: 1.4;
  min-width: 0;
  overflow-wrap: anywhere;
}

.nexus-config-doctor__items {
  display: grid;
  gap: var(--nexus-gap-2);
  min-width: 0;
}

.nexus-config-doctor__item {
  align-items: flex-start;
  background: var(--nexus-surface-2);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-md);
  display: grid;
  gap: var(--nexus-gap-2);
  grid-template-columns: auto minmax(0, 1fr);
  min-width: 0;
  padding: var(--nexus-gap-2);
}

.nexus-config-doctor__copy {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.nexus-config-doctor__copy strong,
.nexus-config-doctor__copy span,
.nexus-config-doctor__copy small {
  letter-spacing: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

.nexus-config-doctor__copy strong {
  color: var(--nexus-text-primary);
  font-size: 0.82rem;
  line-height: 1.3;
}

.nexus-config-doctor__copy span {
  color: var(--nexus-text-secondary);
  font-size: 0.78rem;
  line-height: 1.35;
}

.nexus-config-doctor__copy small {
  color: var(--nexus-status-warn);
  font-size: 0.74rem;
  line-height: 1.35;
}

.nexus-config-doctor__skeleton {
  background: transparent;
}

@media (max-width: 600px) {
  .nexus-config-doctor__summary {
    align-items: stretch;
    flex-direction: column;
  }

  .nexus-config-doctor__item {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
