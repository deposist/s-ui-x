<template>
  <entity-drawer
    :dirty="dirty"
    :loading="loading"
    :model-value="visible"
    :save-disabled="saveBlockedReason !== ''"
    :save-disabled-reason="saveBlockedReason"
    :saving="loading"
    :title="$t('actions.' + title) + ' ' + $t('objects.service')"
    :width="720"
    @close="closeModal"
    @save="saveChanges"
  >
    <form-section icon="lucide:sliders-horizontal" :title="$t('form.sections.configuration')">
      <v-row>
        <v-col cols="12" sm="6">
          <v-select
            hide-details
            :label="$t('type')"
            :items="srvTypeItems"
            v-model="srv.type"
            @update:modelValue="changeType">
          </v-select>
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field v-model="srv.tag" :label="$t('objects.tag')" hide-details :error="isBlankIdentity(srv.tag)"></v-text-field>
        </v-col>
      </v-row>

      <Listen v-if="!NoListen.includes(srv.type)" :data="srv" :inTags="inTags" />
      <Derp v-if="srv.type == srvTypes.DERP" :data="srv" :inTags="inTags" :tsTags="tsTags" />
      <SSMapi v-if="srv.type == srvTypes.SSMAPI" :data="srv" :ssTags="ssTags" />
      <InTLS v-if="HasTls.includes(srv.type)"  :inbound="srv" :tlsConfigs="tlsConfigs" :tls_id="srv.tls_id" />
    </form-section>
  </entity-drawer>
</template>

<script lang="ts">
import Data from '@/store/modules/data'
import { SrvType, SrvTypes, availableSrvTypes, createSrv, defaultSrvType } from '@/types/services'
import RandomUtil from '@/plugins/randomUtil'
import Listen from '@/components/Listen.vue'
import Derp from '@/components/services/Derp.vue'
import InTLS from '@/components/tls/InTLS.vue'
import SSMapi from '@/components/services/SSMAPI.vue'
import EntityDrawer from './EntityDrawer.vue'
import FormSection from './FormSection.vue'
import { isBlankIdentity } from '@/utils/entityIdentity'
export default {
  inheritAttrs: false,
  props: ['visible', 'data', 'id', 'inTags', 'tsTags', 'ssTags', 'tlsConfigs'],
  emits: ['close'],
  data() {
    return {
      srv: createSrv(defaultSrvType(Data().capabilities?.services), { "tag": "" }),
      title: "add",
      loading: false,
      snapshot: "",
      srvTypes: SrvTypes,
      HasTls: [SrvTypes.DERP, SrvTypes.SSMAPI] as SrvType[],
      NoListen: [] as SrvType[],
    }
  },
  methods: {
    isBlankIdentity,
    async updateData(id: number) {
      if (id > 0) {
        const newData = JSON.parse(this.$props.data)
        this.srv = createSrv(newData.type, newData)
        this.title = "edit"
      }
      else {
        const port = RandomUtil.randomIntRange(10000, 60000)
        this.srv = createSrv(defaultSrvType(Data().capabilities?.services), {
          tag: defaultSrvType(Data().capabilities?.services) + "-" + RandomUtil.randomSeq(3),
          listen: '::',
          listen_port: port,
        })
        this.title = "add"
      }
      this.snapshot = JSON.stringify(this.srv)
    },
    changeType() {
      // Tag change only in add service
      const tag = this.$props.id > 0 ? this.srv.tag : this.srv.type + "-" + RandomUtil.randomSeq(3)
      // Use previous data
      const prevConfig = { id: this.srv.id, tag: tag, listen: this.srv.listen, listen_port: this.srv.listen_port }
      this.srv = createSrv(this.srv.type, prevConfig)
    },
    closeModal() {
      this.updateData(0) // reset
      this.$emit('close')
    },
    async saveChanges() {
      if (!this.$props.visible || this.loading || this.saveBlockedReason !== '') return

      const isDuplicatedTag = Data().checkTag('service', this.srv.id, this.srv.tag)
      if (isDuplicatedTag) return

      // save data
      this.loading = true
      try {
        const success = await Data().save("services", this.$props.id == 0 ? "new" : "edit", this.srv)
        if (success) this.closeModal()
      } finally {
        this.loading = false
      }
    },
  },
  computed: {
    saveBlockedReason(): string {
      if (isBlankIdentity(this.srv.tag)) return this.$t('form.cannotSave.tagRequired')
      if (this.srv.listen_port != null && (this.srv.listen_port > 65535 || this.srv.listen_port < 1)) return this.$t('form.cannotSave.portRange')
      return ''
    },
    srvTypeItems() {
      return Object.entries(availableSrvTypes(Data().capabilities?.services)).map(([key, value]) => ({ title: key, value }))
    },
    dirty(): boolean {
      return this.snapshot !== "" && JSON.stringify(this.srv) !== this.snapshot
    },
  },
  watch: {
    visible(v) {
      if (v) {
        this.updateData(this.$props.id)
      }
    },
  },
  components: { EntityDrawer, FormSection, Listen, InTLS, Derp, SSMapi },
}
</script>
