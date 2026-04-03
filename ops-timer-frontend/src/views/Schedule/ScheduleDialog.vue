<template>
  <!-- 创建/编辑日程对话框 -->
  <v-dialog v-model="dialog" max-width="720" scrollable>
    <v-card>
      <v-card-title class="d-flex align-center pa-4 pb-0">
        <v-icon class="mr-2" color="primary">mdi-calendar-edit</v-icon>
        {{ isEdit ? '编辑日程' : '新建日程' }}
        <v-spacer />
        <v-btn icon="mdi-close" variant="text" @click="close" />
      </v-card-title>

      <v-card-text class="pt-3">
        <v-form ref="formRef" @submit.prevent="submit">
          <!-- 基本信息 -->
          <div class="text-caption text-medium-emphasis mb-2 font-weight-medium">基本信息</div>

          <!-- 标题 -->
          <v-text-field
            v-model="form.title"
            label="日程标题"
            variant="outlined"
            density="compact"
            :rules="[v => !!v || '标题不能为空']"
            class="mb-3"
          />

          <!-- 全天开关 -->
          <div class="d-flex align-center mb-3 gap-3">
            <v-switch
              v-model="form.all_day"
              label="全天事件"
              density="compact"
              color="primary"
              hide-details
              class="flex-grow-0"
              @update:model-value="onAllDayChange"            />
            <v-spacer />
            <!-- 颜色选择 -->
            <div class="d-flex align-center gap-2">
              <span class="text-body-2 text-medium-emphasis">颜色</span>
              <div class="d-flex gap-1">
                <div
                  v-for="color in colorOptions"
                  :key="color.value"
                  class="color-dot cursor-pointer"
                  :style="{ backgroundColor: color.value }"
                  :class="{ 'color-dot--active': form.color === color.value }"
                  @click="form.color = color.value"
                />
              </div>
            </div>
          </div>

          <!-- 时间选择 -->
          <v-row dense class="mb-3">
            <v-col cols="12" sm="6">
              <v-text-field
                v-model="form.start_time"
                :label="form.all_day ? '开始日期' : '开始时间'"
                :type="form.all_day ? 'date' : 'datetime-local'"
                variant="outlined"
                density="compact"
                :rules="[v => !!v || '请选择开始时间']"
              />
            </v-col>
            <v-col cols="12" sm="6">
              <v-text-field
                v-model="form.end_time"
                :label="form.all_day ? '结束日期' : '结束时间'"
                :type="form.all_day ? 'date' : 'datetime-local'"
                variant="outlined"
                density="compact"
                :rules="[v => !!v || '请选择结束时间', validateEndTime]"
              />
            </v-col>
          </v-row>

          <!-- 地点 -->
          <v-text-field
            v-model="form.location"
            label="地点（可选）"
            prepend-inner-icon="mdi-map-marker-outline"
            variant="outlined"
            density="compact"
            class="mb-3"
          />

          <!-- 描述 -->
          <v-textarea
            v-model="form.description"
            label="描述（可选）"
            variant="outlined"
            density="compact"
            rows="2"
            auto-grow
            class="mb-3"
          />

          <!-- 状态 + 重复 -->
          <v-row dense class="mb-3">
            <v-col cols="12" sm="6">
              <v-select
                v-model="form.status"
                label="状态"
                :items="statusOptions"
                variant="outlined"
                density="compact"
              />
            </v-col>
            <v-col cols="12" sm="6">
              <v-select
                v-model="form.recurrence_type"
                label="重复"
                :items="recurrenceOptions"
                variant="outlined"
                density="compact"
              />
            </v-col>
          </v-row>

          <!-- 重复结束日期 -->
          <v-text-field
            v-if="form.recurrence_type !== 'none'"
            v-model="form.recurrence_end"
            label="重复结束日期（可选）"
            type="date"
            variant="outlined"
            density="compact"
            class="mb-3"
          />

          <!-- 标签 -->
          <v-combobox
            v-model="form.tags"
            label="标签（可选）"
            multiple
            chips
            closable-chips
            variant="outlined"
            density="compact"
            class="mb-4"
          />

          <v-divider class="mb-4" />

          <!-- 关联资源 -->
          <div class="text-caption text-medium-emphasis mb-2 font-weight-medium d-flex align-center">
            <v-icon size="14" class="mr-1">mdi-link-variant</v-icon>
            关联资源
            <v-spacer />
            <v-btn
              v-if="isEdit"
              size="x-small"
              variant="tonal"
              color="primary"
              prepend-icon="mdi-plus"
              @click="showResourcePicker = true"
            >添加</v-btn>
          </div>

          <!-- 已关联资源列表 -->
          <div v-if="localResources.length" class="mb-3">
            <v-chip
              v-for="res in localResources"
              :key="res.id"
              :color="resourceColor(res)"
              variant="tonal"
              size="small"
              class="mr-2 mb-2"
              closable
              @click:close="removeResource(res)"
            >
              <v-icon start size="12">{{ resourceIcon(res.resource_type) }}</v-icon>
              {{ res.resource_title || res.resource_id }}
              <span v-if="res.note" class="ml-1 text-caption opacity-70">· {{ res.note }}</span>
            </v-chip>
          </div>
          <div v-else-if="isEdit" class="text-body-2 text-medium-emphasis mb-3">
            暂无关联资源，可关联项目、待办或计时单元
          </div>
          <div v-else class="text-body-2 text-medium-emphasis mb-3">
            保存日程后可添加关联资源
          </div>
        </v-form>
      </v-card-text>

      <v-card-actions class="px-4 pb-4">
        <v-btn
          v-if="isEdit"
          color="error"
          variant="text"
          prepend-icon="mdi-trash-can-outline"
          @click="deleteSchedule"
        >删除</v-btn>
        <v-spacer />
        <v-btn variant="text" @click="close">取消</v-btn>
        <v-btn
          color="primary"
          variant="flat"
          :loading="saving"
          @click="submit"
        >{{ isEdit ? '保存' : '创建' }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- 资源选择器对话框 -->
  <v-dialog v-model="showResourcePicker" max-width="480">
    <v-card>
      <v-card-title class="d-flex align-center pa-4 pb-0">
        <v-icon class="mr-2" color="primary">mdi-link-variant</v-icon>
        添加关联资源
        <v-spacer />
        <v-btn icon="mdi-close" variant="text" @click="showResourcePicker = false" />
      </v-card-title>

      <v-card-text class="pt-3">
        <v-btn-toggle
          v-model="resourcePickerType"
          mandatory
          color="primary"
          density="compact"
          variant="outlined"
          class="mb-4 w-100"
        >
          <v-btn value="project" size="small" class="flex-1">
            <v-icon start size="14">mdi-folder-outline</v-icon>项目
          </v-btn>
          <v-btn value="todo" size="small" class="flex-1">
            <v-icon start size="14">mdi-checkbox-marked-outline</v-icon>待办
          </v-btn>
          <v-btn value="unit" size="small" class="flex-1">
            <v-icon start size="14">mdi-timer</v-icon>计时单元
          </v-btn>
        </v-btn-toggle>

        <v-text-field
          v-model="resourceSearch"
          placeholder="搜索..."
          prepend-inner-icon="mdi-magnify"
          variant="outlined"
          density="compact"
          clearable
          class="mb-3"
          @update:model-value="onResourceSearch"
        />

        <v-list density="compact" lines="one" class="pa-0" max-height="240" style="overflow-y:auto">
          <v-list-item
            v-for="item in resourceSearchResults"
            :key="item.id"
            :title="item.title"
            :subtitle="item.subtitle"
            rounded="lg"
            class="mb-1"
            @click="selectResource(item)"
          >
            <template #prepend>
              <v-icon :color="item.color || 'primary'" size="18">
                {{ resourceIcon(resourcePickerType) }}
              </v-icon>
            </template>
          </v-list-item>
          <v-list-item v-if="resourceSearchResults.length === 0" class="text-center">
            <span class="text-body-2 text-medium-emphasis">无结果</span>
          </v-list-item>
        </v-list>

        <v-text-field
          v-model="resourceNote"
          label="备注（可选）"
          variant="outlined"
          density="compact"
          class="mt-3"
          placeholder="关于此关联的备注..."
        />
      </v-card-text>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { scheduleApi } from '@/api/schedules'
import { projectApi } from '@/api/projects'
import { todoApi } from '@/api/todos'
import { unitApi } from '@/api/units'
import type {
  Schedule,
  ScheduleResource,
  CreateScheduleRequest,
  UpdateScheduleRequest,
  ResourceType,
} from '@/types'

// ---- Props & Emits ----

interface Props {
  modelValue: boolean
  schedule?: Schedule | null
  initialStartTime?: string
  initialEndTime?: string
}
const props = withDefaults(defineProps<Props>(), {
  schedule: null,
  initialStartTime: '',
  initialEndTime: '',
})
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: [schedule: Schedule]
  deleted: [id: string]
}>()

// ---- State ----

const dialog = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})
const isEdit = computed(() => !!props.schedule?.id)
const formRef = ref()
const saving = ref(false)
const showResourcePicker = ref(false)
const resourcePickerType = ref<ResourceType>('project')
const resourceSearch = ref('')
const resourceNote = ref('')
const resourceSearchResults = ref<{ id: string; title: string; subtitle: string; color?: string }[]>([])
const localResources = ref<ScheduleResource[]>([])

interface FormData {
  title: string
  description: string
  start_time: string
  end_time: string
  all_day: boolean
  color: string
  location: string
  status: string
  recurrence_type: string
  recurrence_end: string
  tags: string[]
}

const defaultForm = (): FormData => ({
  title: '',
  description: '',
  start_time: '',
  end_time: '',
  all_day: false,
  color: '#1976D2',
  location: '',
  status: 'planned',
  recurrence_type: 'none',
  recurrence_end: '',
  tags: [],
})

const form = ref<FormData>(defaultForm())

// ---- Options ----

const colorOptions = [
  { value: '#1976D2' },
  { value: '#43A047' },
  { value: '#E53935' },
  { value: '#FB8C00' },
  { value: '#8E24AA' },
  { value: '#00ACC1' },
  { value: '#F4511E' },
  { value: '#616161' },
]

const statusOptions = [
  { title: '计划中', value: 'planned' },
  { title: '进行中', value: 'in_progress' },
  { title: '已完成', value: 'completed' },
  { title: '已取消', value: 'cancelled' },
]

const recurrenceOptions = [
  { title: '不重复', value: 'none' },
  { title: '每天', value: 'daily' },
  { title: '每周', value: 'weekly' },
  { title: '每月', value: 'monthly' },
  { title: '每年', value: 'yearly' },
]

// ---- Watchers ----

watch(() => props.modelValue, async (val) => {
  if (!val) return
  await nextTick()
  formRef.value?.resetValidation()

  if (props.schedule) {
    // 编辑模式：填充表单
    const s = props.schedule
    form.value = {
      title: s.title,
      description: s.description,
      start_time: formatToInput(s.start_time, s.all_day),
      end_time: formatToInput(s.end_time, s.all_day),
      all_day: s.all_day,
      color: s.color || '#1976D2',
      location: s.location,
      status: s.status,
      recurrence_type: s.recurrence_type,
      recurrence_end: s.recurrence_end ? s.recurrence_end.split('T')[0] : '',
      tags: s.tags || [],
    }
    localResources.value = [...(s.resources || [])]
  } else {
    // 新建模式
    form.value = {
      ...defaultForm(),
      start_time: props.initialStartTime || '',
      end_time: props.initialEndTime || '',
    }
    localResources.value = []
  }
})

watch(resourcePickerType, () => {
  resourceSearch.value = ''
  resourceSearchResults.value = []
  resourceNote.value = ''
})

// ---- Methods ----

function formatToInput(isoStr: string, allDay: boolean): string {
  if (!isoStr) return ''
  const d = new Date(isoStr)
  if (isNaN(d.getTime())) return ''
  if (allDay) {
    return d.toISOString().split('T')[0]
  }
  // datetime-local format: YYYY-MM-DDTHH:mm
  return d.toISOString().slice(0, 16)
}

function onAllDayChange(val: boolean | null) {
  const v = val ?? false
  // 重新转换时间格式
  if (form.value.start_time) {
    form.value.start_time = formatToInput(new Date(form.value.start_time).toISOString(), v)
  }
  if (form.value.end_time) {
    form.value.end_time = formatToInput(new Date(form.value.end_time).toISOString(), v)
  }
}

function validateEndTime(v: string) {
  if (!v || !form.value.start_time) return true
  return new Date(v) > new Date(form.value.start_time) || '结束时间必须晚于开始时间'
}

function toISOString(inputVal: string, allDay: boolean): string {
  if (!inputVal) return ''
  if (allDay) {
    return inputVal + 'T00:00:00'
  }
  // datetime-local: "YYYY-MM-DDTHH:mm" → add :00 for seconds
  return inputVal.length === 16 ? inputVal + ':00' : inputVal
}

async function submit() {
  const { valid } = await formRef.value?.validate()
  if (!valid) return

  saving.value = true
  try {
    const payload = {
      title: form.value.title,
      description: form.value.description,
      start_time: toISOString(form.value.start_time, form.value.all_day),
      end_time: toISOString(form.value.end_time, form.value.all_day),
      all_day: form.value.all_day,
      color: form.value.color,
      location: form.value.location,
      status: form.value.status as any,
      recurrence_type: form.value.recurrence_type as any,
      recurrence_end: form.value.recurrence_end || undefined,
      tags: form.value.tags,
    }

    let saved: Schedule
    if (isEdit.value && props.schedule?.id) {
      const res = await scheduleApi.update(props.schedule.id, payload as UpdateScheduleRequest)
      saved = res.data
    } else {
      const res = await scheduleApi.create(payload as CreateScheduleRequest)
      saved = res.data
    }
    emit('saved', saved)
    close()
  } catch (err: any) {
    console.error('保存日程失败', err)
  } finally {
    saving.value = false
  }
}

async function deleteSchedule() {
  if (!props.schedule?.id) return
  if (!confirm('确认删除此日程？')) return
  try {
    await scheduleApi.delete(props.schedule.id)
    emit('deleted', props.schedule.id)
    close()
  } catch (err) {
    console.error('删除日程失败', err)
  }
}

function close() {
  emit('update:modelValue', false)
}

// ---- 资源关联 ----

async function onResourceSearch() {
  const q = resourceSearch.value?.trim() || ''
  resourceSearchResults.value = []
  try {
    if (resourcePickerType.value === 'project') {
      const res = await projectApi.list({ status: 'active', page: 1, page_size: 20 })
      const items = (Array.isArray(res.data) ? res.data : (res.data as any)?.list || []) as any[]
      resourceSearchResults.value = items
        .filter((p: any) => !q || p.title.toLowerCase().includes(q.toLowerCase()))
        .slice(0, 10)
        .map((p: any) => ({ id: p.id, title: p.title, subtitle: p.status, color: p.color }))
    } else if (resourcePickerType.value === 'todo') {
      const res = await todoApi.list({ status: 'pending', page: 1, page_size: 20 })
      const items = (Array.isArray(res.data) ? res.data : (res.data as any)?.list || []) as any[]
      resourceSearchResults.value = items
        .filter((t: any) => !q || t.title.toLowerCase().includes(q.toLowerCase()))
        .slice(0, 10)
        .map((t: any) => ({ id: t.id, title: t.title, subtitle: t.status }))
    } else {
      const res = await unitApi.list({ status: 'active', page: 1, page_size: 20 })
      const items = (Array.isArray(res.data) ? res.data : (res.data as any)?.list || []) as any[]
      resourceSearchResults.value = items
        .filter((u: any) => !q || u.title.toLowerCase().includes(q.toLowerCase()))
        .slice(0, 10)
        .map((u: any) => ({ id: u.id, title: u.title, subtitle: u.type, color: u.color }))
    }
  } catch (err) {
    console.error('搜索资源失败', err)
  }
}

watch(showResourcePicker, async (val) => {
  if (val) {
    await onResourceSearch()
  }
})

async function selectResource(item: { id: string; title: string; subtitle: string; color?: string }) {
  if (!props.schedule?.id) return
  // 避免重复添加
  if (localResources.value.find(r => r.resource_id === item.id && r.resource_type === resourcePickerType.value)) {
    showResourcePicker.value = false
    return
  }
  try {
    const res = await scheduleApi.addResource(props.schedule.id, {
      resource_type: resourcePickerType.value,
      resource_id: item.id,
      note: resourceNote.value,
    })
    localResources.value.push(res.data)
    showResourcePicker.value = false
    resourceNote.value = ''
  } catch (err) {
    console.error('添加资源失败', err)
  }
}

async function removeResource(resource: ScheduleResource) {
  if (!props.schedule?.id) return
  try {
    await scheduleApi.removeResource(props.schedule.id, resource.id)
    localResources.value = localResources.value.filter(r => r.id !== resource.id)
  } catch (err) {
    console.error('移除资源失败', err)
  }
}

function resourceIcon(type: string): string {
  switch (type) {
    case 'project': return 'mdi-folder-outline'
    case 'todo': return 'mdi-checkbox-marked-outline'
    case 'unit': return 'mdi-timer'
    default: return 'mdi-link'
  }
}

function resourceColor(res: ScheduleResource): string {
  switch (res.resource_type) {
    case 'project': return 'blue'
    case 'todo': return 'orange'
    case 'unit': return 'green'
    default: return 'grey'
  }
}
</script>

<style scoped>
.color-dot {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 2px solid transparent;
  transition: border-color 0.15s;
}
.color-dot--active {
  border-color: rgba(0, 0, 0, 0.4);
  transform: scale(1.15);
}
.gap-1 { gap: 4px; }
.gap-2 { gap: 8px; }
.gap-3 { gap: 12px; }
.w-100 { width: 100%; }
.flex-1 { flex: 1; }
</style>
