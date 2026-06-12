<template>
  <section class="preset-gallery">
    <div class="preset-gallery__header">
      <div>
        <h2>{{ $t('presets.title') }}</h2>
        <p>{{ $t('presets.subtitle') }}</p>
      </div>
      <v-chip label size="small" variant="tonal">{{ selectedPreset.region }}</v-chip>
    </div>

    <v-row>
      <v-col cols="12" md="4">
        <v-select
          v-model="presetId"
          data-testid="preset-id"
          density="compact"
          hide-details
          :items="presetItems"
          :label="$t('presets.preset')"
          variant="outlined"
        />
      </v-col>
      <v-col cols="12" md="4">
        <v-select
          v-model="proxyOutbound"
          data-testid="preset-proxy-outbound"
          density="compact"
          hide-details
          :items="outboundItems"
          :label="$t('presets.proxyOutbound')"
          variant="outlined"
        />
      </v-col>
      <v-col cols="12" md="4">
        <v-select
          v-model="directOutbound"
          data-testid="preset-direct-outbound"
          density="compact"
          hide-details
          :items="outboundItems"
          :label="$t('presets.directOutbound')"
          variant="outlined"
        />
      </v-col>
    </v-row>

    <p class="preset-gallery__description">{{ $t(selectedPreset.descriptionKey) }}</p>

    <div class="preset-gallery__sources">
      <span>{{ $t('presets.sources') }}</span>
      <a
        v-for="source in selectedPreset.sources"
        :key="source.url"
        :href="source.url"
        rel="noreferrer"
        target="_blank"
      >
        {{ source.name }}
      </a>
    </div>

    <v-alert
      v-if="!canApply"
      density="compact"
      type="warning"
      variant="tonal"
    >
      {{ $t('presets.selectOutbounds') }}
    </v-alert>

    <v-alert
      v-if="canApply && sameOutbound"
      density="compact"
      type="warning"
      variant="tonal"
    >
      {{ $t('presets.sameOutboundWarning') }}
    </v-alert>

    <div v-if="canApply" class="preset-gallery__preview">
      <strong>{{ $t('presets.preview') }}</strong>
      <ul>
        <li v-for="change in preview.changes" :key="change">{{ change }}</li>
      </ul>
    </div>

    <div class="preset-gallery__actions">
      <v-btn
        color="primary"
        :disabled="!canApply"
        prepend-icon="lucide:check"
        variant="tonal"
        @click="emit('apply', preview.config)"
      >
        {{ $t('presets.apply') }}
      </v-btn>
    </div>
  </section>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import type { Config } from '@/types/config'
import {
  applyRoutingDnsPreset,
  routingDnsPresetCatalog,
  type ApplyPresetResult,
} from './routingDnsPresets'

const props = defineProps<{
  config: Config
  outboundTags: string[]
}>()

const emit = defineEmits<{
  apply: [config: Config]
}>()

const { t } = useI18n()
const presetId = ref(routingDnsPresetCatalog[0]?.id ?? '')
const proxyOutbound = ref('')
const directOutbound = ref('')

const selectedPreset = computed(() =>
  routingDnsPresetCatalog.find(preset => preset.id === presetId.value) ?? routingDnsPresetCatalog[0])

const presetItems = computed(() =>
  routingDnsPresetCatalog.map(preset => ({ title: t(preset.titleKey), value: preset.id })))

const outboundItems = computed(() => {
  const tags = new Set(['direct', ...props.outboundTags.filter(Boolean)])
  return [...tags].map(tag => ({ title: tag, value: tag }))
})

const canApply = computed(() => proxyOutbound.value.length > 0 && directOutbound.value.length > 0)

// Same proxy and direct outbound silently defeats the split the preset exists
// for (both sides route through the same outbound). Warn, but still allow apply
// so an operator can stage the rule-set/DNS structure before wiring a proxy.
const sameOutbound = computed(() => proxyOutbound.value.length > 0 && proxyOutbound.value === directOutbound.value)

const preview = computed<ApplyPresetResult>(() => {
  if (!canApply.value) return { config: props.config, changes: [] }
  return applyRoutingDnsPreset(props.config, presetId.value, {
    proxyOutbound: proxyOutbound.value,
    directOutbound: directOutbound.value,
  })
})
</script>

<style scoped>
.preset-gallery {
  background: rgb(var(--v-theme-surface));
  border: 1px solid rgb(var(--v-theme-on-surface) / 12%);
  border-radius: 8px;
  display: grid;
  gap: 12px;
  margin-block: 12px 18px;
  min-width: 0;
  padding: 14px;
}

.preset-gallery__header {
  align-items: flex-start;
  display: flex;
  gap: 12px;
  justify-content: space-between;
  min-width: 0;
}

.preset-gallery__header h2 {
  font-size: 1rem;
  font-weight: 650;
  line-height: 1.3;
  margin: 0;
}

.preset-gallery__header p,
.preset-gallery__description {
  color: rgb(var(--v-theme-on-surface) / 70%);
  font-size: 0.84rem;
  line-height: 1.45;
  margin: 0;
  overflow-wrap: anywhere;
}

.preset-gallery__sources {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.preset-gallery__sources span {
  color: rgb(var(--v-theme-on-surface) / 62%);
  font-size: 0.78rem;
  font-weight: 650;
}

.preset-gallery__sources a {
  font-size: 0.78rem;
  overflow-wrap: anywhere;
}

.preset-gallery__preview {
  background: rgb(var(--v-theme-on-surface) / 4%);
  border-radius: 8px;
  padding: 10px 12px;
}

.preset-gallery__preview strong {
  display: block;
  font-size: 0.82rem;
  margin-bottom: 4px;
}

.preset-gallery__preview ul {
  margin: 0;
  padding-inline-start: 18px;
}

.preset-gallery__preview li {
  font-size: 0.8rem;
  line-height: 1.45;
}

.preset-gallery__actions {
  display: flex;
  justify-content: flex-end;
}

@media (max-width: 600px) {
  .preset-gallery__header,
  .preset-gallery__actions {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
