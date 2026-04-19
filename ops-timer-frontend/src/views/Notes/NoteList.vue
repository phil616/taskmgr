<template>
  <div class="notes-page">
    <div class="d-flex align-center mb-4">
      <div>
        <h2 class="text-h5 font-weight-bold">笔记</h2>
        <div class="text-body-2 text-medium-emphasis mt-1">
          Markdown 笔记、标签、分组和全局搜索
        </div>
      </div>
      <v-spacer />
      <v-btn
        variant="outlined"
        class="mr-2"
        prepend-icon="mdi-folder-plus"
        @click="openGroupDialog()"
      >
        新建分组
      </v-btn>
      <v-btn color="primary" prepend-icon="mdi-plus" @click="openEditor()">
        新建笔记
      </v-btn>
    </div>

    <v-row>
      <v-col cols="12" md="3">
        <v-card class="rounded-lg">
          <v-list density="compact" nav>
            <v-list-item
              :active="selectedGroup === null"
              prepend-icon="mdi-view-grid-outline"
              title="全部笔记"
              @click="selectGroup(null)"
            >
              <template #append>
                <v-chip size="x-small">{{ totalCount }}</v-chip>
              </template>
            </v-list-item>
            <v-list-item
              :active="selectedGroup === 'none'"
              prepend-icon="mdi-tray"
              title="未分组"
              @click="selectGroup('none')"
            />
            <v-divider class="my-1" />
            <v-list-item
              v-for="group in groups"
              :key="group.id"
              :active="selectedGroup === group.id"
              @click="selectGroup(group.id)"
            >
              <template #prepend>
                <v-icon :color="group.color || 'primary'">mdi-folder</v-icon>
              </template>
              <v-list-item-title>{{ group.name }}</v-list-item-title>
              <template #append>
                <v-chip size="x-small" class="mr-1">{{ group.note_count }}</v-chip>
                <v-btn
                  icon="mdi-pencil"
                  size="x-small"
                  variant="text"
                  @click.stop="openGroupDialog(group)"
                />
                <v-btn
                  icon="mdi-delete"
                  size="x-small"
                  variant="text"
                  color="error"
                  @click.stop="deleteGroup(group)"
                />
              </template>
            </v-list-item>
          </v-list>
        </v-card>
      </v-col>

      <v-col cols="12" md="9">
        <v-card class="rounded-lg mb-3 pa-3">
          <v-row dense>
            <v-col cols="12" md="5">
              <v-text-field
                v-model="searchQuery"
                label="全局搜索"
                prepend-inner-icon="mdi-magnify"
                density="compact"
                hide-details
                clearable
                placeholder="搜索标题、内容、标签、分组"
                @update:model-value="debouncedFetch"
              />
            </v-col>
            <v-col cols="12" md="4">
              <v-combobox
                v-model="tagFilter"
                :items="availableTags"
                label="标签筛选"
                prepend-inner-icon="mdi-tag-outline"
                density="compact"
                hide-details
                clearable
                @update:model-value="fetchNotes"
              />
            </v-col>
            <v-col cols="12" md="3" class="d-flex align-center justify-end">
              <div class="text-caption text-medium-emphasis">
                {{ searchQuery ? '搜索结果' : '最近更新' }} · {{ totalCount }} 篇
              </div>
            </v-col>
          </v-row>
        </v-card>

        <v-row>
          <v-col cols="12" lg="5">
            <v-card class="rounded-lg note-list-card">
              <v-list v-if="notes.length > 0" lines="three">
                <v-list-item
                  v-for="note in notes"
                  :key="note.id"
                  :active="activeNote?.id === note.id"
                  class="py-3"
                  @click="selectNote(note)"
                >
                  <v-list-item-title class="font-weight-medium">
                    {{ note.title }}
                  </v-list-item-title>
                  <v-list-item-subtitle class="mb-1">
                    <span v-if="note.group_name" class="mr-2">{{ note.group_name }}</span>
                    <span>{{ formatDateTime(note.updated_at) }}</span>
                  </v-list-item-subtitle>
                  <v-list-item-subtitle class="note-excerpt">
                    {{ excerpt(note.content) }}
                  </v-list-item-subtitle>
                  <template #append>
                    <div class="d-flex flex-column align-end">
                      <div class="mb-1">
                        <v-btn icon="mdi-pencil" size="x-small" variant="text" @click.stop="openEditor(note)" />
                        <v-btn icon="mdi-delete" size="x-small" variant="text" color="error" @click.stop="deleteNote(note)" />
                      </div>
                      <div class="d-flex flex-wrap justify-end ga-1">
                        <v-chip
                          v-for="tag in note.tags.slice(0, 2)"
                          :key="tag"
                          size="x-small"
                          variant="tonal"
                          color="primary"
                        >
                          {{ tag }}
                        </v-chip>
                        <v-chip v-if="note.tags.length > 2" size="x-small" variant="tonal">
                          +{{ note.tags.length - 2 }}
                        </v-chip>
                      </div>
                    </div>
                  </template>
                </v-list-item>
              </v-list>
              <v-card-text v-else class="py-12 text-center text-medium-emphasis">
                <v-icon size="56" class="mb-3">mdi-note-off-outline</v-icon>
                <div class="text-body-1 mb-1">当前没有匹配的笔记</div>
                <div class="text-caption">试试调整搜索条件，或者直接创建一篇 Markdown 笔记。</div>
              </v-card-text>
            </v-card>
          </v-col>

          <v-col cols="12" lg="7">
            <v-card class="rounded-lg note-preview-card">
              <template v-if="activeNote">
                <div class="pa-4 pa-md-5">
                  <div class="d-flex align-start mb-4">
                    <div class="flex-grow-1">
                      <div class="text-h5 font-weight-bold mb-2">{{ activeNote.title }}</div>
                      <div class="d-flex flex-wrap ga-2 mb-3">
                        <v-chip v-if="activeNote.group_name" size="small" variant="tonal" color="info">
                          {{ activeNote.group_name }}
                        </v-chip>
                        <v-chip
                          v-for="tag in activeNote.tags"
                          :key="tag"
                          size="small"
                          variant="tonal"
                          color="primary"
                        >
                          {{ tag }}
                        </v-chip>
                      </div>
                      <div class="text-caption text-medium-emphasis">
                        创建于 {{ formatDateTime(activeNote.created_at) }} · 更新于 {{ formatDateTime(activeNote.updated_at) }}
                      </div>
                    </div>
                    <div class="ml-3">
                      <v-btn icon="mdi-pencil" variant="text" @click="openEditor(activeNote)" />
                    </div>
                  </div>

                  <MdPreview :model-value="activeNote.content" preview-only />
                </div>
              </template>
              <v-card-text v-else class="py-16 text-center text-medium-emphasis">
                <v-icon size="64" class="mb-4">mdi-file-document-outline</v-icon>
                <div class="text-body-1 mb-1">选择一篇笔记查看内容</div>
                <div class="text-caption">左侧列表支持分组、标签和全局搜索。</div>
              </v-card-text>
            </v-card>
          </v-col>
        </v-row>

        <div v-if="totalPages > 1" class="d-flex justify-center mt-4">
          <v-pagination v-model="page" :length="totalPages" @update:model-value="fetchNotes" />
        </div>
      </v-col>
    </v-row>

    <v-dialog v-model="showEditorDialog" max-width="1200" persistent>
      <v-card class="rounded-lg">
        <v-card-title>{{ editingNote ? '编辑笔记' : '新建笔记' }}</v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <v-row dense class="mb-3">
            <v-col cols="12" md="6">
              <v-text-field
                v-model="noteForm.title"
                label="标题"
                :rules="[v => !!v || '必填']"
              />
            </v-col>
            <v-col cols="12" md="3">
              <v-select
                v-model="noteForm.group_id"
                :items="groupOptions"
                label="所属分组"
                clearable
              />
            </v-col>
            <v-col cols="12" md="3">
              <v-combobox
                v-model="noteForm.tags"
                :items="availableTags"
                label="标签"
                multiple
                chips
                closable-chips
              />
            </v-col>
          </v-row>

          <MdEditor
            v-model="noteForm.content"
            language="zh-CN"
            :toolbars="editorToolbars"
            style="height: 560px"
          />
        </v-card-text>
        <v-divider />
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="closeEditor">取消</v-btn>
          <v-btn color="primary" :loading="saving" @click="saveNote">
            {{ editingNote ? '更新' : '创建' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="showGroupDialog" max-width="420" persistent>
      <v-card class="rounded-lg">
        <v-card-title>{{ editingGroup ? '编辑分组' : '新建分组' }}</v-card-title>
        <v-divider />
        <v-card-text>
          <v-text-field v-model="groupForm.name" label="分组名称" />
          <v-text-field v-model="groupForm.color" label="颜色" placeholder="#1976D2" />
        </v-card-text>
        <v-divider />
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="closeGroupDialog">取消</v-btn>
          <v-btn color="primary" @click="saveGroup">
            {{ editingGroup ? '更新' : '创建' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbar" :timeout="2200" location="top" color="success">
      {{ snackbarText }}
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { MdEditor, MdPreview } from 'md-editor-v3'
import type { ToolbarNames } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'
import 'md-editor-v3/lib/preview.css'
import { noteApi } from '@/api/notes'
import type { Note, NoteGroup } from '@/types'
import { formatDateTime } from '@/utils/time'

const notes = ref<Note[]>([])
const groups = ref<NoteGroup[]>([])
const activeNote = ref<Note | null>(null)
const editingNote = ref<Note | null>(null)
const editingGroup = ref<NoteGroup | null>(null)

const loading = ref(false)
const saving = ref(false)
const page = ref(1)
const totalPages = ref(0)
const totalCount = ref(0)
const selectedGroup = ref<string | null>(null)
const searchQuery = ref('')
const tagFilter = ref('')

const showEditorDialog = ref(false)
const showGroupDialog = ref(false)
const snackbar = ref(false)
const snackbarText = ref('')

const noteForm = reactive({
  title: '',
  content: '',
  group_id: null as string | null,
  tags: [] as string[],
})

const groupForm = reactive({
  name: '',
  color: '',
})

let debounceTimer: ReturnType<typeof setTimeout> | null = null

const groupOptions = computed(() =>
  groups.value.map(group => ({ title: group.name, value: group.id })),
)

const availableTags = computed(() => {
  const tags = new Set<string>()
  notes.value.forEach(note => {
    note.tags.forEach(tag => tags.add(tag))
  })
  return Array.from(tags).sort()
})

const editorToolbars: ToolbarNames[] = [
  'bold',
  'underline',
  'italic',
  '-',
  'title',
  'strikeThrough',
  'quote',
  'unorderedList',
  'orderedList',
  'task',
  '-',
  'codeRow',
  'code',
  'link',
  'table',
  '=',
  'preview',
  'fullscreen',
]

function excerpt(markdown: string) {
  return markdown
    .replace(/[#>*`~\-\[\]]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 96) || '空白内容'
}

function toast(message: string) {
  snackbarText.value = message
  snackbar.value = true
}

function selectGroup(groupId: string | null) {
  selectedGroup.value = groupId
  page.value = 1
  fetchNotes()
}

function selectNote(note: Note) {
  activeNote.value = note
}

function debouncedFetch() {
  page.value = 1
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    fetchNotes()
  }, 250)
}

async function fetchNotes() {
  loading.value = true
  try {
    const baseParams = {
      page: page.value,
      page_size: 20,
      group_id: selectedGroup.value || undefined,
      tag: tagFilter.value || undefined,
    }

    const response = searchQuery.value.trim()
      ? await noteApi.search({
          ...baseParams,
          q: searchQuery.value.trim(),
        })
      : await noteApi.list({
          ...baseParams,
          keyword: undefined,
        })

    notes.value = response.data || []
    totalPages.value = response.meta?.total_pages || 0
    totalCount.value = response.meta?.total || 0

    if (!notes.value.length) {
      activeNote.value = null
      return
    }

    const current = activeNote.value?.id
    activeNote.value = notes.value.find(note => note.id === current) || notes.value[0]
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

async function fetchGroups() {
  try {
    const response = await noteApi.listGroups()
    groups.value = response.data || []
  } catch (error) {
    console.error(error)
  }
}

function openEditor(note?: Note) {
  editingNote.value = note || null
  Object.assign(noteForm, {
    title: note?.title || '',
    content: note?.content || '',
    group_id: note?.group_id || null,
    tags: note?.tags ? [...note.tags] : [],
  })
  showEditorDialog.value = true
}

function closeEditor() {
  showEditorDialog.value = false
  editingNote.value = null
  Object.assign(noteForm, {
    title: '',
    content: '',
    group_id: null,
    tags: [],
  })
}

async function saveNote() {
  if (!noteForm.title.trim() || !noteForm.content.trim()) {
    return
  }

  saving.value = true
  try {
    const payload = {
      title: noteForm.title.trim(),
      content: noteForm.content,
      group_id: noteForm.group_id || undefined,
      tags: noteForm.tags.filter(Boolean),
    }

    if (editingNote.value) {
      await noteApi.update(editingNote.value.id, payload)
      toast('笔记已更新')
    } else {
      await noteApi.create(payload)
      toast('笔记已创建')
    }

    closeEditor()
    await Promise.all([fetchGroups(), fetchNotes()])
  } catch (error) {
    console.error(error)
  } finally {
    saving.value = false
  }
}

async function deleteNote(note: Note) {
  if (!window.confirm(`确定删除笔记《${note.title}》吗？`)) {
    return
  }

  try {
    await noteApi.delete(note.id)
    if (activeNote.value?.id === note.id) {
      activeNote.value = null
    }
    toast('笔记已删除')
    await Promise.all([fetchGroups(), fetchNotes()])
  } catch (error) {
    console.error(error)
  }
}

function openGroupDialog(group?: NoteGroup) {
  editingGroup.value = group || null
  Object.assign(groupForm, {
    name: group?.name || '',
    color: group?.color || '',
  })
  showGroupDialog.value = true
}

function closeGroupDialog() {
  showGroupDialog.value = false
  editingGroup.value = null
  Object.assign(groupForm, {
    name: '',
    color: '',
  })
}

async function saveGroup() {
  if (!groupForm.name.trim()) {
    return
  }

  try {
    const payload = {
      name: groupForm.name.trim(),
      color: groupForm.color.trim() || undefined,
    }
    if (editingGroup.value) {
      await noteApi.updateGroup(editingGroup.value.id, payload)
      toast('分组已更新')
    } else {
      await noteApi.createGroup(payload)
      toast('分组已创建')
    }
    closeGroupDialog()
    await fetchGroups()
  } catch (error) {
    console.error(error)
  }
}

async function deleteGroup(group: NoteGroup) {
  if (!window.confirm(`删除分组“${group.name}”后，分组内笔记将移至未分组。是否继续？`)) {
    return
  }

  try {
    await noteApi.deleteGroup(group.id)
    if (selectedGroup.value === group.id) {
      selectedGroup.value = null
    }
    toast('分组已删除')
    await Promise.all([fetchGroups(), fetchNotes()])
  } catch (error) {
    console.error(error)
  }
}

onMounted(async () => {
  await Promise.all([fetchGroups(), fetchNotes()])
})
</script>

<style scoped>
.notes-page :deep(.md-editor) {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}

.notes-page :deep(.md-editor-preview-wrapper),
.notes-page :deep(.md-editor-preview) {
  background: transparent;
}

.note-list-card,
.note-preview-card {
  min-height: 720px;
}

.note-excerpt {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
