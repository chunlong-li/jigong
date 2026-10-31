<template>
  <router-view />
  <SponsorFab v-if="showSponsor" storage-key="jigong_sponsor_v1" :hidden-routes="['/login']" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import SponsorFab from './components/SponsorFab.vue'

const route = useRoute()
const showSponsor = computed(() => !route.path.startsWith('/login'))
</script>
