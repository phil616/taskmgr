<template>
  <div class="notes-page">
    <div class="d-flex flex-wrap ga-2 align-center mb-4">
      <div>
        <h2 class="text-h5 font-weight-bold">笔记</h2>
        <div class="text-body-2 text-medium-emphasis mt-1">
          Markdown 笔记、标签、分组和全局搜索
        </div>
      </div>
      <v-spacer />
      <v-btn variant="tonal" prepend-icon="mdi-file-restore-outline" @click="openDrafts">
        草稿箱{{ drafts.length ? `（${drafts.length}）` : '' }}
      </v-btn>
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
                      <v-btn icon="mdi-content-copy" variant="text" aria-label="复制笔记全文" title="复制全文（Markdown 原文）" @click="copyFullText(activeNote.content)" />
                      <v-btn icon="mdi-pencil" variant="text" @click="openEditor(activeNote)" />
                    </div>
                  </div>

                  <MdPreview :model-value="activeNote.content" :theme="editorTheme" code-theme="github" preview-only />
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

    <v-dialog v-model="showEditorDialog" max-width="1200" persistent scrollable>
      <v-card class="rounded-lg">
        <v-card-title>{{ editingNote ? '编辑笔记' : '新建笔记' }}</v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <v-row dense class="mb-3">
            <v-col cols="12" md="6">
              <v-text-field
                v-model="noteForm.title"
                :disabled="saving"
                label="标题"
                :rules="[v => !!v || '必填']"
              />
            </v-col>
            <v-col cols="12" md="3">
              <v-select
                v-model="noteForm.group_id"
                :disabled="saving"
                :items="groupOptions"
                label="所属分组"
                clearable
              />
            </v-col>
            <v-col cols="12" md="3">
              <v-combobox
                v-model="noteForm.tags"
                :disabled="saving"
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
            editor-id="note-editor"
            language="zh-CN"
            :theme="editorTheme"
            code-theme="github"
            :tab-width="2"
            :read-only="saving"
            no-upload-img
            :toolbars="editorToolbars"
            style="height: clamp(280px, 55dvh, 560px)"
            @on-save="saveDraft"
            @on-error="editorError"
          />
          <div v-if="draftSavedAt" class="text-caption text-medium-emphasis mt-2" role="status">
            草稿已保存于 {{ formatDateTime(draftSavedAt) }}{{ draftDirty ? ' · 有尚未保存的修改' : '' }}
          </div>
        </v-card-text>
        <v-divider />
        <v-card-actions class="flex-wrap ga-1">
          <v-btn variant="text" prepend-icon="mdi-content-copy" @click="copyFullText(noteForm.content)">全文复制</v-btn>
          <v-btn variant="tonal" :disabled="saving" @click="saveAndCloseDraft">存草稿并关闭</v-btn>
          <v-spacer />
          <v-btn variant="text" :disabled="saving || savingDraft" @click="requestCloseEditor">关闭</v-btn>
          <v-btn variant="tonal" color="secondary" :loading="savingDraft" :disabled="saving" @click="saveDraft">保存草稿</v-btn>
          <v-btn color="primary" :loading="saving" :disabled="savingDraft" @click="saveNote">
            {{ editingNote ? '更新' : '创建' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="showDraftDialog" max-width="720" scrollable>
      <v-card>
        <v-card-title>草稿箱</v-card-title>
        <v-card-text>
          <v-alert v-if="draftLoadError" type="error" class="mb-3">{{ draftLoadError }}</v-alert>
          <v-progress-linear v-if="loadingDrafts" indeterminate />
          <v-list v-else-if="drafts.length" lines="three">
            <v-list-item v-for="draft in drafts" :key="draft.id" :title="draft.title || '无标题草稿'" :subtitle="`${formatDateTime(draft.updated_at)} · ${excerpt(draft.content)}`">
              <template #append>
                <v-btn variant="tonal" color="primary" class="mr-1" @click="restoreDraft(draft)">恢复</v-btn>
                <v-btn icon="mdi-content-copy" variant="text" aria-label="复制草稿全文" @click="copyFullText(draft.content)" />
                <v-btn icon="mdi-delete" variant="text" color="error" aria-label="删除草稿" @click="deleteDraft(draft)" />
              </template>
            </v-list-item>
          </v-list>
          <p v-else-if="!draftLoadError" class="py-8 text-center text-medium-emphasis">暂无草稿。在编辑器中可随时保存，标题和正文尚未填写完整也可以保存。</p>
        </v-card-text>
        <v-card-actions><v-spacer /><v-btn @click="showDraftDialog = false">关闭草稿箱</v-btn></v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="showUnsavedDialog" max-width="460" persistent>
      <v-card>
        <v-card-title>保留未保存的修改？</v-card-title>
        <v-card-text>保存到草稿后，可以从 Web 或 Qt 的草稿箱继续编辑。</v-card-text>
        <v-card-actions class="flex-wrap">
          <v-btn :disabled="savingDraft" @click="showUnsavedDialog = false">继续编辑</v-btn>
          <v-btn color="error" :disabled="savingDraft" @click="discardEditor">丢弃修改</v-btn>
          <v-btn color="primary" :loading="savingDraft" @click="saveAndCloseDraft">保存草稿</v-btn>
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

    <v-snackbar v-model="snackbar" :timeout="3500" location="top" :color="snackbarColor">
      {{ snackbarText }}
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import { useTheme } from 'vuetify'
import '@/plugins/markdown'
import { MdEditor, MdPreview } from 'md-editor-v3'
import type { ToolbarNames } from 'md-editor-v3'
import { noteApi } from '@/api/notes'
import type { NoteDraft, NoteDraftPayload } from '@/api/notes'
import { copyText } from '@/utils/clipboard'
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
const snackbarColor = ref('success')
const theme = useTheme()
const editorTheme = computed(() => theme.global.current.value.dark ? 'dark' : 'light')
const drafts = ref<NoteDraft[]>([])
const draftId = ref<string>()
const draftSavedAt = ref('')
const savingDraft = ref(false)
const loadingDrafts = ref(false)
const draftLoadError = ref('')
const showDraftDialog = ref(false)
const showUnsavedDialog = ref(false)
const initialSnapshot = ref('')
const draftSnapshot = ref('')

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
  'revoke',
  'next',
  'prettier',
  'save',
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

function toast(message: string, color = 'success') {
  snackbarText.value = message
  snackbarColor.value = color
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
  draftId.value = undefined
  draftSavedAt.value = ''
  editingNote.value = note || null
  Object.assign(noteForm, {
    title: note?.title || '',
    content: note?.content || '',
    group_id: note?.group_id || null,
    tags: note?.tags ? [...note.tags] : [],
  })
  showEditorDialog.value = true
  initialSnapshot.value = snapshot()
  draftSnapshot.value = ''
}

function payload(): NoteDraftPayload {
  return { note_id: editingNote.value?.id || null, title: noteForm.title,
    content: noteForm.content, group_id: noteForm.group_id,
    tags: noteForm.tags.filter(tag => typeof tag === 'string' && !!tag) }
}
function snapshot() { return JSON.stringify(payload()) }
const editorDirty = computed(() => showEditorDialog.value && snapshot() !== (draftSnapshot.value || initialSnapshot.value))
const draftDirty = computed(() => !!draftSnapshot.value && snapshot() !== draftSnapshot.value)

async function fetchDrafts() {
  loadingDrafts.value = true
  draftLoadError.value = ''
  try { drafts.value = (await noteApi.listDrafts()).data || [] }
  catch { draftLoadError.value = '草稿加载失败，请检查连接并确认后端已更新。' }
  finally { loadingDrafts.value = false }
}
async function openDrafts() { showDraftDialog.value = true; await fetchDrafts() }

async function saveDraft(): Promise<boolean> {
  if (saving.value || savingDraft.value) return false
  savingDraft.value = true
  const data = payload()
  const savedSnapshot = JSON.stringify(data)
  try {
    const saved = (await noteApi.saveDraft(data, draftId.value)).data
    draftId.value = saved.id
    draftSavedAt.value = saved.updated_at
    draftSnapshot.value = savedSnapshot
    drafts.value = [saved, ...drafts.value.filter(draft => draft.id !== saved.id)]
    toast('草稿已保存')
    return true
  } catch (error: any) {
    toast(error.response?.data?.message || '草稿保存失败，编辑内容已保留，请重试。', 'error')
    return false
  } finally { savingDraft.value = false }
}
async function saveAndCloseDraft() {
  if (await saveDraft()) {
    // Do not close if the user typed more while the request was in flight.
    if (draftDirty.value) { toast('草稿已保存，但还有新修改，请再次保存。', 'info'); return }
    showUnsavedDialog.value = false
    closeEditor()
  }
}
function requestCloseEditor() {
  if (editorDirty.value) showUnsavedDialog.value = true
  else closeEditor()
}
function discardEditor() { showUnsavedDialog.value = false; closeEditor() }

async function restoreDraft(draft: NoteDraft) {
  let original: Note | undefined
  if (draft.note_id) {
    try { original = (await noteApi.getById(draft.note_id)).data }
    catch (error: any) {
      if (error.response?.status !== 404) { toast('无法加载原笔记，请检查连接后重试。', 'error'); return }
      if (!window.confirm('原笔记已删除，是否将草稿恢复为新笔记？')) return
    }
  }
  openEditor(original)
  Object.assign(noteForm, { title: draft.title, content: draft.content,
    group_id: groups.value.some(group => group.id === draft.group_id) ? draft.group_id : null,
    tags: [...(draft.tags || [])] })
  draftId.value = draft.id
  draftSavedAt.value = draft.updated_at
  draftSnapshot.value = snapshot()
  showDraftDialog.value = false
}
async function deleteDraft(draft: NoteDraft) {
  if (!window.confirm(`确定删除草稿《${draft.title || '无标题草稿'}》吗？`)) return
  try { await noteApi.deleteDraft(draft.id); drafts.value = drafts.value.filter(item => item.id !== draft.id); toast('草稿已删除') }
  catch { toast('草稿删除失败，请重试。', 'error') }
}
async function copyFullText(content: string) {
  try { await copyText(content); toast('已复制全文（Markdown 原文）') }
  catch { toast('复制失败，请检查剪贴板权限后重试。', 'error') }
}
function editorError(error: { message?: string }) { toast(error.message || '编辑器操作失败', 'error') }

function beforeUnload(event: BeforeUnloadEvent) {
  if (editorDirty.value || saving.value || savingDraft.value) { event.preventDefault(); event.returnValue = '' }
}
onBeforeRouteLeave(() => !(editorDirty.value || saving.value || savingDraft.value)
  || window.confirm('有未保存的笔记修改，离开将丢失这些修改。请取消后保存到草稿。是否离开？'))

function closeEditor() {
  showEditorDialog.value = false
  editingNote.value = null
  draftId.value = undefined
  draftSavedAt.value = ''
  Object.assign(noteForm, {
    title: '',
    content: '',
    group_id: null,
    tags: [],
  })
}

async function saveNote() {
  if (saving.value || savingDraft.value) return
  if (!noteForm.title.trim() || !noteForm.content.trim()) {
    toast('请填写标题和正文；尚未完成的内容可以先保存草稿。', 'warning')
    return
  }

  saving.value = true
  try {
    const notePayload = {
      title: noteForm.title.trim(),
      content: noteForm.content,
      group_id: noteForm.group_id || undefined,
      tags: noteForm.tags.filter(Boolean),
    }

    if (draftId.value) {
      await noteApi.publishDraft(draftId.value, payload())
      drafts.value = drafts.value.filter(draft => draft.id !== draftId.value)
      toast('笔记已保存，草稿已提交')
    } else if (editingNote.value) {
      await noteApi.update(editingNote.value.id, { ...notePayload, group_id: noteForm.group_id || '' })
      toast('笔记已更新')
    } else {
      await noteApi.create(notePayload)
      toast('笔记已创建')
    }

    closeEditor()
    await Promise.all([fetchGroups(), fetchNotes()])
  } catch (error) {
    console.error(error)
    toast('笔记保存失败，编辑内容和草稿已保留，请重试。', 'error')
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
  window.addEventListener('beforeunload', beforeUnload)
  await Promise.all([fetchGroups(), fetchNotes(), fetchDrafts()])
})
onUnmounted(() => {
  window.removeEventListener('beforeunload', beforeUnload)
  if (debounceTimer) clearTimeout(debounceTimer)
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
