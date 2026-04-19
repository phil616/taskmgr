<template>
  <div>
    <div class="d-flex align-center mb-4">
      <h2 class="text-h5 font-weight-bold">计时单元</h2>
      <v-spacer />
      <v-btn-toggle v-model="viewMode" variant="outlined" density="compact" class="mr-2">
        <v-btn value="card" icon="mdi-view-grid" size="small" />
        <v-btn value="list" icon="mdi-view-list" size="small" />
      </v-btn-toggle>
      <v-btn color="primary" prepend-icon="mdi-plus" @click="showCreateDialog = true">
        创建单元
      </v-btn>
    </div>

    <v-card class="rounded-lg mb-4 pa-3">
      <v-row dense>
        <v-col cols="12" sm="4" md="3">
          <v-text-field
            v-model="filters.q"
            placeholder="搜索标题..."
            prepend-inner-icon="mdi-magnify"
            density="compact"
            hide-details
            clearable
            @update:model-value="debouncedFetch"
          />
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-select
            v-model="filters.type"
            :items="typeOptions"
            label="类型"
            density="compact"
            hide-details
            clearable
            @update:model-value="fetchUnits"
          />
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-select
            v-model="filters.status"
            :items="statusOptions"
            label="状态"
            density="compact"
            hide-details
            clearable
            @update:model-value="fetchUnits"
          />
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-select
            v-model="filters.priority"
            :items="priorityOptions"
            label="优先级"
            density="compact"
            hide-details
            clearable
            @update:model-value="fetchUnits"
          />
        </v-col>
        <v-col cols="6" sm="4" md="3">
          <v-select
            v-model="filters.sort_by"
            :items="sortOptions"
            label="排序"
            density="compact"
            hide-details
            @update:model-value="fetchUnits"
          />
        </v-col>
      </v-row>
    </v-card>

    <v-row v-if="viewMode === 'card'">
      <v-col v-for="unit in units" :key="unit.id" cols="12" sm="6" md="4" lg="3">
        <unit-card :unit="unit" @step="handleStep" />
      </v-col>
    </v-row>

    <v-card v-else class="rounded-lg">
      <v-table>
        <thead>
          <tr>
            <th>标题</th>
            <th>类型</th>
            <th>状态</th>
            <th>优先级</th>
            <th>详情</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="unit in units" :key="unit.id">
            <td>
              <router-link :to="`/units/${unit.id}`" class="text-decoration-none">
                {{ unit.title }}
              </router-link>
            </td>
            <td><v-chip size="x-small" variant="tonal">{{ getUnitTypeLabel(unit.type) }}</v-chip></td>
            <td><v-chip size="x-small" :color="getStatusColor(unit.status)" variant="flat">{{ getStatusLabel(unit.status) }}</v-chip></td>
            <td><v-chip size="x-small" :color="getPriorityColor(unit.priority)" variant="tonal">{{ getPriorityLabel(unit.priority) }}</v-chip></td>
            <td class="text-body-2">
              <span v-if="unit.type === 'time_countdown'">{{ unit.remaining_seconds && unit.remaining_seconds > 0 ? formatDuration(unit.remaining_seconds) : '已超期' }}</span>
              <span v-else-if="unit.type === 'time_countup' && unit.elapsed_seconds">{{ formatDuration(unit.elapsed_seconds) }}</span>
              <span v-else-if="unit.type === 'count_countdown'">{{ unit.current_value || 0 }}/{{ unit.target_value }} {{ unit.unit_label }}</span>
              <span v-else>{{ unit.current_value || 0 }} {{ unit.unit_label }}</span>
            </td>
            <td>
              <v-btn icon="mdi-pencil" size="small" variant="text" @click="editUnit(unit)" />
              <v-btn icon="mdi-delete" size="small" variant="text" color="error" @click="deleteUnit(unit.id)" />
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card>

    <div v-if="units.length === 0 && !loading" class="text-center py-12">
      <v-icon size="64" color="grey-lighten-1">mdi-timer-off-outline</v-icon>
      <p class="text-body-1 text-medium-emphasis mt-4">暂无计时单元</p>
      <v-btn color="primary" class="mt-2" @click="showCreateDialog = true">创建第一个单元</v-btn>
    </div>

    <div class="d-flex justify-center mt-4" v-if="totalPages > 1">
      <v-pagination v-model="page" :length="totalPages" @update:model-value="fetchUnits" />
    </div>

    <!-- Create/Edit Dialog -->
    <v-dialog v-model="showCreateDialog" max-width="700" persistent>
      <v-card class="rounded-lg">
        <v-card-title>{{ editingUnit ? '编辑单元' : '创建计时单元' }}</v-card-title>
        <v-divider />
        <v-card-text>
          <v-form ref="formRef">
            <v-text-field v-model="form.title" label="标题" :rules="[v => !!v || '必填']" />

            <v-row dense>
              <v-col cols="6">
                <v-select
                  v-model="timerCategory"
                  :items="categoryOptions"
                  label="计时类别"
                  :disabled="!!editingUnit"
                  :rules="[v => !!v || '必填']"
                />
              </v-col>
              <v-col cols="6">
                <v-btn-toggle
                  v-model="timerDirection"
                  mandatory
                  variant="outlined"
                  density="comfortable"
                  color="primary"
                  class="w-100"
                  style="height:56px"
                >
                  <v-btn value="countdown" class="flex-grow-1">
                    <v-icon start size="18">mdi-timer-sand</v-icon>
                    倒计时
                  </v-btn>
                  <v-btn value="countup" class="flex-grow-1">
                    <v-icon start size="18">mdi-timer-outline</v-icon>
                    正计时
                  </v-btn>
                </v-btn-toggle>
              </v-col>
            </v-row>

            <v-row dense>
              <v-col cols="6">
                <v-select v-model="form.priority" :items="priorityOptions" label="优先级" />
              </v-col>
              <v-col cols="6">
                <v-select v-model="form.status" :items="statusOptions" label="状态" />
              </v-col>
            </v-row>

            <v-text-field v-model="form.color" label="颜色 (HEX)" placeholder="#1565C0" />

            <v-combobox v-model="form.tags" label="标签" multiple chips closable-chips />

            <v-textarea v-model="form.description" label="描述" rows="3" />

            <v-select
              v-model="form.project_id"
              :items="[{ title: '— 不归属任何项目 —', value: null }, ...projects.map(p => ({ title: p.title, value: p.id }))]"
              label="所属项目（可选）"
              clearable
              hide-details
              class="mb-4"
            />

            <!-- Time fields -->
            <template v-if="form.type === 'time_countdown'">
              <v-text-field v-model="form.target_time" label="目标时间" type="datetime-local" />
            </template>
            <template v-else-if="form.type === 'time_countup'">
              <v-text-field v-model="form.start_time" label="开始时间" type="datetime-local" />
            </template>

            <!-- Count fields -->
            <template v-if="form.type === 'count_countdown' || form.type === 'count_countup'">
              <v-row dense>
                <v-col cols="4">
                  <v-text-field v-model.number="form.current_value" label="当前值" type="number" />
                </v-col>
                <v-col cols="4" v-if="form.type === 'count_countdown'">
                  <v-text-field v-model.number="form.target_value" label="目标值" type="number" />
                </v-col>
                <v-col cols="4">
                  <v-text-field v-model.number="form.step" label="步长" type="number" />
                </v-col>
              </v-row>
              <v-text-field v-model="form.unit_label" label="单位标签" placeholder="次、台、GB" />
            </template>

            <!-- 通知阈值 -->
            <v-divider class="my-3" />
            <div class="text-body-2 font-weight-medium mb-2 d-flex align-center ga-1">
              <v-icon size="16" color="warning">mdi-bell-ring-outline</v-icon>
              通知阈值
              <span class="text-caption text-medium-emphasis ml-1">（需在设置中配置邮箱）</span>
            </div>

            <template v-if="form.type === 'time_countdown'">
              <v-combobox
                v-model="form.remind_before_days"
                label="剩余 ≤ N 天时通知"
                hint="输入天数后按 Enter 确认，可设置多个阈值，如 7、3、1"
                persistent-hint
                multiple
                chips
                closable-chips
                :items="[30, 14, 7, 3, 1]"
                hide-no-data
                class="mb-2"
              />
            </template>

            <template v-else-if="form.type === 'time_countup'">
              <v-combobox
                v-model="form.remind_after_days"
                label="持续 ≥ N 天时通知"
                hint="输入天数后按 Enter 确认，可设置多个阈值，如 30、90、180"
                persistent-hint
                multiple
                chips
                closable-chips
                :items="[30, 60, 90, 180, 365]"
                hide-no-data
                class="mb-2"
              />
            </template>

            <template v-else-if="form.type === 'count_countdown' || form.type === 'count_countup'">
              <v-combobox
                v-model="form.remind_on_values"
                :label="form.type === 'count_countdown' ? '当前值 ≥ N 时通知' : '当前值 ≥ N 时通知'"
                :hint="`输入数值后按 Enter 确认，单位：${form.unit_label || '—'}`"
                persistent-hint
                multiple
                chips
                closable-chips
                hide-no-data
                class="mb-2"
              />
            </template>
          </v-form>
        </v-card-text>
        <v-divider />
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="closeDialog">取消</v-btn>
          <v-btn color="primary" :loading="saving" @click="saveUnit">
            {{ editingUnit ? '更新' : '创建' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue'
import type { Unit, Project } from '@/types'
import { unitApi } from '@/api/units'
import { projectApi } from '@/api/projects'
import UnitCard from '@/components/UnitCard.vue'
import {
  getUnitTypeLabel, getStatusColor, getStatusLabel,
  getPriorityColor, getPriorityLabel, formatDuration,
  toApiDateTime, toDateTimeInputValue,
} from '@/utils/time'

const units = ref<Unit[]>([])
const projects = ref<Project[]>([])
const loading = ref(false)
const saving = ref(false)
const page = ref(1)
const totalPages = ref(0)
const viewMode = ref('card')
const showCreateDialog = ref(false)
const editingUnit = ref<Unit | null>(null)

// 计时类别：time（时间型）/ count（数值型）
const timerCategory = ref<'time' | 'count'>('time')
// 计时方向：countdown（倒计时）/ countup（正计时）
const timerDirection = ref<'countdown' | 'countup'>('countdown')

// 由两个维度自动推导 form.type
watch([timerCategory, timerDirection], ([cat, dir]) => {
  form.type = `${cat}_${dir}`
})

const filters = reactive({
  q: '',
  type: '',
  status: '',
  priority: '',
  sort_by: 'created_at',
})

const form = reactive({
  title: '',
  type: 'time_countdown',
  description: '',
  status: 'active',
  priority: 'normal',
  tags: [] as string[],
  color: '',
  project_id: null as string | null,
  target_time: '',
  start_time: '',
  current_value: 0,
  target_value: 0,
  step: 1,
  unit_label: '',
  remind_before_days: [] as number[],
  remind_after_days: [] as number[],
  remind_on_values: [] as number[],
})

const typeOptions = [
  { title: '时间倒计时', value: 'time_countdown' },
  { title: '时间正计时', value: 'time_countup' },
  { title: '数值倒计时', value: 'count_countdown' },
  { title: '数值正计时', value: 'count_countup' },
]

const categoryOptions = [
  { title: '⏱ 时间型', value: 'time' },
  { title: '🔢 数值型', value: 'count' },
]

const statusOptions = [
  { title: '激活', value: 'active' },
  { title: '暂停', value: 'paused' },
  { title: '已完成', value: 'completed' },
  { title: '已归档', value: 'archived' },
]

const priorityOptions = [
  { title: '低', value: 'low' },
  { title: '普通', value: 'normal' },
  { title: '高', value: 'high' },
  { title: '紧急', value: 'critical' },
]

const sortOptions = [
  { title: '创建时间', value: 'created_at' },
  { title: '更新时间', value: 'updated_at' },
  { title: '优先级', value: 'priority' },
]

let debounceTimer: number | null = null
function debouncedFetch() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = window.setTimeout(fetchUnits, 300)
}

async function fetchUnits() {
  loading.value = true
  try {
    const params: Record<string, any> = { page: page.value, page_size: 20 }
    if (filters.q) params.q = filters.q
    if (filters.type) params.type = filters.type
    if (filters.status) params.status = filters.status
    if (filters.priority) params.priority = filters.priority
    if (filters.sort_by) params.sort_by = filters.sort_by

    const resp = await unitApi.list(params)
    units.value = resp.data || []
    totalPages.value = resp.meta?.total_pages || 0
  } catch {
    // ignore
  } finally {
    loading.value = false
  }
}

async function handleStep(id: string, direction: 'up' | 'down') {
  try {
    await unitApi.step(id, direction)
    fetchUnits()
  } catch {
    // ignore
  }
}

function editUnit(unit: Unit) {
  editingUnit.value = unit
  // 解析类别和方向
  const [cat, dir] = unit.type.split('_') as ['time' | 'count', 'countdown' | 'countup']
  timerCategory.value = cat
  timerDirection.value = dir
  Object.assign(form, {
    title: unit.title,
    type: unit.type,
    description: unit.description,
    status: unit.status,
    priority: unit.priority,
    tags: unit.tags || [],
    color: unit.color,
    project_id: unit.project_id || null,
    target_time: toDateTimeInputValue(unit.target_time),
    start_time: toDateTimeInputValue(unit.start_time),
    current_value: unit.current_value || 0,
    target_value: unit.target_value || 0,
    step: unit.step || 1,
    unit_label: unit.unit_label || '',
    remind_before_days: (unit as any).remind_before_days || [],
    remind_after_days: (unit as any).remind_after_days || [],
    remind_on_values: (unit as any).remind_on_values || [],
  })
  showCreateDialog.value = true
}

function closeDialog() {
  showCreateDialog.value = false
  editingUnit.value = null
  resetForm()
}

function resetForm() {
  timerCategory.value = 'time'
  timerDirection.value = 'countdown'
  Object.assign(form, {
    title: '', type: 'time_countdown', description: '', status: 'active',
    priority: 'normal', tags: [], color: '', project_id: null,
    target_time: '', start_time: '', current_value: 0, target_value: 0, step: 1, unit_label: '',
    remind_before_days: [], remind_after_days: [], remind_on_values: [],
  })
}

async function saveUnit() {
  if (!form.title || !form.type) return
  saving.value = true
  try {
    const payload: Record<string, any> = {
      title: form.title,
      type: form.type,
      description: form.description,
      status: form.status,
      priority: form.priority,
      tags: form.tags,
      color: form.color,
    }

    // 处理项目归属
    if (form.project_id) {
      payload.project_id = form.project_id
    } else if (editingUnit.value?.project_id && !form.project_id) {
      // 编辑时从有项目改为无项目，需要清除
      payload.clear_project = true
    }

    if (form.type === 'time_countdown' && form.target_time) {
      payload.target_time = toApiDateTime(form.target_time)
      payload.remind_before_days = form.remind_before_days
    }
    if (form.type === 'time_countup' && form.start_time) {
      payload.start_time = toApiDateTime(form.start_time)
      payload.remind_after_days = form.remind_after_days
    }
    if (form.type === 'count_countdown' || form.type === 'count_countup') {
      payload.current_value = form.current_value
      payload.step = form.step
      payload.unit_label = form.unit_label
      payload.remind_on_values = form.remind_on_values
      if (form.type === 'count_countdown') {
        payload.target_value = form.target_value
      }
    }

    if (editingUnit.value) {
      await unitApi.update(editingUnit.value.id, payload)
    } else {
      await unitApi.create(payload)
    }
    closeDialog()
    fetchUnits()
  } catch {
    // ignore
  } finally {
    saving.value = false
  }
}

async function deleteUnit(id: string) {
  if (!confirm('确定要删除此计时单元吗？')) return
  try {
    await unitApi.delete(id)
    fetchUnits()
  } catch {
    // ignore
  }
}

onMounted(() => {
  fetchUnits()
  projectApi.list({ page_size: 100 }).then(r => { projects.value = r.data || [] })
})
</script>
