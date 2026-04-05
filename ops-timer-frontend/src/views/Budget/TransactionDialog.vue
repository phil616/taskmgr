<template>
  <v-dialog :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)" max-width="520" persistent>
    <v-card rounded="lg">
      <v-card-title class="pa-4 d-flex align-center">
        <v-icon class="mr-2" :color="form.type === 'income' ? 'success' : form.type === 'expense' ? 'error' : 'primary'">
          {{ form.type === 'income' ? 'mdi-arrow-down-circle' : form.type === 'expense' ? 'mdi-arrow-up-circle' : 'mdi-swap-horizontal' }}
        </v-icon>
        {{ transaction ? '编辑记录' : '新增记录' }}
        <v-spacer />
        <v-btn icon="mdi-close" variant="text" size="small" @click="$emit('update:modelValue', false)" />
      </v-card-title>
      <v-divider />

      <v-card-text class="pa-4">
        <!-- 类型切换 -->
        <v-btn-toggle v-model="form.type" class="mb-4 w-100" mandatory density="compact" rounded="lg" color="primary">
          <v-btn value="expense" class="flex-1">
            <v-icon start>mdi-arrow-up-circle</v-icon>支出
          </v-btn>
          <v-btn value="income" class="flex-1">
            <v-icon start>mdi-arrow-down-circle</v-icon>收入
          </v-btn>
          <v-btn value="transfer" class="flex-1">
            <v-icon start>mdi-swap-horizontal</v-icon>转账
          </v-btn>
        </v-btn-toggle>

        <!-- 金额 -->
        <v-text-field
          v-model.number="form.amount"
          label="金额"
          type="number"
          variant="outlined"
          density="compact"
          prefix="¥"
          class="mb-3"
          :rules="[v => v > 0 || '金额必须大于0']"
        />

        <!-- 钱包 -->
        <v-select
          v-model="form.wallet_id"
          :items="walletOptions"
          :label="form.type === 'transfer' ? '来源钱包' : '钱包'"
          variant="outlined"
          density="compact"
          class="mb-3"
          :rules="[v => !!v || '请选择钱包']"
        />

        <!-- 转账目标钱包 -->
        <v-select
          v-if="form.type === 'transfer'"
          v-model="form.to_wallet_id"
          :items="walletOptions.filter(w => w.value !== form.wallet_id)"
          label="目标钱包"
          variant="outlined"
          density="compact"
          class="mb-3"
          :rules="[v => !!v || '请选择目标钱包']"
        />

        <!-- 分类 -->
        <v-select
          v-if="form.type !== 'transfer'"
          v-model="form.category_id"
          :items="filteredCategories"
          label="分类（可选）"
          variant="outlined"
          density="compact"
          class="mb-3"
          clearable
        >
          <template #item="{ props, item }">
            <v-list-item v-bind="props" :prepend-icon="item.raw.icon" :subtitle="undefined">
              <template #prepend>
                <v-icon :color="item.raw.color">{{ item.raw.icon }}</v-icon>
              </template>
            </v-list-item>
          </template>
          <template #selection="{ item }">
            <div class="d-flex align-center">
              <v-icon :color="item.raw.color" size="18" class="mr-1">{{ item.raw.icon }}</v-icon>
              {{ item.raw.title }}
            </div>
          </template>
        </v-select>

        <!-- 关联项目 -->
        <v-select
          v-model="form.project_id"
          :items="projectOptions"
          label="关联项目（可选）"
          variant="outlined"
          density="compact"
          class="mb-3"
          clearable
        />

        <!-- 交易时间 -->
        <v-text-field
          v-model="form.transaction_at"
          label="交易时间"
          type="datetime-local"
          variant="outlined"
          density="compact"
          class="mb-3"
        />

        <!-- 备注 -->
        <v-text-field
          v-model="form.note"
          label="备注（可选）"
          variant="outlined"
          density="compact"
          class="mb-3"
        />

        <!-- 标签 -->
        <v-combobox
          v-model="form.tags"
          label="标签（可选，回车添加）"
          variant="outlined"
          density="compact"
          multiple
          chips
          closable-chips
          class="mb-1"
        />
      </v-card-text>

      <v-divider />
      <v-card-actions class="pa-3">
        <v-btn v-if="transaction" color="error" variant="text" @click="deleteRecord">删除</v-btn>
        <v-spacer />
        <v-btn variant="text" @click="$emit('update:modelValue', false)">取消</v-btn>
        <v-btn color="primary" variant="flat" :loading="saving" @click="save">保存</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { transactionApi, categoryApi } from '@/api/budget'
import { projectApi } from '@/api/projects'
import type { Transaction, BudgetCategory, Wallet, Project } from '@/types'
import dayjs from 'dayjs'

const props = defineProps<{
  modelValue: boolean
  transaction?: Transaction | null
  wallets: Wallet[]
  defaultWalletId?: string
  defaultProjectId?: string
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'saved', tx: Transaction): void
  (e: 'deleted'): void
}>()

const saving = ref(false)
const categories = ref<BudgetCategory[]>([])
const projects = ref<Project[]>([])

const form = ref({
  type: 'expense' as 'income' | 'expense' | 'transfer',
  wallet_id: '',
  to_wallet_id: '',
  category_id: '',
  project_id: '',
  amount: 0 as number,
  note: '',
  tags: [] as string[],
  transaction_at: dayjs().format('YYYY-MM-DDTHH:mm'),
})

const projectOptions = computed(() =>
  projects.value.map(p => ({ title: p.title, value: p.id }))
)

const walletOptions = computed(() =>
  props.wallets.map(w => ({
    title: `${w.name} (¥${w.balance.toFixed(2)})`,
    value: w.id,
  }))
)

const filteredCategories = computed(() =>
  categories.value
    .filter(c => c.type === form.value.type || c.type === 'both')
    .map(c => ({ title: c.name, value: c.id, icon: c.icon, color: c.color }))
)

async function loadCategories() {
  try {
    const res = await categoryApi.list()
    categories.value = res.data ?? []
  } catch (e) {
    console.error(e)
  }
}

async function loadProjects() {
  try {
    const res = await projectApi.list({ status: 'active', page_size: 100 })
    projects.value = res.data ?? []
  } catch (e) {
    console.error(e)
  }
}

watch(() => props.modelValue, (v) => {
  if (!v) return
  loadCategories()
  loadProjects()
  if (props.transaction) {
    const tx = props.transaction
    form.value = {
      type: tx.type,
      wallet_id: tx.wallet_id,
      to_wallet_id: tx.to_wallet_id ?? '',
      category_id: tx.category_id ?? '',
      project_id: tx.project_id ?? '',
      amount: tx.amount,
      note: tx.note,
      tags: tx.tags ?? [],
      transaction_at: dayjs(tx.transaction_at).format('YYYY-MM-DDTHH:mm'),
    }
  } else {
    form.value = {
      type: 'expense',
      wallet_id: props.defaultWalletId ?? (props.wallets[0]?.id ?? ''),
      to_wallet_id: '',
      category_id: '',
      project_id: props.defaultProjectId ?? '',
      amount: 0,
      note: '',
      tags: [],
      transaction_at: dayjs().format('YYYY-MM-DDTHH:mm'),
    }
  }
})

async function save() {
  if (!form.value.wallet_id || form.value.amount <= 0) return
  saving.value = true
  try {
    // 确保时间格式含秒（datetime-local 不含秒时补全）
    const normalizeTime = (t: string) =>
      /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(t) ? `${t}:00` : t

    // 确保 tags 全为字符串（v-combobox 可能混入对象）
    const cleanTags = (form.value.tags ?? []).map(t =>
      typeof t === 'string' ? t : String((t as any)?.title ?? t)
    )

    const payload = {
      wallet_id: form.value.wallet_id,
      category_id: form.value.category_id || undefined,
      project_id: form.value.project_id || undefined,
      type: form.value.type,
      amount: form.value.amount,
      note: form.value.note,
      tags: cleanTags,
      transaction_at: normalizeTime(form.value.transaction_at),
      to_wallet_id: form.value.type === 'transfer' ? form.value.to_wallet_id : undefined,
    }
    let res
    if (props.transaction) {
      // 更新时：category_id 用空字符串表示"清除"，后端识别此语义
      const updatePayload = {
        category_id: form.value.category_id === null || form.value.category_id === ''
          ? ''
          : (form.value.category_id ?? undefined),
        project_id: form.value.project_id === null || form.value.project_id === ''
          ? ''
          : (form.value.project_id ?? undefined),
        amount: payload.amount,
        note: payload.note,
        tags: cleanTags,
        transaction_at: normalizeTime(form.value.transaction_at),
      }
      res = await transactionApi.update(props.transaction.id, updatePayload)
    } else {
      res = await transactionApi.create(payload)
    }
    emit('saved', res.data)
    emit('update:modelValue', false)
  } catch (e) {
    console.error(e)
  } finally {
    saving.value = false
  }
}

async function deleteRecord() {
  if (!props.transaction) return
  if (!confirm('确认删除该收支记录？删除后将回滚对应余额。')) return
  try {
    await transactionApi.remove(props.transaction.id)
    emit('deleted')
    emit('update:modelValue', false)
  } catch (e) {
    console.error(e)
  }
}
</script>

<style scoped>
.w-100 { width: 100%; }
.flex-1 { flex: 1; }
</style>
