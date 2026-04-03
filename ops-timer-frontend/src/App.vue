<template>
  <v-app>
    <v-progress-linear
      v-if="isNavigating"
      indeterminate
      color="primary"
      height="2"
      style="position: fixed; top: 0; left: 0; right: 0; z-index: 9999"
    />
    <router-view />
  </v-app>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const isNavigating = ref(false)
const router = useRouter()

router.beforeEach(() => {
  isNavigating.value = true
})

router.afterEach(() => {
  isNavigating.value = false
})
</script>
