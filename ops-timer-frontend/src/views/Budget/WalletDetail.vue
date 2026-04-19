<template>
  <v-dialog :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)" fullscreen transition="dialog-bottom-transition">
    <v-card>
      <!-- 顶栏 -->
      <v-toolbar :style="{ backgroundColor: wallet.color || '#1976D2' }" class="text-white">
        <v-btn icon="mdi-arrow-left" variant="text" color="white" @click="$emit('update:modelValue', false)" />
        <v-toolbar-title>
          <span class="font-weight-bold">{{ wallet.name }}</span>
          <span class="text-body-2 ml-2" style="opacity: 0.8">{{ walletTypeLabel(wallet.type) }}</span>
        </v-toolbar-title>
        <v-spacer />
        <v-btn variant="flat" color="white" class="mr-1" prepend-icon="mdi-plus" @click="openCreate">新增记录</v-btn>
      </v-toolbar>

      <!-- 余额卡 -->
      <div :style="{ backgroundColor: wallet.color || '#1976D2' }" class="pa-4 pb-6 text-white balance-header">
        <div class="text-caption" style="opacity: 0.8">当前余额</div>
        <div class="text-h3 font-weight-bold mt-1">
          {{ wallet.currency }} {{ formatAmount(walletDetail?.balance ?? wallet.balance) }}
        </div>
        <div class="d-flex mt-3 gap-6">
          <div>
            <div class="text-caption" style="opacity: 0.75">本月收入</div>
            <div class="text-subtitle-1 font-weight-medium">+{{ formatAmount(stat?.total_income ?? 0) }}</div>
          </div>
          <div>
            <div class="text-caption" style="opacity: 0.75">本月支出</div>
            <div class="text-subtitle-1 font-weight-medium">-{{ formatAmount(stat?.total_expense ?? 0) }}</div>
          </div>
          <div>
            <div class="text-caption" style="opacity: 0.75">净余</div>
            <div class="text-subtitle-1 font-weight-medium">{{ stat ? (stat.net_amount >= 0 ? '+' : '') + formatAmount(stat.net_amount) : '-' }}</div>
          </div>
        </div>
      </div>

      <v-container class="pa-4" style="max-width: 860px">
        <!-- 筛选条 -->
        <v-card rounded="lg" elevation="1" class="mb-4 pa-3">
          <div class="d-flex align-center flex-wrap gap-2">
            <v-select
              v-model="filterType"
              :items="[{title: '全部', value: ''}, {title: '收入', value: 'income'}, {title: '支出', value: 'expense'}, {title: '转账', value: 'transfer'}]"
              label="类型"
              variant="outlined"
              density="compact"
              style="max-width: 120px"
              hide-details
              @update:model-value="fetchTransactions"
            />
            <v-text-field
              v-model="filterStart"
              type="date"
              label="开始日期"
              variant="outlined"
              density="compact"
              hide-details
              style="max-width: 150px"
              @update:model-value="fetchTransactions"
            />
            <v-text-field
              v-model="filterEnd"
              type="date"
              label="结束日期"
              variant="outlined"
              density="compact"
              hide-details
              style="max-width: 150px"
              @update:model-value="fetchTransactions"
            />
            <v-text-field
              v-model="filterKeyword"
              label="备注搜索"
              variant="outlined"
              density="compact"
              hide-details
              style="max-width: 160px"
              clearable
              @update:model-value="fetchTransactions"
            />
          </div>
        </v-card>

        <!-- 交易列表 -->
        <v-card rounded="lg" elevation="1" class="mb-4">
          <div v-if="loading" class="pa-8 text-center">
            <v-progress-circular indeterminate color="primary" />
          </div>
          <div v-else-if="transactions.length === 0" class="pa-8 text-center text-medium-emphasis">
            <v-icon size="40" class="mb-2">mdi-text-box-outline</v-icon>
            <div>暂无收支记录</div>
          </div>
          <v-list v-else bg-color="transparent">
            <template v-for="group in groupedTx" :key="group.date">
              <!-- 日期分组标题 -->
              <div class="d-flex align-center px-4 py-2 bg-surface-variant">
                <span class="text-caption font-weight-bold text-medium-emphasis">{{ formatGroupDate(group.date) }}</span>
                <v-spacer />
                <span class="text-caption text-success mr-2">+{{ formatAmount(group.income) }}</span>
                <span class="text-caption text-error">-{{ formatAmount(group.expense) }}</span>
              </div>
              <v-list-item
                v-for="tx in group.items"
                :key="tx.id"
                class="tx-item"
                @click="openEdit(tx)"
              >
                <template #prepend>
                  <v-avatar
                    size="38"
                    :color="tx.category_color || typeColor(tx.type)"
                    class="mr-3"
                  >
                    <v-icon size="20" color="white">{{ tx.category_icon || typeIcon(tx.type) }}</v-icon>
                  </v-avatar>
                </template>
                <v-list-item-title class="text-body-2 font-weight-medium">
                  {{ tx.category_name || typeName(tx.type) }}
                  <span v-if="tx.to_wallet_name" class="text-caption text-medium-emphasis"> → {{ tx.to_wallet_name }}</span>
                </v-list-item-title>
                <v-list-item-subtitle v-if="tx.note" class="text-caption">{{ tx.note }}</v-list-item-subtitle>
                <template #append>
                  <div class="text-right">
                    <div
                      class="text-body-2 font-weight-bold"
                      :class="tx.type === 'income' ? 'text-success' : tx.type === 'expense' ? 'text-error' : 'text-primary'"
                    >
                      {{ tx.type === 'income' ? '+' : '-' }}¥{{ formatAmount(tx.amount) }}
                    </div>
                    <div class="text-caption text-medium-emphasis">{{ formatTime(tx.transaction_at) }}</div>
                  </div>
                </template>
              </v-list-item>
              <v-divider />
            </template>
          </v-list>

          <!-- 分页 -->
          <div v-if="totalPages > 1" class="pa-3 d-flex justify-center">
            <v-pagination
              v-model="currentPage"
              :length="totalPages"
              size="small"
              @update:model-value="fetchTransactions"
            />
          </div>
        </v-card>

        <!-- 分类统计 -->
        <v-card rounded="lg" elevation="1" class="pa-4" v-if="stat?.category_stats?.length">
          <div class="text-subtitle-2 font-weight-bold mb-3">分类统计（本月）</div>
          <div v-for="item in sortedCatStats" :key="item.category_id + item.type" class="mb-2">
            <div class="d-flex align-center mb-1">
              <v-icon :color="item.category_color || '#757575'" size="18" class="mr-2">{{ item.category_icon || 'mdi-tag' }}</v-icon>
              <span class="text-body-2 flex-1">{{ item.category_name }}</span>
              <v-chip :color="item.type === 'income' ? 'success' : 'error'" size="x-small" variant="tonal" class="mr-2">
                {{ item.type === 'income' ? '收入' : '支出' }}
              </v-chip>
              <span class="text-body-2 font-weight-bold" :class="item.type === 'income' ? 'text-success' : 'text-error'">
                ¥{{ formatAmount(item.total) }}
              </span>
            </div>
            <v-progress-linear
              :model-value="statPercent(item)"
              :color="item.type === 'income' ? 'success' : 'error'"
              rounded
              height="4"
              bg-color="surface-variant"
            />
          </div>
        </v-card>
      </v-container>
    </v-card>

    <!-- 新增/编辑对话框 -->
    <TransactionDialog
      v-model="txDialogOpen"
      :transaction="editingTx"
      :wallets="allWallets"
      :default-wallet-id="wallet.id"
      @saved="onSaved"
      @deleted="onDeleted"
    />
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { transactionApi, budgetStatApi, walletApi } from '@/api/budget'
import type { Transaction, Wallet, WalletStatResponse } from '@/types'
import TransactionDialog from './TransactionDialog.vue'
import { APP_TIMEZONE, dayjs, parseAppTime } from '@/utils/time'

const props = defineProps<{
  modelValue: boolean
  wallet: Wallet
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'updated'): void
}>()

const transactions = ref<Transaction[]>([])
const loading = ref(false)
const currentPage = ref(1)
const totalPages = ref(1)
const stat = ref<WalletStatResponse | null>(null)
const walletDetail = ref<Wallet | null>(null)
const allWallets = ref<Wallet[]>([])

const filterType = ref('')
const filterStart = ref(dayjs().tz(APP_TIMEZONE).startOf('month').format('YYYY-MM-DD'))
const filterEnd = ref(dayjs().tz(APP_TIMEZONE).endOf('month').format('YYYY-MM-DD'))
const filterKeyword = ref('')

const txDialogOpen = ref(false)
const editingTx = ref<Transaction | null>(null)

const groupedTx = computed(() => {
  const map = new Map<string, { date: string; items: Transaction[]; income: number; expense: number }>()
  for (const tx of transactions.value) {
    const d = parseAppTime(tx.transaction_at)?.format('YYYY-MM-DD')
    if (!d) continue
    if (!map.has(d)) map.set(d, { date: d, items: [], income: 0, expense: 0 })
    const g = map.get(d)!
    g.items.push(tx)
    if (tx.type === 'income') g.income += tx.amount
    if (tx.type === 'expense') g.expense += tx.amount
  }
  return Array.from(map.values()).sort((a, b) => b.date.localeCompare(a.date))
})

const sortedCatStats = computed(() => {
  if (!stat.value?.category_stats) return []
  return [...stat.value.category_stats].sort((a, b) => b.total - a.total)
})

function statPercent(item: { total: number; type: string }) {
  const base = item.type === 'income' ? stat.value?.total_income ?? 1 : stat.value?.total_expense ?? 1
  return base === 0 ? 0 : Math.round((item.total / base) * 100)
}

function formatAmount(n: number) {
  return n.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function formatGroupDate(d: string) {
  const day = parseAppTime(d)
  if (!day) return d
  const isToday = day.isSame(dayjs().tz(APP_TIMEZONE), 'day')
  const weekday = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][day.day()]
  return `${day.format('M月D日')} ${weekday}${isToday ? ' (今天)' : ''}`
}

function formatTime(dt: string) {
  return parseAppTime(dt)?.format('HH:mm') || '--:--'
}

function walletTypeLabel(type: string) {
  return { bank: '银行卡', cash: '现金', credit: '信用卡', alipay: '支付宝', wechat: '微信', other: '其他' }[type] ?? type
}

function typeColor(t: string) { return { income: '#43A047', expense: '#E53935', transfer: '#1E88E5' }[t] ?? '#757575' }
function typeIcon(t: string) { return { income: 'mdi-arrow-down-circle', expense: 'mdi-arrow-up-circle', transfer: 'mdi-swap-horizontal' }[t] ?? 'mdi-cash' }
function typeName(t: string) { return { income: '收入', expense: '支出', transfer: '转账' }[t] ?? t }

async function fetchTransactions() {
  loading.value = true
  try {
    const res = await transactionApi.list({
      wallet_id: props.wallet.id,
      type: filterType.value as any || undefined,
      start_date: filterStart.value,
      end_date: filterEnd.value,
      keyword: filterKeyword.value || undefined,
      page: currentPage.value,
      page_size: 20,
    })
    transactions.value = res.data ?? []
    if (res.meta) {
      totalPages.value = res.meta.total_pages
    }
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

async function fetchStat() {
  try {
    const res = await budgetStatApi.get({
      wallet_id: props.wallet.id,
      start_date: filterStart.value,
      end_date: filterEnd.value,
    })
    stat.value = res.data ?? null
  } catch (e) {
    console.error(e)
  }
}

async function fetchWalletDetail() {
  try {
    const res = await walletApi.get(props.wallet.id)
    walletDetail.value = res.data ?? null
  } catch (e) { /* ignore */ }
}

async function fetchAllWallets() {
  try {
    const res = await walletApi.list()
    allWallets.value = res.data ?? []
  } catch (e) { /* ignore */ }
}

function openCreate() {
  editingTx.value = null
  txDialogOpen.value = true
}

function openEdit(tx: Transaction) {
  editingTx.value = tx
  txDialogOpen.value = true
}

async function onSaved() {
  await fetchTransactions()
  await fetchStat()
  await fetchWalletDetail()
  emit('updated')
}

async function onDeleted() {
  await fetchTransactions()
  await fetchStat()
  await fetchWalletDetail()
  emit('updated')
}

watch(() => props.modelValue, (v) => {
  if (v) {
    currentPage.value = 1
    fetchTransactions()
    fetchStat()
    fetchWalletDetail()
    fetchAllWallets()
  }
}, { immediate: true })
</script>

<style scoped>
.balance-header {
  background: linear-gradient(135deg, rgba(0,0,0,0.1) 0%, transparent 100%);
}
.tx-item {
  cursor: pointer;
  transition: background 0.15s;
}
.tx-item:hover {
  background: rgba(var(--v-theme-primary), 0.05);
}
.gap-2 { gap: 8px; }
.gap-6 { gap: 24px; }
.flex-1 { flex: 1; }
</style>
