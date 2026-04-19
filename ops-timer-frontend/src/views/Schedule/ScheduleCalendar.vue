<template>
  <div class="schedule-calendar-page">
    <!-- 页面标题栏 -->
    <div class="d-flex align-center mb-4">
      <div>
        <div class="text-h6 font-weight-medium">日程管理</div>
        <div class="text-body-2 text-medium-emphasis">管理你的日程安排，关联项目与待办</div>
      </div>
      <v-spacer />
      <v-btn
        color="primary"
        variant="flat"
        prepend-icon="mdi-plus"
        @click="openCreateDialog()"
      >
        新建日程
      </v-btn>
    </div>

    <!-- 月份导航 -->
    <v-card class="mb-4 pa-2 d-flex align-center" elevation="1" rounded="lg">
      <v-btn icon="mdi-chevron-left" variant="text" @click="prevMonth" />
      <div class="text-subtitle-1 font-weight-medium mx-4">{{ currentMonthLabel }}</div>
      <v-btn icon="mdi-chevron-right" variant="text" @click="nextMonth" />
      <v-spacer />
      <v-btn variant="text" @click="goToday">回到今天</v-btn>
    </v-card>

    <!-- 日程列表 -->
    <v-card elevation="1" rounded="lg" class="list-container">
      <div v-if="loading" class="pa-8 text-center">
        <v-progress-circular indeterminate color="primary"></v-progress-circular>
      </div>
      <div v-else-if="groupedSchedules.length === 0" class="pa-8 text-center text-medium-emphasis">
        暂无日程安排
      </div>
      <v-list v-else bg-color="transparent" class="pa-4">
        <template v-for="group in groupedSchedules" :key="group.date">
          <div class="text-subtitle-2 font-weight-bold text-primary mt-4 mb-2 d-flex align-center">
            <v-icon size="18" class="mr-1">mdi-calendar-today</v-icon>
            {{ formatGroupDate(group.date) }}
          </div>
          
          <v-card
            v-for="schedule in group.schedules"
            :key="schedule.id"
            variant="outlined"
            class="mb-3 schedule-card cursor-pointer"
            hover
            @click="selectSchedule(schedule)"
          >
            <div class="d-flex" style="position: relative">
              <div class="color-indicator" :style="{ backgroundColor: schedule.color || '#1976D2' }"></div>
              
              <div class="pa-3 flex-1 min-width-0">
                <div class="d-flex align-center mb-1 flex-wrap">
                  <div class="text-subtitle-1 font-weight-medium text-truncate mr-2">{{ schedule.title }}</div>
                  <v-chip :color="statusColor(schedule.status)" size="x-small" variant="tonal" class="mr-2">
                    {{ statusLabel(schedule.status) }}
                  </v-chip>
                  <v-chip
                    v-if="schedule.recurrence_type !== 'none'"
                    size="x-small" variant="tonal" color="purple" class="mr-2"
                  >
                    <v-icon start size="10">mdi-repeat</v-icon>
                    {{ recurrenceLabel(schedule.recurrence_type) }}
                  </v-chip>
                  <v-spacer />
                  <div class="text-body-2 text-medium-emphasis d-flex align-center">
                    <v-icon size="14" class="mr-1">mdi-clock-outline</v-icon>
                    {{ formatScheduleTimeShort(schedule) }}
                  </div>
                </div>
                
                <div v-if="schedule.location || schedule.description" class="d-flex align-center mt-1 flex-wrap gap-3">
                  <div v-if="schedule.location" class="text-caption text-medium-emphasis d-flex align-center">
                    <v-icon size="14" class="mr-1">mdi-map-marker-outline</v-icon>
                    {{ schedule.location }}
                  </div>
                  <div v-if="schedule.description" class="text-caption text-medium-emphasis text-truncate" style="max-width: 300px;">
                    {{ schedule.description }}
                  </div>
                </div>
                
                <div class="d-flex align-center mt-2" v-if="schedule.tags?.length || schedule.resources?.length">
                  <div class="d-flex gap-1 mr-4" v-if="schedule.tags?.length">
                    <v-chip v-for="tag in schedule.tags" :key="tag" size="x-small" variant="outlined">
                      {{ tag }}
                    </v-chip>
                  </div>
                  <div class="d-flex align-center" v-if="schedule.resources?.length">
                    <v-icon size="14" color="medium-emphasis" class="mr-1">mdi-link-variant</v-icon>
                    <span class="text-caption text-medium-emphasis">{{ schedule.resources.length }} 个关联资源</span>
                  </div>
                </div>
              </div>
            </div>
          </v-card>
        </template>
      </v-list>
    </v-card>

    <!-- 日程详情侧边面板 -->
    <v-navigation-drawer
      v-model="detailPanel"
      location="right"
      :width="360"
      temporary
    >
      <div v-if="selectedSchedule" class="pa-4">
        <div class="d-flex align-center mb-3">
          <v-icon class="mr-2" :style="{ color: selectedSchedule.color || '#1976D2' }">
            mdi-calendar-check
          </v-icon>
          <div class="text-subtitle-1 font-weight-medium flex-1">{{ selectedSchedule.title }}</div>
          <v-btn icon="mdi-pencil" size="small" variant="text" @click="openEditDialog(selectedSchedule)" />
          <v-btn icon="mdi-close" size="small" variant="text" @click="detailPanel = false" />
        </div>

        <div class="d-flex align-center mb-2">
          <v-icon size="16" class="mr-2 text-medium-emphasis">mdi-clock-outline</v-icon>
          <span class="text-body-2">{{ formatScheduleTime(selectedSchedule) }}</span>
        </div>

        <div v-if="selectedSchedule.location" class="d-flex align-center mb-2">
          <v-icon size="16" class="mr-2 text-medium-emphasis">mdi-map-marker-outline</v-icon>
          <span class="text-body-2">{{ selectedSchedule.location }}</span>
        </div>

        <div class="d-flex align-center mb-3">
          <v-icon size="16" class="mr-2 text-medium-emphasis">mdi-information-outline</v-icon>
          <v-chip :color="statusColor(selectedSchedule.status)" size="x-small" variant="tonal">
            {{ statusLabel(selectedSchedule.status) }}
          </v-chip>
          <v-chip
            v-if="selectedSchedule.recurrence_type !== 'none'"
            size="x-small" variant="tonal" color="purple" class="ml-2"
          >
            <v-icon start size="10">mdi-repeat</v-icon>
            {{ recurrenceLabel(selectedSchedule.recurrence_type) }}
          </v-chip>
        </div>

        <div v-if="selectedSchedule.description" class="mb-3">
          <div class="text-caption text-medium-emphasis mb-1">描述</div>
          <div class="text-body-2 pa-2 bg-surface-variant rounded">
            {{ selectedSchedule.description }}
          </div>
        </div>

        <div v-if="selectedSchedule.tags?.length" class="mb-3">
          <div class="text-caption text-medium-emphasis mb-1">标签</div>
          <div class="d-flex flex-wrap gap-1">
            <v-chip v-for="tag in selectedSchedule.tags" :key="tag" size="x-small" variant="outlined">
              {{ tag }}
            </v-chip>
          </div>
        </div>

        <div v-if="selectedSchedule.resources?.length">
          <div class="text-caption text-medium-emphasis mb-2">关联资源</div>
          <div
            v-for="res in selectedSchedule.resources"
            :key="res.id"
            class="resource-item d-flex align-center pa-2 rounded mb-2"
          >
            <v-icon :color="resourceTypeColor(res.resource_type)" size="16" class="mr-2">
              {{ resourceTypeIcon(res.resource_type) }}
            </v-icon>
            <div class="flex-1 min-width-0">
              <div class="text-body-2 text-truncate">{{ res.resource_title || res.resource_id }}</div>
              <div v-if="res.resource_status" class="text-caption text-medium-emphasis">
                {{ res.resource_status }}
              </div>
            </div>
            <v-chip size="x-small" variant="text" :color="resourceTypeColor(res.resource_type)">
              {{ resourceTypeLabel(res.resource_type) }}
            </v-chip>
          </div>
        </div>
        <div v-else class="text-body-2 text-medium-emphasis">暂无关联资源</div>

        <v-divider class="my-4" />
        <v-btn variant="tonal" color="primary" size="small" block @click="openEditDialog(selectedSchedule)">
          <v-icon start>mdi-pencil</v-icon>编辑日程
        </v-btn>
      </div>
    </v-navigation-drawer>

    <!-- 创建/编辑对话框 -->
    <ScheduleDialog
      v-model="dialogOpen"
      :schedule="editingSchedule"
      :initial-start-time="newEventStartTime"
      :initial-end-time="newEventEndTime"
      @saved="onScheduleSaved"
      @deleted="onScheduleDeleted"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import ScheduleDialog from './ScheduleDialog.vue'
import { scheduleApi } from '@/api/schedules'
import { APP_TIMEZONE, dayjs, parseAppTime } from '@/utils/time'
import type { Schedule } from '@/types'

// ---- 状态 ----

const loading = ref(false)
const dialogOpen = ref(false)
const editingSchedule = ref<Schedule | null>(null)
const newEventStartTime = ref('')
const newEventEndTime = ref('')
const detailPanel = ref(false)
const selectedSchedule = ref<Schedule | null>(null)

const schedules = ref<Schedule[]>([])

// 当前查阅的月份，默认本月 1 号
const currentDate = ref(dayjs.tz(APP_TIMEZONE).startOf('month'))

const currentMonthLabel = computed(() => currentDate.value.format('YYYY年 MM月'))

const groupedSchedules = computed(() => {
  const map = new Map<string, Schedule[]>()
  for (const s of schedules.value) {
    const d = parseAppTime(s.start_time)?.format('YYYY-MM-DD')
    if (!d) continue
    if (!map.has(d)) map.set(d, [])
    map.get(d)!.push(s)
  }
  return Array.from(map.entries())
    .sort((a, b) => a[0].localeCompare(b[0]))
    .map(([date, list]) => ({
      date,
      schedules: list.sort((a, b) => a.start_time.localeCompare(b.start_time))
    }))
})

// ---- 导航操作 ----

function prevMonth() {
  currentDate.value = currentDate.value.subtract(1, 'month')
  fetchSchedules()
}

function nextMonth() {
  currentDate.value = currentDate.value.add(1, 'month')
  fetchSchedules()
}

function goToday() {
  currentDate.value = dayjs.tz(APP_TIMEZONE).startOf('month')
  fetchSchedules()
}

// ---- 加载数据 ----

async function fetchSchedules() {
  loading.value = true
  try {
    // 拉取当前月份前后各延伸几天，以防止边界漏算（直接取当月起始和结束）
    const start_date = currentDate.value.startOf('month').format('YYYY-MM-DD')
    const end_date = currentDate.value.endOf('month').format('YYYY-MM-DD')
    
    const res = await scheduleApi.list({ start_date, end_date, page_size: 500 })
    schedules.value = Array.isArray(res.data) ? res.data : []
  } catch (err) {
    console.error('加载日程失败', err)
  } finally {
    loading.value = false
  }
}

// ---- 对话框操作 ----

function openCreateDialog() {
  editingSchedule.value = null
  const now = dayjs.tz(APP_TIMEZONE)
  // 默认创建一个1小时的日程
  newEventStartTime.value = now.startOf('hour').format('YYYY-MM-DDTHH:mm')
  newEventEndTime.value = now.startOf('hour').add(1, 'hour').format('YYYY-MM-DDTHH:mm')
  dialogOpen.value = true
}

function openEditDialog(schedule: Schedule) {
  editingSchedule.value = schedule
  detailPanel.value = false
  dialogOpen.value = true
}

function selectSchedule(schedule: Schedule) {
  selectedSchedule.value = schedule
  detailPanel.value = true
}

function onScheduleSaved() {
  fetchSchedules()
  if (selectedSchedule.value) {
    // 重新从列表中找到更新后的内容并刷新 detailPanel
    // (在真实场景可直接通过 API 返回的结果覆盖 selectedSchedule，但列表刷新更稳妥)
    detailPanel.value = false
  }
}

function onScheduleDeleted() {
  detailPanel.value = false
  fetchSchedules()
}

// ---- 格式化辅助 ----

function formatGroupDate(dateStr: string) {
  const d = parseAppTime(dateStr)
  if (!d) return dateStr
  const isToday = d.isSame(dayjs.tz(APP_TIMEZONE), 'day')
  const weekday = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][d.day()]
  return `${d.format('M月D日')} ${weekday}${isToday ? ' (今天)' : ''}`
}

function formatScheduleTimeShort(s: Schedule): string {
  if (s.all_day) return '全天'
  return `${parseAppTime(s.start_time)?.format('HH:mm')} - ${parseAppTime(s.end_time)?.format('HH:mm')}`
}

function formatScheduleTime(s: Schedule): string {
  if (s.all_day) {
    const start = parseAppTime(s.start_time)?.format('YYYY年M月D日')
    const end = parseAppTime(s.end_time)?.format('YYYY年M月D日')
    return start === end ? `${start} · 全天` : `${start} ~ ${end} · 全天`
  }
  const start = parseAppTime(s.start_time)
  const end = parseAppTime(s.end_time)
  if (!start || !end) return ''
  if (start.format('YYYY-MM-DD') === end.format('YYYY-MM-DD')) {
    return `${start.format('YYYY年M月D日 HH:mm')} ~ ${end.format('HH:mm')}`
  }
  return `${start.format('M月D日 HH:mm')} ~ ${end.format('M月D日 HH:mm')}`
}

function statusColor(s: string) {
  return { planned: 'blue', in_progress: 'green', completed: 'grey', cancelled: 'red' }[s] ?? 'grey'
}

function statusLabel(s: string) {
  return { planned: '计划中', in_progress: '进行中', completed: '已完成', cancelled: '已取消' }[s] ?? s
}

function recurrenceLabel(t: string) {
  return { daily: '每天', weekly: '每周', monthly: '每月', yearly: '每年' }[t] ?? t
}

function resourceTypeIcon(t: string) {
  return { project: 'mdi-folder-outline', todo: 'mdi-checkbox-marked-outline', unit: 'mdi-timer' }[t] ?? 'mdi-link'
}

function resourceTypeColor(t: string) {
  return { project: 'blue', todo: 'orange', unit: 'green' }[t] ?? 'grey'
}

function resourceTypeLabel(t: string) {
  return { project: '项目', todo: '待办', unit: '计时单元' }[t] ?? t
}

// 初始化
onMounted(() => {
  fetchSchedules()
})
</script>

<style scoped>
.schedule-calendar-page {
  max-width: 1000px;
  margin: 0 auto;
}

.list-container {
  min-height: 600px;
  background: rgb(var(--v-theme-surface));
}

.schedule-card {
  transition: transform 0.2s, box-shadow 0.2s;
}

.schedule-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.1) !important;
}

.color-indicator {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 6px;
}

.resource-item {
  background: rgba(var(--v-theme-primary), 0.05);
  border: 1px solid rgba(var(--v-theme-outline), 0.2);
}

.min-width-0 {
  min-width: 0;
}

.flex-1 {
  flex: 1;
}

.gap-1 { gap: 4px; }
.gap-3 { gap: 12px; }
</style>
