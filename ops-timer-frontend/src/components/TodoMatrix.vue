<template>
  <v-card class="rounded-lg mb-4 pa-4" aria-label="待办评分坐标图">
    <div class="d-flex align-center flex-wrap ga-2 mb-2">
      <h3 class="text-subtitle-1 font-weight-bold">重要程度 × 困难程度</h3>
      <v-spacer />
      <v-chip size="small" variant="tonal">当前页 {{ todos.length }} 项</v-chip>
    </div>
    <p class="text-body-2 text-medium-emphasis mb-4">向右更重要，向上更困难。拖动圆点，松手保存；中心为 (0.5, 0.5)。</p>
    <v-select v-model="selectedId" :items="todos" item-title="title" item-value="id" label="选择待办（重叠时可单独选取）" density="compact" hide-details class="mb-4" :disabled="!!savingId" />
    <div class="matrix-frame">
      <span class="y-title">困难程度 ↑</span>
      <div ref="plane" class="matrix-plane">
        <div class="axis axis-x" /><div class="axis axis-y" />
        <span class="quadrant top-left">较不重要 · 较困难</span>
        <span class="quadrant top-right">较重要 · 较困难</span>
        <span class="quadrant bottom-left">较不重要 · 较容易</span>
        <span class="quadrant bottom-right">较重要 · 较容易</span>
        <span class="center-label">0.5, 0.5</span>
        <span v-for="tick in [0, 0.5, 1]" :key="`x${tick}`" class="tick tick-x" :style="{ left: `${tick * 100}%` }">{{ tick }}</span>
        <span v-for="tick in [0, 0.5, 1]" :key="`y${tick}`" class="tick tick-y" :style="{ bottom: `${tick * 100}%` }">{{ tick }}</span>
        <button v-for="(todo, index) in todos" :key="todo.id" type="button" class="todo-point"
          :class="{ selected: selectedId === todo.id, saving: savingId === todo.id }"
          :data-id="todo.id" :style="pointStyle(todo)" :disabled="!!savingId || disabled"
          :aria-label="`${todo.title}，重要程度 ${scores(todo).importance}，困难程度 ${scores(todo).difficulty}。方向键调整，Enter 保存，Escape 取消。`"
          :title="`${todo.title} (${scores(todo).importance}, ${scores(todo).difficulty})`"
          @focus="selectedId = todo.id" @click="selectedId = todo.id" @keydown="onKey($event, todo)">
          {{ index + 1 }}
        </button>
        <div v-if="!todos.length" class="empty-label">当前筛选下暂无待办</div>
      </div>
      <span class="x-title">重要程度 →</span>
    </div>
    <div v-if="selectedTodo" class="text-body-2 mt-4">
      <strong>{{ selectedTodo.title }}</strong>
      <span class="ml-2">重要 {{ scores(selectedTodo).importance.toFixed(2) }} · 困难 {{ scores(selectedTodo).difficulty.toFixed(2) }}</span>
      <span class="d-block text-caption text-medium-emphasis">也可用方向键微调 0.01，Enter 保存，Esc 取消。</span>
      <v-btn v-if="draft && !dragging" size="small" variant="tonal" class="mt-2" :loading="!!savingId" @click="commit">保存位置</v-btn>
    </div>
    <p aria-live="polite" class="text-caption mt-2" :class="{ 'text-error': failed }">{{ message || '拖动时预览坐标，松手后自动保存。' }}</p>
  </v-card>
</template>

<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import interact from 'interactjs'
import type { Todo } from '@/types'
import { todoApi } from '@/api/todos'

type Scores = { importance: number; difficulty: number }
const props = defineProps<{ todos: Todo[]; disabled?: boolean }>()
const emit = defineEmits<{ saved: [todo: Todo] }>()
const plane = ref<HTMLElement>()
const selectedId = ref('')
const selectedTodo = computed(() => props.todos.find(t => t.id === selectedId.value))
const draft = ref<(Scores & { id: string }) | null>(null)
const dragging = ref(false)
const savingId = ref('')
const message = ref('')
const failed = ref(false)
const clamp = (value: number) => Math.round(Math.min(1, Math.max(0, value)) * 100) / 100
const scores = (todo: Todo): Scores => draft.value?.id === todo.id ? draft.value : todo
const pointStyle = (todo: Todo) => ({ left: `${scores(todo).importance * 100}%`, bottom: `${scores(todo).difficulty * 100}%`, zIndex: selectedId.value === todo.id ? 3 : 2 })
watch(selectedId, () => { if (!savingId.value) draft.value = null }, { flush: 'sync' })
watch(() => props.todos, (todos) => {
  if (!todos.some(t => t.id === selectedId.value)) selectedId.value = todos[0]?.id || ''
}, { immediate: true })

async function commit() {
  if (!draft.value || savingId.value) return
  const value = { ...draft.value }
  savingId.value = value.id
  message.value = '正在保存…'
  failed.value = false
  try {
    const response = await todoApi.update(value.id, { importance: value.importance, difficulty: value.difficulty })
    emit('saved', response.data)
    message.value = '位置已保存'
  } catch {
    message.value = '保存失败，已恢复原位置。请检查网络后重试。'
    failed.value = true
  } finally {
    draft.value = null
    savingId.value = ''
  }
}
function onKey(event: KeyboardEvent, todo: Todo) {
  if (event.key === 'Escape') { draft.value = null; message.value = '已取消调整'; return }
  if (event.key === 'Enter' && draft.value) { event.preventDefault(); void commit(); return }
  if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown'].includes(event.key)) return
  event.preventDefault()
  const value = scores(todo)
  draft.value = { id: todo.id, importance: clamp(value.importance + (event.key === 'ArrowRight' ? 0.01 : event.key === 'ArrowLeft' ? -0.01 : 0)), difficulty: clamp(value.difficulty + (event.key === 'ArrowUp' ? 0.01 : event.key === 'ArrowDown' ? -0.01 : 0)) }
}
let drag: ReturnType<typeof interact> | undefined
let position = { x: 0, y: 0 }
onMounted(() => {
  drag = interact('.todo-point', { context: plane.value! }).draggable({
    inertia: false,
    listeners: {
      start(event) {
        const id = (event.target as HTMLElement).dataset.id!
        const todo = props.todos.find(t => t.id === id)
        if (!todo || savingId.value || props.disabled) { event.interaction.stop(); return }
        selectedId.value = id
        draft.value = null
        position = { x: todo.importance, y: todo.difficulty }
        dragging.value = true
        failed.value = false
        message.value = '拖动中…'
      },
      move(event) {
        if (!dragging.value || !plane.value) return
        const rect = plane.value.getBoundingClientRect()
        position.x = Math.min(1, Math.max(0, position.x + event.dx / rect.width))
        position.y = Math.min(1, Math.max(0, position.y - event.dy / rect.height))
        draft.value = { id: (event.target as HTMLElement).dataset.id!, importance: clamp(position.x), difficulty: clamp(position.y) }
      },
      end() { dragging.value = false; void commit() },
    },
  })
})
onBeforeUnmount(() => drag?.unset())
</script>

<style scoped>
.matrix-frame { position: relative; padding: 28px 24px 42px 44px; }
.matrix-plane { position: relative; height: clamp(260px, 40vw, 400px); border: 1px solid rgba(var(--v-theme-on-surface), .22); border-radius: 8px; background: linear-gradient(90deg, transparent 49.9%, rgba(var(--v-theme-primary), .04) 50%), linear-gradient(0deg, rgba(var(--v-theme-success), .06) 50%, rgba(var(--v-theme-warning), .06) 50%); }
.axis { position: absolute; pointer-events: none; background: rgba(var(--v-theme-on-surface), .3); }
.axis-x { left: 0; right: 0; top: 50%; height: 1px; }.axis-y { top: 0; bottom: 0; left: 50%; width: 1px; }
.quadrant { position: absolute; font-size: 11px; color: rgba(var(--v-theme-on-surface), .5); pointer-events: none; }
.top-left { top: 12px; left: 12px; }.top-right { top: 12px; right: 12px; }.bottom-left { bottom: 12px; left: 12px; }.bottom-right { bottom: 12px; right: 12px; }
.center-label { position: absolute; top: calc(50% + 5px); left: calc(50% + 6px); font-size: 10px; opacity: .65; }
.tick { position: absolute; font-size: 12px; }.tick-x { bottom: -25px; transform: translateX(-50%); }.tick-y { left: -28px; transform: translateY(50%); }
.x-title { position: absolute; right: 22px; bottom: 0; font-size: 12px; }.y-title { position: absolute; top: 0; left: 42px; font-size: 12px; }
.todo-point { position: absolute; transform: translate(-50%, 50%); width: 32px; height: 32px; border: 2px solid rgb(var(--v-theme-surface)); border-radius: 50%; background: rgb(var(--v-theme-primary)); color: rgb(var(--v-theme-on-primary)); box-shadow: 0 2px 5px #0003; font-size: 12px; cursor: grab; touch-action: none; user-select: none; }
.todo-point.selected { outline: 3px solid rgba(var(--v-theme-primary), .4); }.todo-point:active { cursor: grabbing; }.todo-point:focus-visible { outline: 3px solid rgb(var(--v-theme-on-surface)); }.todo-point:disabled { cursor: wait; opacity: .65; }
.empty-label { position: absolute; top: 25%; left: 0; right: 0; text-align: center; font-size: 13px; opacity: .7; }
</style>
