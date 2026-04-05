<template>
  <div v-if="project">
    <div class="d-flex align-center mb-4">
      <v-btn icon="mdi-arrow-left" variant="text" @click="$router.back()" />
      <v-icon v-if="project.icon" class="ml-2 mr-2">{{ project.icon }}</v-icon>
      <h2 class="text-h5 font-weight-bold">{{ project.title }}</h2>
      <v-spacer />
      <v-chip :color="getStatusColor(project.status)">{{ getStatusLabel(project.status) }}</v-chip>
    </div>

    <v-row class="mb-4" v-if="project.unit_stats">
      <v-col cols="6" sm="3">
        <v-card class="rounded-lg text-center pa-4" variant="tonal" color="success">
          <div class="text-h5 font-weight-bold">{{ project.unit_stats.active_count }}</div>
          <div class="text-caption">活跃单元</div>
        </v-card>
      </v-col>
      <v-col cols="6" sm="3">
        <v-card class="rounded-lg text-center pa-4" variant="tonal" color="info">
          <div class="text-h5 font-weight-bold">{{ project.unit_stats.completed_count }}</div>
          <div class="text-caption">已完成</div>
        </v-card>
      </v-col>
      <v-col cols="6" sm="3">
        <v-card class="rounded-lg text-center pa-4" variant="tonal">
          <div class="text-h5 font-weight-bold">{{ project.unit_stats.total_count }}</div>
          <div class="text-caption">总计</div>
        </v-card>
      </v-col>
    </v-row>

    <v-card v-if="project.description" class="rounded-lg mb-4">
      <v-card-title>项目介绍</v-card-title>
      <v-divider />
      <v-card-text class="md-preview-wrap">
        <MdPreview :model-value="project.description" preview-only />
      </v-card-text>
    </v-card>

    <!-- 项目预算 -->
    <v-card class="rounded-lg mb-4">
      <v-card-title class="d-flex align-center">
        <v-icon class="mr-2" color="amber-darken-2">mdi-wallet</v-icon>
        项目预算
        <v-spacer />
        <v-btn size="small" variant="text" prepend-icon="mdi-pencil" @click="openBudgetDialog">
          设置预算
        </v-btn>
      </v-card-title>
      <v-divider />
      <v-card-text>
        <template v-if="budgetStats">
          <!-- 预算进度条 -->
          <div v-if="project.max_budget > 0" class="mb-4">
            <div class="d-flex justify-space-between align-center mb-1">
              <span class="text-body-2 text-medium-emphasis">预算使用</span>
              <span class="text-body-2 font-weight-medium">
                ¥{{ formatAmount(budgetStats.total_expense) }} / ¥{{ formatAmount(project.max_budget) }}
              </span>
            </div>
            <v-progress-linear
              :model-value="Math.min(budgetStats.usage_rate * 100, 100)"
              :color="budgetProgressColor"
              height="12"
              rounded
              class="mb-1"
            />
            <div class="d-flex justify-space-between">
              <span class="text-caption" :class="budgetStats.remaining >= 0 ? 'text-success' : 'text-error'">
                {{ budgetStats.remaining >= 0 ? '剩余' : '超支' }} ¥{{ formatAmount(Math.abs(budgetStats.remaining)) }}
              </span>
              <span class="text-caption text-medium-emphasis">
                {{ (budgetStats.usage_rate * 100).toFixed(1) }}%
              </span>
            </div>
          </div>
          <div v-else class="text-center text-medium-emphasis py-2 mb-3">
            <v-icon size="20" class="mr-1">mdi-information-outline</v-icon>
            未设置最大预算，点击右上角"设置预算"配置
          </div>

          <!-- 收支汇总 -->
          <v-row>
            <v-col cols="4">
              <div class="text-center">
                <div class="text-caption text-medium-emphasis">总收入</div>
                <div class="text-body-1 font-weight-bold text-success">+¥{{ formatAmount(budgetStats.total_income) }}</div>
              </div>
            </v-col>
            <v-col cols="4">
              <div class="text-center">
                <div class="text-caption text-medium-emphasis">总支出</div>
                <div class="text-body-1 font-weight-bold text-error">-¥{{ formatAmount(budgetStats.total_expense) }}</div>
              </div>
            </v-col>
            <v-col cols="4">
              <div class="text-center">
                <div class="text-caption text-medium-emphasis">净额</div>
                <div class="text-body-1 font-weight-bold" :class="budgetStats.net_amount >= 0 ? 'text-success' : 'text-error'">
                  {{ budgetStats.net_amount >= 0 ? '+' : '' }}¥{{ formatAmount(budgetStats.net_amount) }}
                </div>
              </div>
            </v-col>
          </v-row>
        </template>
        <div v-else-if="budgetLoaded" class="text-center text-medium-emphasis py-4">
          <v-icon size="20" class="mr-1">mdi-information-outline</v-icon>
          未设置最大预算，点击右上角"设置预算"配置
        </div>
        <div v-else class="text-center py-4">
          <v-progress-circular indeterminate size="24" />
        </div>
      </v-card-text>
    </v-card>

    <!-- 项目收支记录 -->
    <v-card class="rounded-lg mb-4">
      <v-card-title class="d-flex align-center">
        <v-icon class="mr-2" color="blue">mdi-format-list-bulleted</v-icon>
        项目收支记录
        <v-chip size="small" class="ml-2" variant="tonal">{{ projectTransactions.length }}</v-chip>
        <v-spacer />
        <v-btn size="small" color="primary" variant="tonal" prepend-icon="mdi-plus" @click="openAddTxDialog">
          添加记录
        </v-btn>
      </v-card-title>
      <v-divider />

      <v-list v-if="projectTransactions.length > 0" lines="two">
        <v-list-item
          v-for="tx in projectTransactions"
          :key="tx.id"
          @click="openEditTxDialog(tx)"
          class="cursor-pointer"
        >
          <template #prepend>
            <v-avatar :color="tx.category_color || '#9E9E9E'" size="36">
              <v-icon size="18" color="white">{{ tx.category_icon || 'mdi-cash' }}</v-icon>
            </v-avatar>
          </template>
          <v-list-item-title class="font-weight-medium">
            {{ tx.category_name || '未分类' }}
            <span v-if="tx.note" class="text-medium-emphasis text-body-2 ml-2">{{ tx.note }}</span>
          </v-list-item-title>
          <v-list-item-subtitle>
            {{ dayjs(tx.transaction_at).format('YYYY-MM-DD HH:mm') }}
            <span v-if="tx.wallet_name" class="ml-1">· {{ tx.wallet_name }}</span>
          </v-list-item-subtitle>
          <template #append>
            <span
              class="text-body-1 font-weight-bold"
              :class="tx.type === 'income' ? 'text-success' : tx.type === 'expense' ? 'text-error' : 'text-primary'"
            >
              {{ tx.type === 'income' ? '+' : tx.type === 'expense' ? '-' : '⇄' }}¥{{ formatAmount(tx.amount) }}
            </span>
          </template>
        </v-list-item>
      </v-list>

      <v-card-text v-else class="text-center text-medium-emphasis py-8">
        <v-icon size="48" color="grey-lighten-1" class="mb-3">mdi-receipt-text-outline</v-icon>
        <div>该项目暂无收支记录</div>
        <v-btn color="primary" variant="tonal" class="mt-3" prepend-icon="mdi-plus" @click="openAddTxDialog">
          添加记录
        </v-btn>
      </v-card-text>
    </v-card>

    <!-- 关联单元管理 -->
    <v-card class="rounded-lg">
      <v-card-title class="d-flex align-center">
        关联单元
        <v-chip size="small" class="ml-2" variant="tonal">{{ units.length }}</v-chip>
        <v-spacer />
        <v-btn
          size="small"
          color="primary"
          variant="tonal"
          prepend-icon="mdi-link-plus"
          @click="openAddDialog"
        >
          添加单元
        </v-btn>
      </v-card-title>
      <v-divider />

      <v-table v-if="units.length > 0">
        <thead>
          <tr>
            <th>标题</th>
            <th>类型</th>
            <th>状态</th>
            <th>优先级</th>
            <th>时间 / 进度</th>
            <th class="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="unit in units" :key="unit.id">
            <td>
              <router-link :to="`/units/${unit.id}`" class="text-decoration-none font-weight-medium">
                {{ unit.title }}
              </router-link>
            </td>
            <td>
              <v-chip size="x-small" variant="tonal">{{ getUnitTypeLabel(unit.type) }}</v-chip>
            </td>
            <td>
              <v-chip size="x-small" :color="getStatusColor(unit.status)" variant="flat">
                {{ getStatusLabel(unit.status) }}
              </v-chip>
            </td>
            <td>
              <v-chip size="x-small" :color="getPriorityColor(unit.priority)" variant="tonal">
                {{ getPriorityLabel(unit.priority) }}
              </v-chip>
            </td>
            <td class="text-body-2 text-medium-emphasis">
              <span v-if="unit.type === 'time_countdown'">
                {{ unit.remaining_seconds && unit.remaining_seconds > 0 ? formatDuration(unit.remaining_seconds) : '已超期' }}
              </span>
              <span v-else-if="unit.type === 'time_countup' && unit.elapsed_seconds">
                {{ formatDuration(unit.elapsed_seconds) }}
              </span>
              <span v-else-if="unit.type === 'count_countdown'">
                {{ unit.current_value || 0 }} / {{ unit.target_value }} {{ unit.unit_label }}
              </span>
              <span v-else>{{ unit.current_value || 0 }} {{ unit.unit_label }}</span>
            </td>
            <td class="text-right">
              <v-btn
                icon="mdi-link-off"
                size="small"
                variant="text"
                color="error"
                :loading="removingId === unit.id"
                title="从项目中移除"
                @click="removeUnit(unit)"
              />
            </td>
          </tr>
        </tbody>
      </v-table>

      <v-card-text v-else class="text-center text-medium-emphasis py-8">
        <v-icon size="48" color="grey-lighten-1" class="mb-3">mdi-link-off</v-icon>
        <div>该项目暂无关联单元</div>
        <v-btn color="primary" variant="tonal" class="mt-3" prepend-icon="mdi-link-plus" @click="openAddDialog">
          添加单元
        </v-btn>
      </v-card-text>
    </v-card>

    <!-- 设置预算对话框 -->
    <v-dialog v-model="showBudgetDialog" max-width="420" persistent>
      <v-card class="rounded-lg">
        <v-card-title class="d-flex align-center">
          <v-icon class="mr-2">mdi-wallet</v-icon>
          设置项目预算
          <v-spacer />
          <v-btn icon="mdi-close" variant="text" size="small" @click="showBudgetDialog = false" />
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <v-text-field
            v-model.number="budgetForm.max_budget"
            label="最大预算"
            type="number"
            variant="outlined"
            density="compact"
            prefix="¥"
            hint="设置为 0 表示不限制预算"
            persistent-hint
            :rules="[v => v >= 0 || '预算不能为负数']"
          />
        </v-card-text>
        <v-divider />
        <v-card-actions class="pa-3">
          <v-spacer />
          <v-btn variant="text" @click="showBudgetDialog = false">取消</v-btn>
          <v-btn color="primary" variant="flat" :loading="savingBudget" @click="saveBudget">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 添加单元对话框 -->
    <v-dialog v-model="showAddDialog" max-width="720" scrollable>
      <v-card class="rounded-lg">
        <v-card-title class="d-flex align-center">
          选择要加入「{{ project.title }}」的单元
          <v-spacer />
          <v-btn icon="mdi-close" variant="text" size="small" @click="showAddDialog = false" />
        </v-card-title>
        <v-divider />
        <v-card-text style="max-height: 480px; overflow-y: auto">
          <v-text-field
            v-model="addSearch"
            placeholder="搜索单元标题..."
            prepend-inner-icon="mdi-magnify"
            density="compact"
            hide-details
            clearable
            class="mb-3"
          />

          <div v-if="loadingAll" class="text-center py-6">
            <v-progress-circular indeterminate size="32" />
          </div>
          <div v-else-if="filteredAvailableUnits.length === 0" class="text-center text-medium-emphasis py-6">
            {{ addSearch ? '无匹配结果' : '所有单元均已在此项目中' }}
          </div>
          <v-list v-else select-strategy="independent" v-model:selected="selectedUnitIds" lines="two">
            <v-list-item
              v-for="unit in filteredAvailableUnits"
              :key="unit.id"
              :value="unit.id"
              rounded="lg"
              class="mb-1"
            >
              <template #prepend="{ isSelected }">
                <v-checkbox-btn :model-value="isSelected" color="primary" />
              </template>
              <v-list-item-title class="font-weight-medium">{{ unit.title }}</v-list-item-title>
              <v-list-item-subtitle>
                <v-chip size="x-small" variant="tonal" class="mr-1">{{ getUnitTypeLabel(unit.type) }}</v-chip>
                <v-chip size="x-small" :color="getStatusColor(unit.status)" variant="flat">{{ getStatusLabel(unit.status) }}</v-chip>
                <span v-if="unit.project_id" class="ml-2 text-caption text-warning">
                  当前所属：{{ getProjectTitle(unit.project_id) }}
                </span>
              </v-list-item-subtitle>
            </v-list-item>
          </v-list>
        </v-card-text>
        <v-divider />
        <v-card-actions>
          <span class="text-caption text-medium-emphasis ml-2">
            已选 {{ selectedUnitIds.length }} 个
          </span>
          <v-spacer />
          <v-btn variant="text" @click="showAddDialog = false">取消</v-btn>
          <v-btn
            color="primary"
            :loading="adding"
            :disabled="selectedUnitIds.length === 0"
            @click="confirmAddUnits"
          >
            确认添加
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 移除确认对话框 -->
    <v-dialog v-model="showRemoveConfirm" max-width="420" persistent>
      <v-card class="rounded-lg">
        <v-card-title>移除单元</v-card-title>
        <v-card-text>
          确定要将「<strong>{{ removingUnit?.title }}</strong>」从本项目中移除吗？该单元本身不会被删除。
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="showRemoveConfirm = false">取消</v-btn>
          <v-btn color="error" :loading="!!removingId" @click="confirmRemove">确认移除</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 收支记录弹窗（复用 TransactionDialog） -->
    <TransactionDialog
      v-model="showTxDialog"
      :transaction="editingTx"
      :wallets="wallets"
      :default-project-id="projectId"
      @saved="onTxSaved"
      @deleted="onTxDeleted"
    />
  </div>
  <div v-else class="text-center py-12"><v-progress-circular indeterminate /></div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/preview.css'
import dayjs from 'dayjs'
import type { Project, Unit, ProjectBudgetStats, Transaction, Wallet } from '@/types'
import { projectApi } from '@/api/projects'
import { unitApi } from '@/api/units'
import { walletApi, transactionApi } from '@/api/budget'
import TransactionDialog from '@/views/Budget/TransactionDialog.vue'
import {
  getStatusColor, getStatusLabel, getPriorityColor, getPriorityLabel,
  getUnitTypeLabel, formatDuration,
} from '@/utils/time'

const route = useRoute()
const projectId = computed(() => route.params.id as string)
const project = ref<Project | null>(null)
const units = ref<Unit[]>([])
const allUnits = ref<Unit[]>([])
const loadingAll = ref(false)
const budgetStats = ref<ProjectBudgetStats | null>(null)
const budgetLoaded = ref(false)
const projectTransactions = ref<Transaction[]>([])
const wallets = ref<Wallet[]>([])

// 预算进度条颜色
const budgetProgressColor = computed(() => {
  if (!budgetStats.value) return 'primary'
  const rate = budgetStats.value.usage_rate
  if (rate >= 1) return 'error'
  if (rate >= 0.8) return 'warning'
  return 'success'
})

// 设置预算
const showBudgetDialog = ref(false)
const savingBudget = ref(false)
const budgetForm = ref({ max_budget: 0 })

// 添加单元
const showAddDialog = ref(false)
const addSearch = ref('')
const selectedUnitIds = ref<string[]>([])
const adding = ref(false)

// 移除单元
const showRemoveConfirm = ref(false)
const removingUnit = ref<Unit | null>(null)
const removingId = ref<string | null>(null)

// 收支记录弹窗
const showTxDialog = ref(false)
const editingTx = ref<Transaction | null>(null)

function formatAmount(n: number) {
  return Math.abs(n).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

async function fetchProject() {
  try {
    const resp = await projectApi.getById(projectId.value)
    project.value = resp.data
    budgetStats.value = resp.data?.budget_stats ?? null
  } catch (e) {
    console.error(e)
  } finally {
    budgetLoaded.value = true
  }
}

async function fetchUnits() {
  const resp = await projectApi.getUnits(projectId.value, { page_size: 200 })
  units.value = resp.data || []
}

async function fetchProjectTransactions() {
  try {
    const resp = await transactionApi.list({ project_id: projectId.value, page_size: 200 })
    projectTransactions.value = resp.data || []
  } catch (e) {
    console.error(e)
  }
}

async function fetchWallets() {
  try {
    const resp = await walletApi.list()
    wallets.value = resp.data || []
  } catch (e) {
    console.error(e)
  }
}

async function fetchAllUnits() {
  loadingAll.value = true
  try {
    const resp = await unitApi.list({ page_size: 200 })
    allUnits.value = resp.data || []
  } finally {
    loadingAll.value = false
  }
}

function getProjectTitle(pid: string): string {
  if (project.value && project.value.id === pid) return project.value.title
  return pid
}

const currentProjectUnitIds = computed(() => new Set(units.value.map(u => u.id)))

const availableUnits = computed(() =>
  allUnits.value.filter(u => !currentProjectUnitIds.value.has(u.id))
)

const filteredAvailableUnits = computed(() => {
  if (!addSearch.value) return availableUnits.value
  const q = addSearch.value.toLowerCase()
  return availableUnits.value.filter(u => u.title.toLowerCase().includes(q))
})

// --- 预算设置 ---
function openBudgetDialog() {
  budgetForm.value.max_budget = project.value?.max_budget ?? 0
  showBudgetDialog.value = true
}

async function saveBudget() {
  if (budgetForm.value.max_budget < 0) return
  savingBudget.value = true
  try {
    await projectApi.update(projectId.value, { max_budget: budgetForm.value.max_budget })
    showBudgetDialog.value = false
    await fetchProject()
  } catch (e) {
    console.error(e)
  } finally {
    savingBudget.value = false
  }
}

// --- 收支记录 ---
function openAddTxDialog() {
  editingTx.value = null
  showTxDialog.value = true
}

function openEditTxDialog(tx: Transaction) {
  editingTx.value = tx
  showTxDialog.value = true
}

async function onTxSaved() {
  await Promise.all([fetchProject(), fetchProjectTransactions()])
}

async function onTxDeleted() {
  await Promise.all([fetchProject(), fetchProjectTransactions()])
}

// --- 单元管理 ---
async function openAddDialog() {
  selectedUnitIds.value = []
  addSearch.value = ''
  showAddDialog.value = true
  await fetchAllUnits()
}

async function confirmAddUnits() {
  if (selectedUnitIds.value.length === 0) return
  adding.value = true
  try {
    await Promise.all(
      selectedUnitIds.value.map(id =>
        unitApi.assignToProject(id, projectId.value)
      )
    )
    showAddDialog.value = false
    await Promise.all([fetchProject(), fetchUnits()])
  } finally {
    adding.value = false
  }
}

function removeUnit(unit: Unit) {
  removingUnit.value = unit
  showRemoveConfirm.value = true
}

async function confirmRemove() {
  if (!removingUnit.value) return
  removingId.value = removingUnit.value.id
  try {
    await unitApi.removeFromProject(removingUnit.value.id)
    showRemoveConfirm.value = false
    await Promise.all([fetchProject(), fetchUnits()])
  } finally {
    removingId.value = null
    removingUnit.value = null
  }
}

onMounted(async () => {
  await fetchProject()
  fetchUnits()
  fetchProjectTransactions()
  fetchWallets()
})
</script>

<style>
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
