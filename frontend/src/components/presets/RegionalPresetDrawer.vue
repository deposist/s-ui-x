<template>
  <v-navigation-drawer
    v-model="visible"
    class="regional-preset-drawer"
    location="right"
    temporary
    :width="drawerWidth"
  >
    <div class="regional-preset-drawer__shell">
      <header class="regional-preset-drawer__header">
        <div>
          <h2>{{ t('regionalPresets.title') }}</h2>
          <p>{{ t('regionalPresets.subtitle') }}</p>
        </div>
        <v-btn
          :aria-label="t('actions.close')"
          icon="mdi-close"
          size="small"
          variant="text"
          @click="closeDrawer"
        />
      </header>

      <main class="regional-preset-drawer__body">
        <template v-if="step === 'selection'">
          <v-alert density="compact" type="info" variant="tonal">
            {{ t('regionalPresets.security.note') }}
          </v-alert>

          <v-row>
            <v-col cols="12" sm="6">
              <v-select
                v-model="proxyOutbound"
                density="compact"
                hide-details
                :items="outboundItems"
                :label="t('regionalPresets.proxyOutbound')"
                variant="outlined"
              />
            </v-col>
            <v-col cols="12" sm="6">
              <v-select
                v-model="directOutbound"
                density="compact"
                hide-details
                :items="outboundItems"
                :label="t('regionalPresets.directOutbound')"
                variant="outlined"
              />
            </v-col>
          </v-row>

          <v-alert v-if="!hasOutbounds" density="compact" type="warning" variant="tonal">
            {{ t('regionalPresets.selectOutbounds') }}
          </v-alert>
          <v-alert v-else-if="sameOutbound" density="compact" type="warning" variant="tonal">
            {{ t('regionalPresets.sameOutboundWarning') }}
          </v-alert>

          <section class="regional-preset-drawer__cards">
            <region-card
              :state="ruState"
              :status="regionStatus('RU')"
              :title="t('regionalPresets.region.ru.title')"
              :description="t('regionalPresets.region.ru.description')"
              region-label="RU"
              :dns-text="dnsText(ruState, 'RU')"
              :exception-error="exceptionErrors.ru"
              @toggle="ruState.enabled = $event"
              @direction="ruState.direction = $event"
              @add-exception="addException('ru', $event)"
              @remove-exception="removeException('ru', $event)"
            />

            <region-card
              :state="zhState"
              :status="regionStatus('ZH')"
              :title="t('regionalPresets.region.zh.title')"
              :description="t('regionalPresets.region.zh.description')"
              region-label="ZH"
              :dns-text="dnsText(zhState, 'ZH')"
              :exception-error="exceptionErrors.zh"
              @toggle="zhState.enabled = $event"
              @direction="zhState.direction = $event"
              @add-exception="addException('zh', $event)"
              @remove-exception="removeException('zh', $event)"
            />
          </section>

          <div class="regional-preset-drawer__manual-link">
            <span>{{ t('regionalPresets.needFullControl') }}</span>
            <span>{{ t('regionalPresets.editRulesManually') }}</span>
          </div>
        </template>

        <template v-else-if="step === 'preview'">
          <v-alert density="compact" type="info" variant="tonal">
            {{ t('regionalPresets.previewGroups.securityNote') }}
          </v-alert>

          <preview-card
            :title="t('regionalPresets.region.ru.title')"
            :state="ruState"
            :group="preview.ru"
          />
          <preview-card
            :title="t('regionalPresets.region.zh.title')"
            :state="zhState"
            :group="preview.zh"
          />
        </template>

        <template v-else-if="step === 'success'">
          <div class="regional-preset-drawer__result">
            <v-icon color="success" icon="mdi-check-circle" size="56" />
            <h3>{{ t('regionalPresets.applied') }}</h3>
            <p>{{ t('regionalPresets.result.customItemsKept') }}</p>
            <div class="regional-preset-drawer__result-summary">
              <span>{{ t('regionalPresets.region.ru.title') }}: {{ resultLabel(ruState) }}</span>
              <span>{{ t('regionalPresets.region.zh.title') }}: {{ resultLabel(zhState) }}</span>
            </div>
          </div>
        </template>

        <template v-else-if="step === 'error'">
          <div class="regional-preset-drawer__result">
            <v-icon color="error" icon="mdi-alert-circle-outline" size="56" />
            <h3>{{ t('regionalPresets.result.failed') }}</h3>
            <p>{{ errorMessage }}</p>
          </div>
        </template>
      </main>

      <footer class="regional-preset-drawer__footer">
        <template v-if="step === 'selection'">
          <v-btn variant="text" @click="closeDrawer">{{ t('regionalPresets.cancel') }}</v-btn>
          <v-btn color="primary" :disabled="!canPreview" variant="flat" @click="openPreview">
            {{ t('regionalPresets.preview') }}
          </v-btn>
        </template>
        <template v-else-if="step === 'preview'">
          <v-btn variant="text" @click="step = 'selection'">{{ t('regionalPresets.back') }}</v-btn>
          <v-btn color="primary" variant="flat" @click="applySelectedPresets">
            {{ t('regionalPresets.apply') }}
          </v-btn>
        </template>
        <template v-else-if="step === 'success'">
          <v-btn color="primary" variant="flat" @click="closeDrawer">{{ t('regionalPresets.done') }}</v-btn>
        </template>
        <template v-else>
          <v-btn color="primary" variant="tonal" @click="step = 'selection'">{{ t('regionalPresets.back') }}</v-btn>
        </template>
      </footer>
    </div>
  </v-navigation-drawer>
</template>

<script lang="ts" setup>
import { computed, defineComponent, h, reactive, ref, resolveComponent, watch, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'

import type { Config } from '@/types/config'
import {
  applyPresets,
  computePreview,
  detectPresetState,
  isPresetManagedItem,
  type PresetDirection,
  type PresetPreviewGroup,
  type PresetRegion,
  type PresetRegionKey,
  type RegionalPresetState,
  validatePresetCatalogShape,
} from './routingDnsPresets'

const props = defineProps<{
  modelValue: boolean
  config: Config
  outboundTags: string[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  apply: [config: Config]
}>()

const { t } = useI18n()
const drawerWidth = 520
const step = ref<'selection' | 'preview' | 'success' | 'error'>('selection')
const proxyOutbound = ref('')
const directOutbound = ref('direct')
const errorMessage = ref('')

const ruState = reactive<RegionalPresetState>({ region: 'RU', enabled: false, direction: 'direct', exceptions: [] })
const zhState = reactive<RegionalPresetState>({ region: 'ZH', enabled: false, direction: 'direct', exceptions: [] })
const exceptionErrors = reactive<Record<PresetRegionKey, string>>({ ru: '', zh: '' })

const visible = computed({
  get: () => props.modelValue,
  set: value => emit('update:modelValue', value),
})

const outboundItems = computed(() => {
  const tags = new Set(['direct', ...props.outboundTags.filter(Boolean)])
  return [...tags].map(tag => ({ title: tag, value: tag }))
})

const hasOutbounds = computed(() => proxyOutbound.value.length > 0 && directOutbound.value.length > 0)
const sameOutbound = computed(() => hasOutbounds.value && proxyOutbound.value === directOutbound.value)
const hasEnabledRegion = computed(() => ruState.enabled || zhState.enabled)
const canPreview = computed(() => hasEnabledRegion.value && hasOutbounds.value)

const preview = computed(() => {
  if (!hasOutbounds.value) {
    return {
      ru: emptyPreviewGroup(),
      zh: emptyPreviewGroup(),
    }
  }
  return computePreview(props.config, ruState, zhState, {
    proxyOutbound: proxyOutbound.value,
    directOutbound: directOutbound.value,
  })
})

watch(() => props.modelValue, open => {
  if (open) resetFromConfig()
})

const emptyPreviewGroup = (): PresetPreviewGroup => ({
  willAdd: [],
  willChange: [],
  willKeep: [],
  willRemove: [],
  securityWarnings: [],
})

const assignState = (target: RegionalPresetState, source: RegionalPresetState) => {
  target.region = source.region
  target.enabled = source.enabled
  target.direction = source.direction
  target.exceptions = [...source.exceptions]
}

const resetFromConfig = () => {
  const detected = detectPresetState(props.config)
  assignState(ruState, detected.ru)
  assignState(zhState, detected.zh)
  proxyOutbound.value = props.outboundTags.find(tag => tag && tag !== 'direct') ?? ''
  directOutbound.value = outboundItems.value.some(item => item.value === 'direct') ? 'direct' : (props.outboundTags[0] ?? '')
  exceptionErrors.ru = ''
  exceptionErrors.zh = ''
  errorMessage.value = ''
  step.value = 'selection'
}

const closeDrawer = () => {
  visible.value = false
}

const isValidDomain = (value: string) => {
  const domain = value.trim().replace(/^\.+|\.+$/g, '').toLowerCase()
  if (!domain || domain.includes('/') || domain.includes('*') || domain.includes(' ')) return false
  return /^[a-z0-9-]+(\.[a-z0-9-]+)+$/i.test(domain)
}

const addException = (region: PresetRegionKey, value: string) => {
  const normalized = value.trim().replace(/^\.+|\.+$/g, '').toLowerCase()
  exceptionErrors[region] = ''
  if (!isValidDomain(normalized)) {
    exceptionErrors[region] = t('regionalPresets.advanced.invalidDomain')
    return
  }
  const target = region === 'ru' ? ruState : zhState
  if (!target.exceptions.includes(normalized)) target.exceptions.push(normalized)
}

const removeException = (region: PresetRegionKey, index: number) => {
  const target = region === 'ru' ? ruState : zhState
  target.exceptions.splice(index, 1)
}

const dnsText = (state: RegionalPresetState, label: string) => t('regionalPresets.dns.behavior', {
  mode: t(`regionalPresets.direction.${state.direction}.title`),
  region: label,
})

const hasCustomRegionalConfig = (region: PresetRegion) => {
  const needle = region === 'RU' ? ['ru', 'russia', 'blocked', 'private'] : ['cn', 'china', 'zh']
  const raw = JSON.stringify(props.config?.route ?? {}).toLowerCase()
  return needle.some(item => raw.includes(item)) && !detectPresetState(props.config)[region === 'RU' ? 'ru' : 'zh'].enabled
}

const regionStatus = (region: PresetRegion) => {
  const state = region === 'RU' ? ruState : zhState
  if (state.enabled) return t('regionalPresets.region.status.enabled')
  if (hasCustomRegionalConfig(region)) return t('regionalPresets.region.status.customDetected')
  return t('regionalPresets.region.status.notConfigured')
}

const resultLabel = (state: RegionalPresetState) => state.enabled
  ? t(`regionalPresets.direction.${state.direction}.title`)
  : t('disable')

const openPreview = () => {
  if (!validatePresetCatalogShape()) {
    errorMessage.value = t('regionalPresets.result.regionalDataUnavailable')
    step.value = 'error'
    return
  }
  step.value = 'preview'
}

const applySelectedPresets = () => {
  try {
    const result = applyPresets(props.config, ruState, zhState, {
      proxyOutbound: proxyOutbound.value,
      directOutbound: directOutbound.value,
    })
    emit('apply', result.config)
    step.value = 'success'
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : t('regionalPresets.result.regionalDataUnavailable')
    step.value = 'error'
  }
}

const groupItems = (group: PresetPreviewGroup) => [
  { key: 'willAdd', title: t('regionalPresets.previewGroups.willAdd'), items: group.willAdd },
  { key: 'willChange', title: t('regionalPresets.previewGroups.willChange'), items: group.willChange },
  { key: 'willKeep', title: t('regionalPresets.previewGroups.willKeep'), items: group.willKeep },
  { key: 'willRemove', title: t('regionalPresets.previewGroups.willRemove'), items: group.willRemove },
]

const PreviewCard = defineComponent({
  name: 'PreviewCard',
  props: {
    title: { type: String, required: true },
    state: { type: Object as PropType<RegionalPresetState>, required: true },
    group: { type: Object as PropType<PresetPreviewGroup>, required: true },
  },
  setup(cardProps) {
    return () => h('section', { class: 'regional-preset-drawer__preview-card' }, [
      h('div', { class: 'regional-preset-drawer__preview-title' }, [
        h('strong', cardProps.title),
        h('span', cardProps.state.enabled
          ? t(`regionalPresets.direction.${cardProps.state.direction}.title`)
          : t('regionalPresets.previewGroups.noChanges')),
      ]),
      ...groupItems(cardProps.group).map(section => section.items.length > 0
        ? h('div', { class: 'regional-preset-drawer__preview-group', key: section.key }, [
          h('span', { class: 'regional-preset-drawer__preview-group-title' }, section.title),
          h('ul', section.items.map(item => h('li', { key: item }, item))),
        ])
        : null),
      cardProps.group.securityWarnings.length > 0
        ? h('div', { class: 'regional-preset-drawer__security-warnings' }, [
          h('span', { class: 'regional-preset-drawer__preview-group-title' }, t('regionalPresets.previewGroups.securityWarnings')),
          ...cardProps.group.securityWarnings.map(warning => h('p', { key: warning }, t(warning))),
        ])
        : h('p', { class: 'regional-preset-drawer__muted' }, t('regionalPresets.previewGroups.noWarnings')),
    ])
  },
})

const RegionCard = defineComponent({
  name: 'RegionCard',
  props: {
    state: { type: Object as PropType<RegionalPresetState>, required: true },
    status: { type: String, required: true },
    title: { type: String, required: true },
    description: { type: String, required: true },
    regionLabel: { type: String, required: true },
    dnsText: { type: String, required: true },
    exceptionError: { type: String, required: true },
  },
  emits: ['toggle', 'direction', 'add-exception', 'remove-exception'],
  setup(cardProps, { emit: cardEmit }) {
    const exceptionInput = ref('')
    const add = () => {
      cardEmit('add-exception', exceptionInput.value)
      exceptionInput.value = ''
    }
    return () => h('section', { class: 'regional-preset-drawer__region-card' }, [
      h('div', { class: 'regional-preset-drawer__region-head' }, [
        h('div', [
          h('h3', cardProps.title),
          h('p', cardProps.description),
          h('span', { class: 'regional-preset-drawer__status' }, cardProps.status),
        ]),
        h(resolveVuetify('VSwitch'), {
          modelValue: cardProps.state.enabled,
          color: 'primary',
          density: 'compact',
          hideDetails: true,
          'onUpdate:modelValue': (value: boolean) => cardEmit('toggle', value),
        }),
      ]),
      h('div', { class: ['regional-preset-drawer__region-body', !cardProps.state.enabled ? 'is-disabled' : ''] }, [
        h(resolveVuetify('VRadioGroup'), {
          modelValue: cardProps.state.direction,
          hideDetails: true,
          disabled: !cardProps.state.enabled,
          'onUpdate:modelValue': (value: PresetDirection) => cardEmit('direction', value),
        }, {
          default: () => [
            h(resolveVuetify('VRadio'), {
              label: `${t('regionalPresets.direction.direct.title')} — ${t('regionalPresets.direction.direct.description')}`,
              value: 'direct',
            }),
            h(resolveVuetify('VRadio'), {
              label: `${t('regionalPresets.direction.proxy.title')} — ${t('regionalPresets.direction.proxy.description')}`,
              value: 'proxy',
            }),
          ],
        }),
        h('p', { class: 'regional-preset-drawer__dns-text' }, cardProps.dnsText),
        h(resolveVuetify('VExpansionPanels'), { variant: 'accordion' }, {
          default: () => h(resolveVuetify('VExpansionPanel'), null, {
            title: () => t('regionalPresets.advanced.title'),
            text: () => h('div', { class: 'regional-preset-drawer__exceptions' }, [
              h('p', t('regionalPresets.advanced.exceptionsHelp')),
              h('div', { class: 'regional-preset-drawer__exception-input' }, [
                h(resolveVuetify('VTextField'), {
                  modelValue: exceptionInput.value,
                  density: 'compact',
                  disabled: !cardProps.state.enabled,
                  errorMessages: cardProps.exceptionError,
                  hideDetails: !cardProps.exceptionError,
                  label: t('regionalPresets.advanced.exceptions'),
                  variant: 'outlined',
                  'onUpdate:modelValue': (value: string) => { exceptionInput.value = value },
                  onKeydown: (event: KeyboardEvent) => {
                    if (event.key === 'Enter') add()
                  },
                }),
                h(resolveVuetify('VBtn'), {
                  disabled: !cardProps.state.enabled,
                  variant: 'tonal',
                  onClick: add,
                }, () => t('regionalPresets.advanced.addDomain')),
              ]),
              cardProps.state.exceptions.length === 0
                ? h('p', { class: 'regional-preset-drawer__muted' }, t('regionalPresets.advanced.noExceptions'))
                : h('div', { class: 'regional-preset-drawer__exception-list' }, cardProps.state.exceptions.map((item, index) =>
                  h(resolveVuetify('VChip'), {
                    key: item,
                    closable: true,
                    label: true,
                    size: 'small',
                    variant: 'tonal',
                    'onClick:close': () => cardEmit('remove-exception', index),
                  }, () => item))),
            ]),
          }),
        }),
      ]),
    ])
  },
})

function resolveVuetify(name: string) {
  return resolveComponent(name)
}

defineExpose({ isPresetManagedItem })
</script>

<style scoped>
.regional-preset-drawer {
  border-inline-start: 1px solid rgb(var(--v-theme-on-surface) / 12%);
}

.regional-preset-drawer__shell {
  display: grid;
  grid-template-rows: auto 1fr auto;
  height: 100%;
  min-height: 0;
}

.regional-preset-drawer__header,
.regional-preset-drawer__footer {
  background: rgb(var(--v-theme-surface));
  border-block-end: 1px solid rgb(var(--v-theme-on-surface) / 12%);
  display: flex;
  gap: 12px;
  justify-content: space-between;
  padding: 16px;
}

.regional-preset-drawer__footer {
  border-block-end: 0;
  border-block-start: 1px solid rgb(var(--v-theme-on-surface) / 12%);
}

.regional-preset-drawer__header h2,
.regional-preset-drawer__region-head h3,
.regional-preset-drawer__result h3 {
  font-size: 1rem;
  font-weight: 650;
  line-height: 1.3;
  margin: 0;
}

.regional-preset-drawer__header p,
.regional-preset-drawer__region-head p,
.regional-preset-drawer__dns-text,
.regional-preset-drawer__muted,
.regional-preset-drawer__result p,
.regional-preset-drawer__manual-link {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-size: 0.82rem;
  line-height: 1.45;
  margin: 0;
}

.regional-preset-drawer__body {
  display: grid;
  gap: 14px;
  min-height: 0;
  overflow: auto;
  padding: 16px;
}

.regional-preset-drawer__cards {
  display: grid;
  gap: 14px;
}

.regional-preset-drawer__region-card,
.regional-preset-drawer__preview-card {
  background: rgb(var(--v-theme-surface));
  border: 1px solid rgb(var(--v-theme-on-surface) / 12%);
  border-radius: 14px;
  display: grid;
  gap: 12px;
  padding: 14px;
}

.regional-preset-drawer__region-head,
.regional-preset-drawer__preview-title {
  align-items: flex-start;
  display: flex;
  gap: 12px;
  justify-content: space-between;
}

.regional-preset-drawer__status {
  background: rgb(var(--v-theme-on-surface) / 8%);
  border-radius: 999px;
  color: rgb(var(--v-theme-on-surface) / 76%);
  display: inline-flex;
  font-size: 0.72rem;
  font-weight: 650;
  margin-top: 8px;
  padding: 3px 8px;
}

.regional-preset-drawer__region-body {
  display: grid;
  gap: 10px;
}

.regional-preset-drawer__region-body.is-disabled {
  opacity: 0.55;
}

.regional-preset-drawer__manual-link {
  display: flex;
  gap: 6px;
}

.regional-preset-drawer__preview-group {
  display: grid;
  gap: 4px;
}

.regional-preset-drawer__preview-group-title {
  color: rgb(var(--v-theme-on-surface) / 74%);
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
}

.regional-preset-drawer__preview-group ul {
  margin: 0;
  padding-inline-start: 18px;
}

.regional-preset-drawer__preview-group li,
.regional-preset-drawer__security-warnings p {
  font-size: 0.82rem;
  line-height: 1.45;
}

.regional-preset-drawer__security-warnings {
  background: rgb(var(--v-theme-warning) / 12%);
  border: 1px solid rgb(var(--v-theme-warning) / 30%);
  border-radius: 10px;
  display: grid;
  gap: 4px;
  padding: 10px;
}

.regional-preset-drawer__exceptions,
.regional-preset-drawer__exception-input {
  display: grid;
  gap: 10px;
}

.regional-preset-drawer__exception-input {
  grid-template-columns: 1fr auto;
}

.regional-preset-drawer__exception-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.regional-preset-drawer__result {
  align-items: center;
  align-self: center;
  display: grid;
  gap: 12px;
  justify-items: center;
  text-align: center;
}

.regional-preset-drawer__result-summary {
  background: rgb(var(--v-theme-on-surface) / 5%);
  border-radius: 12px;
  display: grid;
  gap: 6px;
  min-width: min(320px, 100%);
  padding: 12px;
  text-align: start;
}

@media (max-width: 600px) {
  .regional-preset-drawer__exception-input {
    grid-template-columns: 1fr;
  }
}
</style>
