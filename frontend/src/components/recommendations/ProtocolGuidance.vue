<template>
  <v-alert
    v-if="hintKey"
    class="protocol-guidance"
    density="compact"
    type="info"
    variant="tonal"
  >
    <div class="protocol-guidance__header">
      <strong>{{ $t('guidance.title') }}</strong>
    </div>
    <p class="text-caption mt-1 mb-0">{{ $t(hintKey) }}</p>
    <v-list v-if="resolved.length > 0" bg-color="transparent" density="compact" class="pt-1 pb-0">
      <v-list-item v-for="item in resolved" :key="item.id" class="px-0">
        <v-list-item-title>{{ $t(item.labelKey) }}</v-list-item-title>
        <v-list-item-subtitle v-if="item.descriptionKey">{{ $t(item.descriptionKey) }}</v-list-item-subtitle>
        <div class="text-caption mt-1">
          {{ $t('guidance.recommendedValue') }}: <code>{{ formatValue(item.recommendedValue) }}</code>
        </div>
      </v-list-item>
    </v-list>
  </v-alert>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { RuntimeCapabilities } from '@/types/runtimeCapabilities'
import { protocolGuidance, type ProtocolGuidanceCategory } from '@/utils/protocolGuidance'
import {
  resolveRecommendations,
  type RecommendationContext,
  type RecommendationMode,
} from '@/utils/recommendations'

const props = defineProps<{
  capabilities: RuntimeCapabilities | null
  category: ProtocolGuidanceCategory
  mode: RecommendationMode
  model: object
}>()

const model = computed(() => props.model as Record<string, unknown>)
const guidance = computed(() => protocolGuidance(
  props.capabilities,
  props.category,
  typeof model.value.type === 'string' ? model.value.type : '',
))
const hintKey = computed(() => guidance.value.hintKey)
const context = computed<RecommendationContext<Record<string, unknown>>>(() => ({
  mode: props.mode,
  model: model.value,
}))
const resolved = computed(() => resolveRecommendations(guidance.value.specs, context.value))

const formatValue = (value: unknown): string => typeof value === 'string' ? value : JSON.stringify(value)
</script>

<style scoped>
.protocol-guidance {
  margin-block: 8px;
}

.protocol-guidance__header {
  align-items: center;
  display: flex;
  gap: 8px;
}

code {
  overflow-wrap: anywhere;
}
</style>
