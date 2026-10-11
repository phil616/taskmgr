<template>
  <div>
    <div class="d-flex align-center flex-wrap ga-2 mb-4">
      <h2 class="text-h5 font-weight-bold">待办事项</h2>
      <v-spacer />
      <v-btn variant="outlined" class="mr-2" prepend-icon="mdi-folder-plus" @click="showGroupDialog = true">
        新建分组
      </v-btn>
      <v-btn color="primary" prepend-icon="mdi-plus" @click="showTodoDialog = true">
        新建待办
      </v-btn>
    </div>

    <v-row>
      <v-col cols="12" md="3">
        <v-card class="rounded-lg">
          <v-list density="compact" nav>
            <v-list-item
              :active="!selectedGroup"
              @click="selectedGroup = null; fetchTodos()"
              prepend-icon="mdi-format-list-bulleted"
              title="全部"
            >
              <template v-slot:append>
                <v-chip size="x-small">{{ totalCount }}</v-chip>
              </template>
            </v-list-item>
            <v-list-item
              :active="selectedGroup === 'none'"
              @click="selectedGroup = 'none'; fetchTodos()"
              prepend-icon="mdi-tray"
              title="未分组"
            />
            <v-divider class="my-1" />
            <v-list-item
              v-for="group in groups"
              :key="group.id"
              :active="selectedGroup === group.id"
              @click="selectedGroup = group.id; fetchTodos()"
            >
              <template v-slot:prepend>
                <v-icon :color="group.color || 'primary'">mdi-circle</v-icon>
              </template>
              <v-list-item-title>{{ group.name }}</v-list-item-title>
              <template v-slot:append>
                <v-chip size="x-small">{{ group.todo_count }}</v-chip>
                <v-btn icon="mdi-delete" size="x-small" variant="text" @click.stop="deleteGroup(group.id)" />
              </template>
            </v-list-item>
          </v-list>
        </v-card>
      </v-col>

      <v-col cols="12" md="9">
        <v-card class="rounded-lg mb-3 pa-3">
          <v-row dense>
            <v-col cols="4">
              <v-select v-model="statusFilter" :items="statusOptions" label="状态" density="compact" hide-details clearable @update:model-value="fetchTodos" />
            </v-col>
            <v-col cols="4">
              <v-select v-model="priorityFilter" :items="priorityOptions" label="优先级" density="compact" hide-details clearable @update:model-value="fetchTodos" />
            </v-col>
            <v-col cols="4" class="d-flex align-center justify-end">
              <v-btn v-if="selectedIds.length > 0" size="small" color="success" variant="tonal" class="mr-2" @click="batchComplete">
                批量完成 ({{ selectedIds.length }})
              </v-btn>
              <v-btn v-if="selectedIds.length > 0" size="small" color="error" variant="tonal" @click="batchDelete">
                批量删除
              </v-btn>
            </v-col>
          </v-row>
        </v-card>

        <TodoMatrix :todos="todos" :disabled="loading" @saved="onScoresSaved" />

        <v-card class="rounded-lg">
          <v-list v-if="todos.length > 0">
            <v-list-item v-for="todo in todos" :key="todo.id" class="py-2">
              <template v-slot:prepend>
                <v-checkbox-btn
                  :model-value="selectedIds.includes(todo.id)"
                  @update:model-value="toggleSelect(todo.id)"
                  class="mr-1"
                />
                <v-checkbox-btn
                  :model-value="todo.status === 'done'"
                  @update:model-value="toggleTodo(todo)"
                  color="success"
                />
              </template>
              <v-list-item-title
                :class="{ 'text-decoration-line-through text-medium-emphasis': todo.status === 'done' }"
              >
                {{ todo.title }}
              </v-list-item-title>
              <v-list-item-subtitle>
                <v-chip size="x-small" :color="getPriorityColor(todo.priority)" variant="tonal" class="mr-1">
                  {{ getPriorityLabel(todo.priority) }}
                </v-chip>
                <v-chip size="x-small" :color="getStatusColor(todo.status)" variant="flat" class="mr-1">
                  {{ getStatusLabel(todo.status) }}
                </v-chip>
                <span class="text-caption mr-2">重要程度: {{ todo.importance }} · 困难程度: {{ todo.difficulty }}</span>
                <span v-if="todo.due_date" class="text-caption">截止: {{ formatDate(todo.due_date) }}</span>
              </v-list-item-subtitle>
              <template v-slot:append>
                <v-btn icon="mdi-pencil" size="x-small" variant="text" @click="editTodo(todo)" />
                <v-btn icon="mdi-delete" size="x-small" variant="text" color="error" @click="deleteTodo(todo.id)" />
              </template>
            </v-list-item>
          </v-list>
          <v-card-text v-else class="text-center text-medium-emphasis py-8">
            暂无待办事项
          </v-card-text>
        </v-card>

        <div class="d-flex justify-center mt-4" v-if="totalPages > 1">
          <v-pagination v-model="page" :length="totalPages" @update:model-value="fetchTodos" />
        </div>
      </v-col>
    </v-row>

    <!-- Todo Dialog -->
    <v-dialog v-model="showTodoDialog" max-width="600" persistent>
      <v-card class="rounded-lg">
        <v-card-title>{{ editingTodo ? '编辑待办' : '新建待办' }}</v-card-title>
        <v-divider />
        <v-card-text>
          <v-text-field v-model="todoForm.title" label="标题" :rules="[v => !!v || '必填']" />
          <v-textarea v-model="todoForm.description" label="描述" rows="3" />
          <v-row dense>
            <v-col cols="6">
              <v-select v-model="todoForm.priority" :items="priorityOptions" label="优先级" />
            </v-col>
            <v-col cols="6">
              <v-text-field v-model="todoForm.due_date" label="截止日期" type="date" />
            </v-col>
          </v-row>
          <v-slider v-model="todoForm.importance" :min="0" :max="1" :step="0.01" thumb-label="always" label="重要程度" />
          <v-slider v-model="todoForm.difficulty" :min="0" :max="1" :step="0.01" thumb-label="always" label="困难程度" />
          <p class="text-caption mb-4">0 表示最不重要／最不困难，1 表示最重要／最困难</p>
          <v-select v-model="todoForm.group_id" :items="groupOptions" label="分组" clearable />
        </v-card-text>
        <v-divider />
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="closeTodoDialog">取消</v-btn>
          <v-btn color="primary" :loading="saving" @click="saveTodo">{{ editingTodo ? '更新' : '创建' }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Group Dialog -->
    <v-dialog v-model="showGroupDialog" max-width="400" persistent>
      <v-card class="rounded-lg">
        <v-card-title>新建分组</v-card-title>
        <v-divider />
        <v-card-text>
          <v-text-field v-model="groupForm.name" label="分组名称" />
          <v-text-field v-model="groupForm.color" label="颜色" placeholder="#1565C0" />
        </v-card-text>
        <v-divider />
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="showGroupDialog = false">取消</v-btn>
          <v-btn color="primary" @click="saveGroup">创建</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import type { Todo, TodoGroup } from '@/types'
import TodoMatrix from '@/components/TodoMatrix.vue'
import { todoApi } from '@/api/todos'
import { formatDate, getPriorityColor, getPriorityLabel, getStatusColor, getStatusLabel } from '@/utils/time'

const todos = ref<Todo[]>([])
const groups = ref<TodoGroup[]>([])
const loading = ref(false)
const saving = ref(false)
const page = ref(1)
const totalPages = ref(0)
const totalCount = ref(0)
const selectedGroup = ref<string | null>(null)
const statusFilter = ref('')
const priorityFilter = ref('')
const selectedIds = ref<string[]>([])
const showTodoDialog = ref(false)
const showGroupDialog = ref(false)
const editingTodo = ref<Todo | null>(null)

const todoForm = reactive({ title: '', description: '', priority: 'normal', importance: 0.5, difficulty: 0.5, due_date: '', group_id: '' as string | null })
const groupForm = reactive({ name: '', color: '' })

const statusOptions = [
  { title: '待办', value: 'pending' },
  { title: '进行中', value: 'in_progress' },
  { title: '已完成', value: 'done' },
  { title: '已取消', value: 'cancelled' },
]
const priorityOptions = [
  { title: '低', value: 'low' },
  { title: '普通', value: 'normal' },
  { title: '高', value: 'high' },
  { title: '紧急', value: 'critical' },
]
const groupOptions = computed(() =>
  groups.value.map(g => ({ title: g.name, value: g.id }))
)

async function fetchTodos() {
  loading.value = true
  try {
    const params: Record<string, any> = { page: page.value, page_size: 50 }
    if (selectedGroup.value) params.group_id = selectedGroup.value
    if (statusFilter.value) params.status = statusFilter.value
    if (priorityFilter.value) params.priority = priorityFilter.value
    const resp = await todoApi.list(params)
    todos.value = resp.data || []
    totalPages.value = resp.meta?.total_pages || 0
    totalCount.value = resp.meta?.total || 0
  } catch { /* ignore */ } finally { loading.value = false }
}

function onScoresSaved(todo: Todo) {
  const index = todos.value.findIndex(item => item.id === todo.id)
  if (index !== -1) todos.value[index] = todo
}

async function fetchGroups() {
  try { const resp = await todoApi.listGroups(); groups.value = resp.data || [] } catch { /* ignore */ }
}

function toggleSelect(id: string) {
  const idx = selectedIds.value.indexOf(id)
  if (idx >= 0) selectedIds.value.splice(idx, 1)
  else selectedIds.value.push(id)
}

async function toggleTodo(todo: Todo) {
  const s = todo.status === 'done' ? 'pending' : 'done'
  await todoApi.updateStatus(todo.id, s)
  fetchTodos()
  fetchGroups()
}

function editTodo(todo: Todo) {
  editingTodo.value = todo
  Object.assign(todoForm, {
    title: todo.title, description: todo.description,
    importance: todo.importance, difficulty: todo.difficulty,
    priority: todo.priority, due_date: todo.due_date?.slice(0, 10) || '',
    group_id: todo.group_id,
  })
  showTodoDialog.value = true
}

function closeTodoDialog() {
  showTodoDialog.value = false
  editingTodo.value = null
  Object.assign(todoForm, { title: '', description: '', priority: 'normal', importance: 0.5, difficulty: 0.5, due_date: '', group_id: null })
}

async function saveTodo() {
  if (!todoForm.title) return
  saving.value = true
  try {
    const payload: any = { ...todoForm }
    if (!payload.due_date) payload.due_date = undefined
    if (!payload.group_id) payload.group_id = undefined
    if (editingTodo.value) await todoApi.update(editingTodo.value.id, payload)
    else await todoApi.create(payload)
    closeTodoDialog()
    fetchTodos()
    fetchGroups()
  } catch { /* ignore */ } finally { saving.value = false }
}

async function deleteTodo(id: string) {
  await todoApi.delete(id); fetchTodos(); fetchGroups()
}

async function batchComplete() {
  await todoApi.batchAction('complete', selectedIds.value)
  selectedIds.value = []
  fetchTodos(); fetchGroups()
}

async function batchDelete() {
  if (!confirm(`确定删除 ${selectedIds.value.length} 项？`)) return
  await todoApi.batchAction('delete', selectedIds.value)
  selectedIds.value = []
  fetchTodos(); fetchGroups()
}

async function saveGroup() {
  if (!groupForm.name) return
  await todoApi.createGroup(groupForm)
  showGroupDialog.value = false
  Object.assign(groupForm, { name: '', color: '' })
  fetchGroups()
}

async function deleteGroup(id: string) {
  if (!confirm('删除分组后，其中的待办会移至未分组')) return
  await todoApi.deleteGroup(id)
  if (selectedGroup.value === id) selectedGroup.value = null
  fetchGroups(); fetchTodos()
}

onMounted(() => { fetchTodos(); fetchGroups() })
</script>
