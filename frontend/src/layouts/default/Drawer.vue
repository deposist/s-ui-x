<template>
  <v-navigation-drawer
    v-model="showDrawer"
    :temporary="isMobile"
    :expand-on-hover="!isMobile"
    :rail="!isMobile"
    :permanent="!isMobile"
    @click="isMobile ? $emit('toggleDrawer') : null"
  >
    <v-list-item
      height="63"
      title="S-UI"
    >
      <template v-slot:prepend>
        <span aria-hidden="true" class="default-drawer__logo">S</span>
      </template>
      <template v-slot:append v-if="isMobile">
        <v-icon icon="mdi-close" />
      </template>
    </v-list-item>

    <v-divider></v-divider>

    <v-list density="compact" nav>
      <v-list-item link
        v-for="item in menu"
        :key="item.title"
        :to="item.path"
        :active="router.currentRoute.value.path == item.path">
        <template v-slot:prepend>
          <v-icon :icon="item.classicIcon"></v-icon>
        </template>
        <v-list-item-title v-text="$t(item.title)"></v-list-item-title>
      </v-list-item>
    </v-list>
    <template v-slot:append>
      <v-list-item prepend-icon="mdi-logout" :title="$t('menu.logout')" @click="Logout"></v-list-item>
    </template>
  </v-navigation-drawer>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import router from '@/router'
import { logout } from '@/plugins/httputil'
import { appMenu as menu } from '@/layouts/menu'

const props = defineProps(['isMobile','displayDrawer'])

const showDrawer = computed((): boolean => {
  return props.displayDrawer
})


const Logout = async () => {
  logout()
}
</script>

<style scoped>
.default-drawer__logo {
  align-items: center;
  background: #00d4ff;
  border-radius: 6px;
  color: #0a0a0a;
  display: inline-flex;
  font-size: 18px;
  font-weight: 700;
  height: 32px;
  justify-content: center;
  line-height: 1;
  width: 32px;
}
</style>
