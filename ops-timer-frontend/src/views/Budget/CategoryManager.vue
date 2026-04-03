<template>
  <v-dialog :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)" max-width="600" scrollable>
    <v-card rounded="lg">
      <v-card-title class="pa-4 d-flex align-center">
        <v-icon class="mr-2">mdi-tag-multiple</v-icon>
        分类管理
        <v-spacer />
        <v-btn icon="mdi-plus" variant="text" @click="openCreate" />
        <v-btn icon="mdi-close" variant="text" size="small" @click="$emit('update:modelValue', false)" />
      </v-card-title>
      <v-divider />

      <!-- 类型 Tab -->
      <v-tabs v-model="tab" color="primary" class="px-2">
        <v-tab value="expense">支出分类</v-tab>
        <v-tab value="income">收入分类</v-tab>
      </v-tabs>
      <v-divider />

      <v-card-text class="pa-0" style="min-height: 320px">
        <v-progress-circular v-if="loading" indeterminate class="ma-8" />
        <v-list v-else bg-color="transparent">
          <template v-for="cat in currentList" :key="cat.id">
            <v-list-item>
              <template #prepend>
                <v-avatar :color="cat.color || '#757575'" size="36" class="mr-2">
                  <v-icon size="18" color="white">{{ cat.icon || 'mdi-tag' }}</v-icon>
                </v-avatar>
              </template>
              <v-list-item-title>{{ cat.name }}</v-list-item-title>
              <v-list-item-subtitle>
                <v-chip size="x-small" variant="tonal" :color="cat.type === 'income' ? 'success' : cat.type === 'expense' ? 'error' : 'primary'">
                  {{ { income: '收入', expense: '支出', both: '通用' }[cat.type] }}
                </v-chip>
                <v-chip v-if="cat.is_system" size="x-small" variant="text" class="ml-1">系统内置</v-chip>
              </v-list-item-subtitle>
              <template #append>
                <v-btn
                  icon="mdi-pencil"
                  size="small"
                  variant="text"
                  :disabled="cat.is_system"
                  @click="openEdit(cat)"
                />
                <v-btn
                  icon="mdi-delete"
                  size="small"
                  variant="text"
                  color="error"
                  :disabled="cat.is_system"
                  @click="deleteCategory(cat)"
                />
              </template>
            </v-list-item>
            <v-divider />
          </template>
          <div v-if="currentList.length === 0" class="pa-6 text-center text-medium-emphasis">暂无分类</div>
        </v-list>
      </v-card-text>
    </v-card>

    <!-- 新增/编辑分类对话框 -->
    <v-dialog v-model="editDialog" max-width="400" persistent>
      <v-card rounded="lg">
        <v-card-title class="pa-4">{{ editingCat ? '编辑分类' : '新增分类' }}</v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <v-text-field
            v-model="catForm.name"
            label="分类名称"
            variant="outlined"
            density="compact"
            class="mb-3"
          />
          <v-select
            v-model="catForm.type"
            :items="[{title: '支出', value: 'expense'}, {title: '收入', value: 'income'}, {title: '通用', value: 'both'}]"
            label="类型"
            variant="outlined"
            density="compact"
            class="mb-3"
          />
          <div class="text-caption mb-2 text-medium-emphasis">颜色</div>
          <div class="d-flex flex-wrap gap-1 mb-3">
            <div
              v-for="c in colorOptions"
              :key="c"
              class="color-dot cursor-pointer"
              :style="{ backgroundColor: c, border: catForm.color === c ? '3px solid #333' : '3px solid transparent' }"
              @click="catForm.color = c"
            />
          </div>
          <div class="text-caption mb-2 text-medium-emphasis">图标（MDI 图标名）</div>
          <v-text-field
            v-model="catForm.icon"
            variant="outlined"
            density="compact"
            placeholder="如 mdi-food"
          />
          <div class="mt-2 d-flex align-center">
            <v-icon :color="catForm.color || '#757575'" size="32">{{ catForm.icon || 'mdi-tag' }}</v-icon>
            <span class="text-caption ml-2 text-medium-emphasis">预览</span>
          </div>
        </v-card-text>
        <v-divider />
        <v-card-actions class="pa-3">
          <v-spacer />
          <v-btn variant="text" @click="editDialog = false">取消</v-btn>
          <v-btn color="primary" variant="flat" :loading="saving" @click="saveCategory">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { categoryApi } from '@/api/budget'
import type { BudgetCategory } from '@/types'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
}>()

const categories = ref<BudgetCategory[]>([])
const loading = ref(false)
const tab = ref('expense')
const editDialog = ref(false)
const editingCat = ref<BudgetCategory | null>(null)
const saving = ref(false)

const catForm = ref({ name: '', type: 'expense' as string, color: '#757575', icon: 'mdi-tag' })
const colorOptions = [
  '#F4511E', '#E53935', '#8E24AA', '#FB8C00', '#F57C00',
  '#43A047', '#00ACC1', '#1E88E5', '#039BE5', '#757575',
]

const currentList = computed(() =>
  categories.value.filter(c => c.type === tab.value || c.type === 'both')
)

async function fetchCategories() {
  loading.value = true
  try {
    const res = await categoryApi.list()
    categories.value = res.data ?? []
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingCat.value = null
  catForm.value = { name: '', type: tab.value, color: '#757575', icon: 'mdi-tag' }
  editDialog.value = true
}

function openEdit(cat: BudgetCategory) {
  editingCat.value = cat
  catForm.value = { name: cat.name, type: cat.type, color: cat.color || '#757575', icon: cat.icon || 'mdi-tag' }
  editDialog.value = true
}

async function saveCategory() {
  if (!catForm.value.name) return
  saving.value = true
  try {
    if (editingCat.value) {
      await categoryApi.update(editingCat.value.id, catForm.value)
    } else {
      await categoryApi.create({ ...catForm.value, type: catForm.value.type as any })
    }
    editDialog.value = false
    await fetchCategories()
  } catch (e) {
    console.error(e)
  } finally {
    saving.value = false
  }
}

async function deleteCategory(cat: BudgetCategory) {
  if (!confirm(`确认删除分类「${cat.name}」？`)) return
  try {
    await categoryApi.remove(cat.id)
    await fetchCategories()
  } catch (e: any) {
    alert(e?.response?.data?.message ?? '删除失败')
  }
}

watch(() => props.modelValue, (v) => {
  if (v) fetchCategories()
})
</script>

<style scoped>
.color-dot {
  width: 24px; height: 24px; border-radius: 50%; cursor: pointer; transition: transform 0.15s;
}
.color-dot:hover { transform: scale(1.2); }
.gap-1 { gap: 4px; }
</style>
