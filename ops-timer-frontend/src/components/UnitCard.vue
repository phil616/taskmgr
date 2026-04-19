<template>
  <v-card class="rounded-lg unit-card" :class="{ 'border-start': true }" :style="borderStyle">
    <v-card-text class="pb-2">
      <div class="d-flex align-center mb-2">
        <v-icon :color="getTimeColor(unit.remaining_seconds, isCountdown)" size="20" class="mr-2">
          {{ getUnitTypeIcon(unit.type) }}
        </v-icon>
        <span class="text-body-2 text-medium-emphasis">{{ getUnitTypeLabel(unit.type) }}</span>
        <v-spacer />
        <v-chip
          :color="getStatusColor(unit.status)"
          size="x-small"
          variant="flat"
        >
          {{ getStatusLabel(unit.status) }}
        </v-chip>
      </div>

      <router-link :to="`/units/${unit.id}`" class="text-decoration-none text-high-emphasis">
        <h3 class="text-subtitle-1 font-weight-bold mb-1">{{ unit.title }}</h3>
      </router-link>

      <div v-if="unit.tags && unit.tags.length" class="mb-2">
        <v-chip v-for="tag in unit.tags.slice(0, 3)" :key="tag" size="x-small" variant="outlined" class="mr-1">
          {{ tag }}
        </v-chip>
      </div>

      <!-- Time countdown -->
      <div v-if="unit.type === 'time_countdown'" class="mt-3">
        <div class="text-h5 font-weight-bold" :class="`text-${getTimeColor(unit.remaining_seconds, true)}`">
          {{ displayTime }}
        </div>
        <v-progress-linear
          :model-value="timeCountdownProgress"
          :color="getTimeColor(unit.remaining_seconds, true)"
          bg-color="surface-variant"
          height="6"
          rounded
          class="mt-2"
        />
        <div class="d-flex justify-space-between text-caption text-medium-emphasis mt-1">
          <span>目标: {{ formatDate(unit.target_time) }}</span>
          <span>{{ Math.round(timeCountdownProgress) }}%</span>
        </div>
      </div>

      <!-- Time countup -->
      <div v-else-if="unit.type === 'time_countup'" class="mt-3">
        <div class="text-h5 font-weight-bold text-primary">
          {{ displayTime }}
        </div>
        <div class="text-caption text-medium-emphasis mt-1">
          起始: {{ formatDate(unit.start_time) }}
        </div>
      </div>

      <!-- Count countdown -->
      <div v-else-if="unit.type === 'count_countdown'" class="mt-3">
        <div class="d-flex align-center mb-1">
          <span class="text-h5 font-weight-bold">{{ unit.current_value || 0 }}</span>
          <span class="text-body-2 text-medium-emphasis mx-1">/</span>
          <span class="text-body-1">{{ unit.target_value }} {{ unit.unit_label }}</span>
        </div>
        <v-progress-linear
          :model-value="unit.progress || 0"
          color="primary"
          height="8"
          rounded
          class="mb-1"
        />
        <div class="text-caption text-medium-emphasis">
          还剩 {{ (unit.target_value || 0) - (unit.current_value || 0) }} {{ unit.unit_label }}
          ({{ Math.round(unit.progress || 0) }}%)
        </div>
      </div>

      <!-- Count countup -->
      <div v-else-if="unit.type === 'count_countup'" class="mt-3">
        <div class="text-h5 font-weight-bold text-primary">
          {{ unit.current_value || 0 }} <span class="text-body-2">{{ unit.unit_label }}</span>
        </div>
      </div>
    </v-card-text>

    <v-card-actions v-if="isCountType" class="pt-0">
      <v-btn size="small" variant="tonal" color="error" @click="$emit('step', unit.id, 'down')">
        <v-icon>mdi-minus</v-icon>
      </v-btn>
      <v-btn size="small" variant="tonal" color="success" @click="$emit('step', unit.id, 'up')">
        <v-icon>mdi-plus</v-icon>
      </v-btn>
      <v-spacer />
      <v-chip :color="getPriorityColor(unit.priority)" size="x-small" variant="tonal">
        {{ getPriorityLabel(unit.priority) }}
      </v-chip>
    </v-card-actions>
    <v-card-actions v-else class="pt-0">
      <v-spacer />
      <v-chip :color="getPriorityColor(unit.priority)" size="x-small" variant="tonal">
        {{ getPriorityLabel(unit.priority) }}
      </v-chip>
    </v-card-actions>
  </v-card>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import type { Unit } from '@/types'
import {
  formatDuration, formatDate, getTimeColor, getPriorityColor, getPriorityLabel,
  getStatusColor, getStatusLabel, getUnitTypeLabel, getUnitTypeIcon, parseAppTime,
} from '@/utils/time'

const props = defineProps<{ unit: Unit }>()
defineEmits<{ step: [id: string, direction: 'up' | 'down'] }>()

const now = ref(Date.now())
let timer: number | null = null

const isCountdown = computed(() =>
  props.unit.type === 'time_countdown' || props.unit.type === 'count_countdown'
)

// 倒计时进度：已消耗时间占总时长的百分比，随 now 实时更新
const timeCountdownProgress = computed(() => {
  if (!props.unit.target_time) return 0
  const targetTs = parseAppTime(props.unit.target_time)?.valueOf()
  const createdTs = parseAppTime(props.unit.created_at)?.valueOf()
  if (targetTs === undefined || createdTs === undefined) return 0
  const totalMs = targetTs - createdTs
  if (totalMs <= 0) return 100
  const elapsedMs = now.value - createdTs
  return Math.max(0, Math.min(100, (elapsedMs / totalMs) * 100))
})

const isCountType = computed(() =>
  props.unit.type === 'count_countdown' || props.unit.type === 'count_countup'
)

const displayTime = computed(() => {
  if (props.unit.type === 'time_countdown' && props.unit.remaining_seconds !== undefined) {
    const adjusted = props.unit.remaining_seconds - (Date.now() - now.value) / 1000
    if (adjusted <= 0) return '已超期'
    return formatDuration(adjusted)
  }
  if (props.unit.type === 'time_countup' && props.unit.elapsed_seconds !== undefined) {
    const adjusted = props.unit.elapsed_seconds + (Date.now() - now.value) / 1000
    return formatDuration(adjusted)
  }
  return '-'
})

const borderStyle = computed(() => {
  const color = props.unit.color || getDefaultColor()
  return { borderLeftColor: color, borderLeftWidth: '4px' }
})

function getDefaultColor(): string {
  if (props.unit.type === 'time_countdown' && props.unit.remaining_seconds !== undefined) {
    if (props.unit.remaining_seconds <= 0) return '#D32F2F'
    if (props.unit.remaining_seconds <= 7 * 86400) return '#F57C00'
  }
  return '#1565C0'
}

onMounted(() => {
  now.value = Date.now()
  timer = window.setInterval(() => { now.value = Date.now() }, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.unit-card {
  transition: all 0.2s ease;
}
.unit-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1) !important;
}
</style>
