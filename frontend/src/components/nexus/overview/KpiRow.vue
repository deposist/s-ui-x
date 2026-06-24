<template>
  <div class="nexus-overview-kpis">
    <kpi-card
      class="nexus-overview-kpis__traffic"
      :delta="$t('nexus.overview.kpi.liveTrafficDelta')"
      :label="$t('nexus.overview.kpi.liveTraffic')"
      :value="loading ? '-' : formatOverviewRate(summary.liveTrafficBps)"
    >
      <template #meta>
        <!-- Time window selector for sparkline -->
        <div class="nexus-overview-kpis__window-selector">
          <v-menu>
            <template #activator="{ props }">
              <v-btn
                variant="text"
                density="compact"
                size="small"
                class="nexus-overview-kpis__window-btn"
                v-bind="props"
              >
                {{ timeWindowLabel }}
                <v-icon icon="lucide:chevron-down" size="14" class="ms-1" />
              </v-btn>
            </template>
            <v-list density="compact" min-width="120">
              <v-list-item
                v-for="opt in timeWindowOptions"
                :key="opt.value"
                :active="opt.value === sparkWindowSize"
                @click="sparkWindowSize = opt.value"
              >
                <v-list-item-title class="text-caption">{{ opt.label }}</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-menu>
        </div>
      </template>

      <template #trend>
        <area-spark
          :aria-label="$t('nexus.overview.kpi.trafficTrend')"
          :labels="traffic.labels"
          :values="trafficTrend"
        />
      </template>
    </kpi-card>

    <kpi-card
      :delta="wsStateLabel"
      :label="$t('nexus.overview.kpi.onlineClients')"
      :value="formatOverviewCount(summary.onlineClients)"
      class="nexus-overview-kpis__online-clients"
    >
      <template #trend>
        <div class="nexus-overview-kpis__signal">
          {{ $t('nexus.overview.kpi.clientSignal') }}
        </div>
      </template>
    </kpi-card>

    <kpi-card
      :delta="$t('nexus.overview.kpi.activeInbounds', { count: formatOverviewCount(summary.activeInbounds) })"
      :label="$t('nexus.overview.kpi.enabledInbounds')"
      :value="formatOverviewCount(summary.totalInbounds)"
      class="nexus-overview-kpis__enabled-inbounds"
    >
      <template #trend>
        <div class="nexus-overview-kpis__signal">
          {{ $t('nexus.overview.kpi.inboundOnlineTags', { count: formatOverviewCount(summary.activeInbounds) }) }}
        </div>
      </template>
    </kpi-card>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import AreaSpark from '@/components/nexus/primitives/AreaSpark.vue'
import KpiCard from '@/components/nexus/primitives/KpiCard.vue'
import type { WsConnectionState } from '@/store/ws'
import {
  formatOverviewCount,
  formatOverviewRate,
} from './overviewFormatters'
import type { KpiSummary } from './selectors/kpiSelectors'
import type { SystemStatus } from './selectors/systemStatusSelectors'
import type { TrafficSeries } from './selectors/trafficSelectors'

const props = defineProps<{
  loading: boolean
  summary: KpiSummary
  status: SystemStatus
  traffic: TrafficSeries
  wsState: WsConnectionState
  sparkWindowSize: number
}>()

const emit = defineEmits<{
  'update:sparkWindowSize': [value: number]
}>()

const sparkWindowSize = computed({
  get: () => props.sparkWindowSize,
  set: (val) => emit('update:sparkWindowSize', val),
})

const { t } = useI18n()

const timeWindowOptions = [
  { value: 6, label: '1 min' },
  { value: 30, label: '5 min' },
  { value: 180, label: '30 min' },
  { value: 360, label: '60 min' },
  { value: 1800, label: '5 hours' },
  { value: 4320, label: '12 hours' },
  { value: 8640, label: '24 hours' },
]

const timeWindowLabel = computed(() => {
  const activeOpt = timeWindowOptions.find(opt => opt.value === sparkWindowSize.value)
  return activeOpt ? activeOpt.label : '24 hours'
})

const trafficTrend = computed(() => {
  return props.traffic.download.map((download, index) => {
    return download + (props.traffic.upload[index] ?? 0)
  })
})

const wsStateLabel = computed(() => {
  if (props.wsState === 'connected') return t('nexus.status.realtime')
  if (props.wsState === 'reconnecting') return t('nexus.status.reconnecting')
  return t('nexus.status.pollFallback')
})
</script>

<style scoped>
.nexus-overview-kpis {
  display: grid;
  gap: var(--nexus-gap-4);
  grid-template-columns: repeat(4, minmax(0, 1fr));
  min-width: 0;
}

.nexus-overview-kpis__traffic {
  grid-column: span 2;
}

.nexus-overview-kpis__online-clients :deep(.nexus-kpi-card__value),
.nexus-overview-kpis__enabled-inbounds :deep(.nexus-kpi-card__value),
.nexus-overview-kpis__traffic :deep(.nexus-kpi-card__value) {
  font-family: var(--nexus-font-mono);
}

.nexus-overview-kpis__window-selector {
  flex: 0 0 auto;
}

.nexus-overview-kpis__window-btn {
  text-transform: none;
  font-size: 0.72rem !important;
  color: var(--nexus-text-secondary);
  height: 24px !important;
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-sm);
  background: var(--nexus-surface-2);
}

.nexus-overview-kpis__window-btn:hover {
  color: var(--nexus-text-primary);
  border-color: var(--nexus-border-strong);
}

.nexus-overview-kpis__signal {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-size: 0.78rem;
  letter-spacing: 0;
  line-height: 1.35;
  min-width: 0;
  overflow-wrap: anywhere;
}

@media (max-width: 1264px) {
  .nexus-overview-kpis {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 600px) {
  .nexus-overview-kpis {
    grid-template-columns: minmax(0, 1fr);
  }

  .nexus-overview-kpis__traffic {
    grid-column: auto;
  }
}
</style>
