<template>
  <div v-if="unit">
    <div class="d-flex align-center mb-4">
      <v-btn icon="mdi-arrow-left" variant="text" @click="$router.back()" />
      <h2 class="text-h5 font-weight-bold ml-2">{{ unit.title }}</h2>
      <v-spacer />
      <v-chip :color="getStatusColor(unit.status)" class="mr-2">{{ getStatusLabel(unit.status) }}</v-chip>
      <v-chip :color="getPriorityColor(unit.priority)">{{ getPriorityLabel(unit.priority) }}</v-chip>
    </div>

    <v-row>
      <v-col cols="12" md="8">
        <v-card class="rounded-lg mb-4">
          <v-card-text>
            <div class="d-flex align-center mb-4">
              <v-icon :color="getTimeColor(unit.remaining_seconds, isCountdown)" size="28" class="mr-2">
                {{ getUnitTypeIcon(unit.type) }}
              </v-icon>
              <span class="text-body-1">{{ getUnitTypeLabel(unit.type) }}</span>
            </div>

            <!-- Main display -->
            <div v-if="unit.type === 'time_countdown'" class="text-center py-6">
              <div class="text-h3 font-weight-bold" :class="`text-${getTimeColor(displayCountdown, true)}`">
                {{ displayCountdown > 0 ? formatDuration(displayCountdown) : '已超期' }}
              </div>
              <div class="text-body-2 text-medium-emphasis mt-2">目标时间: {{ formatDateTime(unit.target_time) }}</div>
              <v-progress-linear
                :model-value="timeCountdownProgress"
                :color="getTimeColor(displayCountdown, true)"
                bg-color="surface-variant"
                height="10"
                rounded
                class="mt-4 mx-auto"
                style="max-width: 480px"
              />
              <div class="d-flex justify-space-between text-caption text-medium-emphasis mt-1 mx-auto" style="max-width: 480px">
                <span>创建: {{ formatDateTime(unit.created_at) }}</span>
                <span class="font-weight-medium">{{ Math.round(timeCountdownProgress) }}% 已过</span>
                <span>截止: {{ formatDateTime(unit.target_time) }}</span>
              </div>
            </div>

            <div v-else-if="unit.type === 'time_countup'" class="text-center py-6">
              <div class="text-h3 font-weight-bold text-primary">
                {{ unit.elapsed_seconds !== undefined ? formatDuration(displayCountup) : '-' }}
              </div>
              <div class="text-body-2 text-medium-emphasis mt-2">开始时间: {{ formatDateTime(unit.start_time) }}</div>
            </div>

            <div v-else-if="unit.type === 'count_countdown'" class="text-center py-6">
              <div class="text-h3 font-weight-bold">{{ unit.current_value || 0 }} / {{ unit.target_value }}</div>
              <div class="text-body-2 text-medium-emphasis">{{ unit.unit_label }}</div>
              <v-progress-linear :model-value="unit.progress || 0" color="primary" height="12" rounded class="mt-4 mx-auto" style="max-width: 400px" />
              <div class="text-body-2 mt-2">{{ Math.round(unit.progress || 0) }}% 完成</div>
              <div class="mt-4">
                <v-btn color="error" variant="tonal" class="mr-2" @click="handleStep('down')">
                  <v-icon>mdi-minus</v-icon> {{ unit.step || 1 }}
                </v-btn>
                <v-btn color="success" variant="tonal" @click="handleStep('up')">
                  <v-icon>mdi-plus</v-icon> {{ unit.step || 1 }}
                </v-btn>
              </div>
            </div>

            <div v-else class="text-center py-6">
              <div class="text-h3 font-weight-bold text-primary">{{ unit.current_value || 0 }} <span class="text-h6">{{ unit.unit_label }}</span></div>
              <div class="mt-4">
                <v-btn color="error" variant="tonal" class="mr-2" @click="handleStep('down')">
                  <v-icon>mdi-minus</v-icon> {{ unit.step || 1 }}
                </v-btn>
                <v-btn color="success" variant="tonal" @click="handleStep('up')">
                  <v-icon>mdi-plus</v-icon> {{ unit.step || 1 }}
                </v-btn>
              </div>
            </div>
          </v-card-text>
        </v-card>

        <v-card v-if="unit.description" class="rounded-lg mb-4">
          <v-card-title>描述</v-card-title>
          <v-divider />
          <v-card-text class="md-preview-wrap">
            <MdPreview :model-value="unit.description" preview-only />
          </v-card-text>
        </v-card>

        <!-- Logs for count types -->
        <v-card v-if="isCountType" class="rounded-lg">
          <v-card-title>操作记录</v-card-title>
          <v-divider />
          <v-table v-if="logs.length > 0">
            <thead>
              <tr>
                <th>时间</th>
                <th>变更</th>
                <th>操作前</th>
                <th>操作后</th>
                <th>备注</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="log in logs" :key="log.id">
                <td>{{ formatDateTime(log.operated_at) }}</td>
                <td>
                  <v-chip :color="log.delta > 0 ? 'success' : 'error'" size="x-small">
                    {{ log.delta > 0 ? '+' : '' }}{{ log.delta }}
                  </v-chip>
                </td>
                <td>{{ log.value_before }}</td>
                <td>{{ log.value_after }}</td>
                <td>{{ log.note || '-' }}</td>
              </tr>
            </tbody>
          </v-table>
          <v-card-text v-else class="text-center text-medium-emphasis">暂无操作记录</v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" md="4">
        <v-card class="rounded-lg mb-4">
          <v-card-title>信息</v-card-title>
          <v-divider />
          <v-list density="compact">
            <v-list-item>
              <v-list-item-title class="text-caption text-medium-emphasis">状态</v-list-item-title>
              <v-list-item-subtitle>
                <v-select
                  v-model="unit.status"
                  :items="statusItems"
                  density="compact"
                  hide-details
                  variant="outlined"
                  @update:model-value="updateStatus"
                />
              </v-list-item-subtitle>
            </v-list-item>
            <v-list-item>
              <v-list-item-title class="text-caption text-medium-emphasis">创建时间</v-list-item-title>
              <v-list-item-subtitle>{{ formatDateTime(unit.created_at) }}</v-list-item-subtitle>
            </v-list-item>
            <v-list-item>
              <v-list-item-title class="text-caption text-medium-emphasis">更新时间</v-list-item-title>
              <v-list-item-subtitle>{{ formatDateTime(unit.updated_at) }}</v-list-item-subtitle>
            </v-list-item>
          </v-list>
        </v-card>

        <v-card class="rounded-lg" v-if="unit.tags && unit.tags.length">
          <v-card-title>标签</v-card-title>
          <v-divider />
          <v-card-text>
            <v-chip v-for="tag in unit.tags" :key="tag" class="mr-1 mb-1" size="small" variant="outlined">
              {{ tag }}
            </v-chip>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
  <div v-else class="text-center py-12">
    <v-progress-circular indeterminate />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/preview.css'
import type { Unit, UnitLog } from '@/types'
import { unitApi } from '@/api/units'
import {
  formatDuration, formatDateTime, getTimeColor, getStatusColor, getStatusLabel, parseAppTime,
  getPriorityColor, getPriorityLabel, getUnitTypeLabel, getUnitTypeIcon,
} from '@/utils/time'

const route = useRoute()
const unit = ref<Unit | null>(null)
const logs = ref<UnitLog[]>([])
const now = ref(Date.now())
let ticker: number | null = null

const isCountdown = computed(() =>
  unit.value?.type === 'time_countdown' || unit.value?.type === 'count_countdown'
)
const isCountType = computed(() =>
  unit.value?.type === 'count_countdown' || unit.value?.type === 'count_countup'
)

// 实时修正：倒计时（从 API 快照向前推算已过去的秒数）
const displayCountdown = computed(() => {
  if (!unit.value || unit.value.remaining_seconds === undefined) return 0
  return unit.value.remaining_seconds - (now.value - fetchedAt.value) / 1000
})

// 实时修正：正计时（从 API 快照向后累加）
const displayCountup = computed(() => {
  if (!unit.value || unit.value.elapsed_seconds === undefined) return 0
  return unit.value.elapsed_seconds + (now.value - fetchedAt.value) / 1000
})

// 倒计时进度：已消耗时间占总时长的百分比
const timeCountdownProgress = computed(() => {
  if (!unit.value?.target_time) return 0
  const targetTs = parseAppTime(unit.value.target_time)?.valueOf()
  const createdTs = parseAppTime(unit.value.created_at)?.valueOf()
  if (targetTs === undefined || createdTs === undefined) return 0
  const totalMs = targetTs - createdTs
  if (totalMs <= 0) return 100
  const elapsedMs = now.value - createdTs
  return Math.max(0, Math.min(100, (elapsedMs / totalMs) * 100))
})

// 记录最近一次拉取数据的时刻
const fetchedAt = ref(Date.now())

const statusItems = [
  { title: '激活', value: 'active' },
  { title: '暂停', value: 'paused' },
  { title: '已完成', value: 'completed' },
  { title: '已归档', value: 'archived' },
]

async function fetchUnit() {
  const resp = await unitApi.getById(route.params.id as string)
  unit.value = resp.data
  fetchedAt.value = Date.now()
}

async function fetchLogs() {
  if (!isCountType.value) return
  const resp = await unitApi.getLogs(route.params.id as string, { page_size: 50 })
  logs.value = resp.data || []
}

async function handleStep(direction: 'up' | 'down') {
  await unitApi.step(route.params.id as string, direction)
  fetchUnit()
  fetchLogs()
}

async function updateStatus(status: string) {
  await unitApi.updateStatus(route.params.id as string, status)
}

onMounted(async () => {
  await fetchUnit()
  fetchLogs()
  ticker = window.setInterval(() => { now.value = Date.now() }, 1000)
})

onUnmounted(() => {
  if (ticker) clearInterval(ticker)
})
</script>

<style>
/* 让 MdPreview 的背景透明，字号适配 Vuetify 卡片 */
.md-preview-wrap .md-editor-preview-wrapper {
  padding: 0;
  background: transparent;
}
.md-preview-wrap .md-editor-preview {
  font-size: 0.875rem;
  line-height: 1.6;
  background: transparent;
}
</style>
