<template>
  <section class="settings-ip-cert">
    <div class="settings-ip-cert__header">
      <div class="settings-ip-cert__heading">
        <v-icon color="primary" icon="lucide:shield-check" />
        <div>
          <h3>{{ $t('ipCert.title') }}</h3>
          <p>{{ $t('ipCert.subtitle') }}</p>
        </div>
      </div>
      <div class="settings-ip-cert__actions">
        <v-chip :color="status?.issued ? 'success' : 'default'" density="compact" label>
          {{ status?.issued ? $t('ipCert.issued') : $t('ipCert.notIssued') }}
        </v-chip>
      </div>
    </div>

    <v-row dense>
      <v-col cols="12" sm="6">
        <v-text-field
          v-model="form.ip"
          density="comfortable"
          hide-details="auto"
          :label="$t('ipCert.ip')"
          variant="outlined"
        />
      </v-col>
      <v-col cols="12" sm="6">
        <v-text-field
          v-model="form.email"
          density="comfortable"
          hide-details="auto"
          :label="$t('ipCert.email')"
          type="email"
          variant="outlined"
        />
      </v-col>
      <v-col cols="12" sm="6">
        <v-text-field
          v-model.number="form.port"
          density="comfortable"
          :hint="$t('ipCert.portHint')"
          persistent-hint
          :label="$t('ipCert.port')"
          type="number"
          variant="outlined"
        />
      </v-col>
      <v-col cols="12" sm="6">
        <v-select
          v-model="form.applyTarget"
          density="comfortable"
          hide-details="auto"
          :items="applyTargets"
          :label="$t('ipCert.applyTarget')"
          variant="outlined"
        />
      </v-col>
    </v-row>

    <v-switch
      v-model="form.autoRenew"
      color="primary"
      density="compact"
      hide-details
      :label="$t('ipCert.autoRenew')"
    />

    <v-alert
      v-if="isPanelTarget"
      density="compact"
      type="info"
      variant="tonal"
    >
      {{ $t('ipCert.restartNotice') }}
    </v-alert>

    <div v-if="status?.issued" class="settings-ip-cert__status">
      <span><strong>{{ $t('ipCert.expires') }}:</strong> {{ formatDate(status.notAfter) }}</span>
      <span><strong>{{ $t('ipCert.daysRemaining') }}:</strong> {{ daysRemaining }}</span>
      <span v-if="status.lastIssue"><strong>{{ $t('ipCert.lastRenew') }}:</strong> {{ formatDate(status.lastIssue) }}</span>
      <span v-if="status.certPath" class="settings-ip-cert__path">
        <strong>{{ $t('ipCert.certPath') }}:</strong> {{ status.certPath }}
      </span>
    </div>

    <div class="settings-ip-cert__footer">
      <v-btn
        color="primary"
        :disabled="!form.ip || !form.email"
        :loading="loading"
        prepend-icon="lucide:shield-check"
        variant="tonal"
        @click="issue"
      >
        {{ loading ? $t('ipCert.issuing') : $t('ipCert.issue') }}
      </v-btn>
    </div>
  </section>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import HttpUtils from '@/plugins/httputil'
import Data from '@/store/modules/data'
import type { IpCertStatus } from '@/types/ipcert'

const { t } = useI18n()
const tlsConfigs = computed<any[]>(() => Data().tlsConfigs ?? [])

const loading = ref(false)
const status = ref<IpCertStatus>()

const form = reactive({
  ip: '',
  email: '',
  port: 80,
  applyTarget: 'panel',
  autoRenew: false,
})

const applyTargets = computed(() => {
  const items = [{ title: t('ipCert.applyPanel'), value: 'panel' }]
  for (const tls of tlsConfigs.value) {
    items.push({ title: tls.name ?? `TLS #${tls.id}`, value: `inbound:${tls.id}` })
  }
  return items
})

const isPanelTarget = computed(() => !form.applyTarget || form.applyTarget === 'panel')

const daysRemaining = computed(() =>
  status.value ? Math.max(0, Math.round(status.value.daysRemaining * 10) / 10) : 0,
)

const formatDate = (raw: string): string => {
  if (!raw) return '—'
  const d = new Date(raw)
  return Number.isNaN(d.getTime()) ? raw : d.toLocaleString()
}

const loadStatus = async () => {
  const msg = await HttpUtils.get('api/ip-cert/status')
  if (msg.success && msg.obj) {
    status.value = msg.obj as IpCertStatus
  }
}

const loadSettings = async () => {
  const msg = await HttpUtils.get('api/settings')
  if (!msg.success || !msg.obj) return
  const s = msg.obj
  form.ip = s.ipCertTargetIP ?? ''
  form.email = s.ipCertEmail ?? ''
  form.port = Number(s.ipCertChallengePort ?? 80)
  form.applyTarget = s.ipCertApplyTarget || 'panel'
  form.autoRenew = s.ipCertEnabled === 'true' || s.ipCertEnabled === true
}

// Persist the user-editable controls so the renewal cron uses the same params.
const saveControls = async (): Promise<boolean> => {
  const payload = {
    ipCertEnabled: String(form.autoRenew),
    ipCertTargetIP: form.ip,
    ipCertEmail: form.email,
    ipCertChallengePort: String(form.port),
    ipCertApplyTarget: form.applyTarget,
  }
  const msg = await HttpUtils.post('api/save', {
    object: 'settings',
    action: 'set',
    data: JSON.stringify(payload),
  })
  return msg.success
}

const issue = async () => {
  loading.value = true
  try {
    if (!(await saveControls())) return
    const msg = await HttpUtils.post('api/ip-cert/issue', {
      data: JSON.stringify({
        ip: form.ip,
        email: form.email,
        port: form.port,
        applyTarget: form.applyTarget,
      }),
    })
    if (msg.success && msg.obj) {
      status.value = msg.obj as IpCertStatus
    }
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadStatus(), loadSettings()])
})
</script>

<style scoped>
.settings-ip-cert {
  border: 1px solid rgba(var(--v-theme-on-surface), 0.12);
  border-radius: 8px;
  display: grid;
  gap: 14px;
  min-width: 0;
  padding: 16px;
}

.settings-ip-cert__header {
  align-items: flex-start;
  display: flex;
  gap: 16px;
  justify-content: space-between;
  min-width: 0;
}

.settings-ip-cert__heading {
  align-items: flex-start;
  display: flex;
  gap: 12px;
  min-width: 0;
}

.settings-ip-cert__heading h3,
.settings-ip-cert__heading p {
  letter-spacing: 0;
  margin: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

.settings-ip-cert__heading h3 {
  font-size: 1rem;
  font-weight: 600;
  line-height: 1.4;
}

.settings-ip-cert__heading p {
  color: rgba(var(--v-theme-on-surface), 0.72);
  font-size: 0.875rem;
  line-height: 1.4;
  margin-top: 2px;
}

.settings-ip-cert__actions {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: flex-end;
}

.settings-ip-cert__status {
  color: rgba(var(--v-theme-on-surface), 0.82);
  display: grid;
  font-size: 0.86rem;
  gap: 4px;
  min-width: 0;
}

.settings-ip-cert__path {
  font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace;
  font-size: 0.78rem;
  overflow-wrap: anywhere;
}

.settings-ip-cert__footer {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

@media (max-width: 600px) {
  .settings-ip-cert {
    padding: 12px;
  }

  .settings-ip-cert__header,
  .settings-ip-cert__actions {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
