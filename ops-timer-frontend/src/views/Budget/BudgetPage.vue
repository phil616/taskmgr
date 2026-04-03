<template>
  <div class="budget-page">
    <!-- 页面顶栏 -->
    <div class="d-flex align-center mb-5">
      <div>
        <div class="text-h6 font-weight-medium">预算管理</div>
        <div class="text-body-2 text-medium-emphasis">管理你的钱包账户与收支记录</div>
      </div>
      <v-spacer />
      <v-btn variant="text" class="mr-2" prepend-icon="mdi-tag-multiple" @click="showCategories = true">
        分类管理
      </v-btn>
      <v-btn color="primary" variant="flat" prepend-icon="mdi-plus" @click="openCreateWallet">
        新增钱包
      </v-btn>
    </div>

    <!-- 总览卡片 -->
    <v-row class="mb-5">
      <v-col cols="12" sm="4">
        <v-card rounded="lg" elevation="1" class="pa-4 overview-card income-card">
          <div class="text-caption text-medium-emphasis mb-1">本月总收入</div>
          <div class="text-h5 font-weight-bold text-success">
            + ¥{{ formatAmount(overviewIncome) }}
          </div>
        </v-card>
      </v-col>
      <v-col cols="12" sm="4">
        <v-card rounded="lg" elevation="1" class="pa-4 overview-card expense-card">
          <div class="text-caption text-medium-emphasis mb-1">本月总支出</div>
          <div class="text-h5 font-weight-bold text-error">
            - ¥{{ formatAmount(overviewExpense) }}
          </div>
        </v-card>
      </v-col>
      <v-col cols="12" sm="4">
        <v-card rounded="lg" elevation="1" class="pa-4 overview-card net-card">
          <div class="text-caption text-medium-emphasis mb-1">本月净余额</div>
          <div class="text-h5 font-weight-bold" :class="overviewNet >= 0 ? 'text-success' : 'text-error'">
            {{ overviewNet >= 0 ? '+' : '' }}¥{{ formatAmount(overviewNet) }}
          </div>
        </v-card>
      </v-col>
    </v-row>

    <!-- 钱包列表 -->
    <div v-if="loadingWallets" class="text-center pa-8">
      <v-progress-circular indeterminate color="primary" />
    </div>
    <div v-else-if="wallets.length === 0" class="text-center pa-8 text-medium-emphasis">
      <v-icon size="48" class="mb-2">mdi-wallet-outline</v-icon>
      <div>还没有钱包，点击右上角新增</div>
    </div>
    <v-row v-else>
      <v-col v-for="wallet in wallets" :key="wallet.id" cols="12" sm="6" md="4">
        <v-card
          rounded="lg"
          elevation="1"
          class="wallet-card cursor-pointer"
          hover
          @click="openWallet(wallet)"
        >
          <div class="wallet-header" :style="{ backgroundColor: wallet.color || '#1976D2' }">
            <div class="d-flex align-center justify-space-between">
              <div class="d-flex align-center">
                <v-icon size="28" color="white" class="mr-2">{{ wallet.icon || walletTypeIcon(wallet.type) }}</v-icon>
                <div>
                  <div class="text-subtitle-1 font-weight-bold text-white">{{ wallet.name }}</div>
                  <div class="text-caption" style="color: rgba(255,255,255,0.75)">{{ walletTypeLabel(wallet.type) }}</div>
                </div>
              </div>
              <div class="d-flex align-center">
                <v-chip v-if="wallet.is_default" size="x-small" variant="tonal" color="white" class="mr-1">默认</v-chip>
                <v-btn
                  icon="mdi-pencil"
                  size="small"
                  variant="text"
                  color="white"
                  @click.stop="openEditWallet(wallet)"
                />
              </div>
            </div>
            <div class="text-h4 font-weight-bold text-white mt-3">
              {{ wallet.currency }} {{ formatAmount(wallet.balance) }}
            </div>
          </div>
          <div class="pa-3 d-flex justify-space-around">
            <div class="text-center">
              <div class="text-caption text-medium-emphasis">本月收入</div>
              <div class="text-body-2 font-weight-medium text-success">+{{ formatAmount(wallet.total_income) }}</div>
            </div>
            <v-divider vertical />
            <div class="text-center">
              <div class="text-caption text-medium-emphasis">本月支出</div>
              <div class="text-body-2 font-weight-medium text-error">-{{ formatAmount(wallet.total_expense) }}</div>
            </div>
            <v-divider vertical />
            <div class="text-center">
              <div class="text-caption text-medium-emphasis">明细</div>
              <v-btn size="x-small" variant="text" color="primary" @click.stop="openWallet(wallet)">查看</v-btn>
            </div>
          </div>
        </v-card>
      </v-col>
    </v-row>

    <!-- 新增/编辑钱包对话框 -->
    <v-dialog v-model="walletDialog" max-width="480" persistent>
      <v-card rounded="lg">
        <v-card-title class="pa-4 d-flex align-center">
          <v-icon class="mr-2">mdi-wallet</v-icon>
          {{ editingWallet ? '编辑钱包' : '新增钱包' }}
          <v-spacer />
          <v-btn icon="mdi-close" variant="text" size="small" @click="walletDialog = false" />
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <v-text-field
            v-model="walletForm.name"
            label="钱包名称"
            variant="outlined"
            density="compact"
            class="mb-3"
            :rules="[v => !!v || '请输入名称']"
          />
          <v-select
            v-model="walletForm.type"
            label="账户类型"
            :items="walletTypeOptions"
            variant="outlined"
            density="compact"
            class="mb-3"
          />
          <v-text-field
            v-model.number="walletForm.balance"
            label="初始余额"
            type="number"
            variant="outlined"
            density="compact"
            class="mb-3"
            prefix="¥"
            :disabled="!!editingWallet"
            :hint="editingWallet ? '余额由收支记录自动计算，不可直接修改' : ''"
            persistent-hint
          />
          <v-text-field
            v-model="walletForm.description"
            label="备注说明（可选）"
            variant="outlined"
            density="compact"
            class="mb-3"
          />
          <div class="d-flex gap-3 mb-3">
            <div class="flex-1">
              <div class="text-caption mb-1 text-medium-emphasis">颜色</div>
              <div class="d-flex flex-wrap gap-1">
                <div
                  v-for="c in colorOptions"
                  :key="c"
                  class="color-dot cursor-pointer"
                  :style="{ backgroundColor: c, border: walletForm.color === c ? '3px solid #333' : '3px solid transparent' }"
                  @click="walletForm.color = c"
                />
              </div>
            </div>
          </div>
          <v-switch
            v-model="walletForm.is_default"
            label="设为默认钱包"
            color="primary"
            density="compact"
          />
        </v-card-text>
        <v-divider />
        <v-card-actions class="pa-3">
          <v-btn v-if="editingWallet" color="error" variant="text" @click="deleteWallet">删除钱包</v-btn>
          <v-spacer />
          <v-btn variant="text" @click="walletDialog = false">取消</v-btn>
          <v-btn color="primary" variant="flat" :loading="saving" @click="saveWallet">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 分类管理抽屉 -->
    <CategoryManager v-model="showCategories" />

    <!-- 钱包详情路由跳转 -->
    <WalletDetail
      v-if="activeWallet"
      v-model="showWalletDetail"
      :wallet="activeWallet"
      @updated="onDetailUpdated"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { walletApi, budgetStatApi } from '@/api/budget'
import type { Wallet, CreateWalletRequest } from '@/types'
import CategoryManager from './CategoryManager.vue'
import WalletDetail from './WalletDetail.vue'

const wallets = ref<Wallet[]>([])
const loadingWallets = ref(false)
const walletDialog = ref(false)
const editingWallet = ref<Wallet | null>(null)
const saving = ref(false)
const showCategories = ref(false)
const showWalletDetail = ref(false)
const activeWallet = ref<Wallet | null>(null)

const overviewIncome = ref(0)
const overviewExpense = ref(0)
const overviewNet = computed(() => overviewIncome.value - overviewExpense.value)

const walletForm = ref({
  name: '',
  type: 'bank' as string,
  balance: 0,
  currency: 'CNY',
  color: '#1976D2',
  icon: '',
  description: '',
  is_default: false,
})

const colorOptions = [
  '#1976D2', '#388E3C', '#F57C00', '#D32F2F',
  '#7B1FA2', '#0097A7', '#455A64', '#E91E63',
]

const walletTypeOptions = [
  { title: '银行卡', value: 'bank' },
  { title: '现金', value: 'cash' },
  { title: '信用卡', value: 'credit' },
  { title: '支付宝', value: 'alipay' },
  { title: '微信', value: 'wechat' },
  { title: '其他', value: 'other' },
]

function walletTypeLabel(type: string) {
  return walletTypeOptions.find(o => o.value === type)?.title ?? type
}

function walletTypeIcon(type: string) {
  return {
    bank: 'mdi-bank',
    cash: 'mdi-cash',
    credit: 'mdi-credit-card',
    alipay: 'mdi-alpha-a-circle',
    wechat: 'mdi-wechat',
    other: 'mdi-wallet',
  }[type] ?? 'mdi-wallet'
}

function formatAmount(n: number) {
  return n.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

async function fetchWallets() {
  loadingWallets.value = true
  try {
    const res = await walletApi.list()
    wallets.value = res.data ?? []
  } catch (e) {
    console.error(e)
  } finally {
    loadingWallets.value = false
  }
}

async function fetchOverview() {
  try {
    const now = new Date()
    const start = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-01`
    const end = new Date(now.getFullYear(), now.getMonth() + 1, 0)
    const endStr = `${end.getFullYear()}-${String(end.getMonth() + 1).padStart(2, '0')}-${String(end.getDate()).padStart(2, '0')}`
    const res = await budgetStatApi.get({ start_date: start, end_date: endStr })
    overviewIncome.value = res.data?.total_income ?? 0
    overviewExpense.value = res.data?.total_expense ?? 0
  } catch (e) {
    console.error(e)
  }
}

function openCreateWallet() {
  editingWallet.value = null
  walletForm.value = { name: '', type: 'bank', balance: 0, currency: 'CNY', color: '#1976D2', icon: '', description: '', is_default: false }
  walletDialog.value = true
}

function openEditWallet(w: Wallet) {
  editingWallet.value = w
  walletForm.value = {
    name: w.name,
    type: w.type,
    balance: w.balance,
    currency: w.currency,
    color: w.color || '#1976D2',
    icon: w.icon || '',
    description: w.description || '',
    is_default: w.is_default,
  }
  walletDialog.value = true
}

async function saveWallet() {
  if (!walletForm.value.name) return
  saving.value = true
  try {
    if (editingWallet.value) {
      await walletApi.update(editingWallet.value.id, {
        name: walletForm.value.name,
        type: walletForm.value.type as any,
        color: walletForm.value.color,
        icon: walletForm.value.icon,
        description: walletForm.value.description,
        is_default: walletForm.value.is_default,
      })
    } else {
      await walletApi.create(walletForm.value as CreateWalletRequest)
    }
    walletDialog.value = false
    await fetchWallets()
    fetchOverview()
  } catch (e) {
    console.error(e)
  } finally {
    saving.value = false
  }
}

async function deleteWallet() {
  if (!editingWallet.value) return
  if (!confirm(`确认删除钱包「${editingWallet.value.name}」？关联的收支记录不会被删除。`)) return
  try {
    await walletApi.remove(editingWallet.value.id)
    walletDialog.value = false
    await fetchWallets()
    fetchOverview()
  } catch (e) {
    console.error(e)
  }
}

function openWallet(w: Wallet) {
  activeWallet.value = w
  showWalletDetail.value = true
}

// 钱包详情有收支变动时，同步刷新总览数据
async function onDetailUpdated() {
  await fetchWallets()
  fetchOverview()
}

onMounted(async () => {
  await fetchWallets()
  fetchOverview()
})
</script>

<style scoped>
.budget-page {
  max-width: 1100px;
  margin: 0 auto;
}

.wallet-card {
  transition: transform 0.2s, box-shadow 0.2s;
  overflow: hidden;
}
.wallet-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 18px rgba(0,0,0,0.12) !important;
}

.wallet-header {
  padding: 16px;
  border-radius: 8px 8px 0 0;
}

.overview-card {
  border-left: 4px solid transparent;
}
.income-card { border-left-color: #4CAF50; }
.expense-card { border-left-color: #F44336; }
.net-card { border-left-color: #2196F3; }

.color-dot {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  transition: transform 0.15s;
}
.color-dot:hover { transform: scale(1.2); }
.gap-1 { gap: 4px; }
.gap-3 { gap: 12px; }
.flex-1 { flex: 1; }
</style>
