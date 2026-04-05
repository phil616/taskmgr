<template>
  <div>
    <!-- 页头 -->
    <div class="d-flex align-center mb-4">
      <h2 class="text-h5 font-weight-bold">密钥管理</h2>
      <v-spacer />
      <v-btn color="primary" prepend-icon="mdi-plus" @click="openCreateDialog">创建密钥</v-btn>
    </div>

    <!-- 筛选栏 -->
    <v-card class="rounded-lg mb-4" variant="flat" color="surface">
      <v-card-text class="py-3">
        <v-row dense align="center">
          <v-col cols="12" sm="4" md="3">
            <v-text-field
              v-model="filters.name"
              label="搜索名称"
              prepend-inner-icon="mdi-magnify"
              clearable
              hide-details
              density="compact"
              @update:model-value="debouncedFetch"
            />
          </v-col>
          <v-col cols="12" sm="4" md="3">
            <v-text-field
              v-model="filters.tag"
              label="按标签筛选"
              prepend-inner-icon="mdi-tag-outline"
              clearable
              hide-details
              density="compact"
              @update:model-value="debouncedFetch"
            />
          </v-col>
          <v-col cols="12" sm="4" md="3">
            <v-select
              v-model="filters.project_id"
              :items="projectOptions"
              label="按项目筛选"
              clearable
              hide-details
              density="compact"
              @update:model-value="fetchSecrets"
            />
          </v-col>
          <v-col cols="auto">
            <v-btn variant="text" icon="mdi-history" title="全部审计日志" @click="showAllAuditLogs" />
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <!-- 密钥列表 -->
    <v-card class="rounded-lg">
      <v-table hover>
        <thead>
          <tr>
            <th>名称</th>
            <th>描述</th>
            <th>标签</th>
            <th>项目</th>
            <th>创建时间</th>
            <th class="text-center" style="width: 200px;">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="secret in secrets" :key="secret.id">
            <td>
              <div class="d-flex align-center">
                <v-icon size="small" color="warning" class="mr-2">mdi-key-variant</v-icon>
                <span class="font-weight-medium">{{ secret.name }}</span>
              </div>
            </td>
            <td>
              <span class="text-medium-emphasis text-body-2">
                {{ secret.description ? (secret.description.slice(0, 60) + (secret.description.length > 60 ? '...' : '')) : '-' }}
              </span>
            </td>
            <td>
              <v-chip
                v-for="tag in (secret.tags || []).slice(0, 3)"
                :key="tag"
                size="x-small"
                variant="tonal"
                color="primary"
                class="mr-1"
              >{{ tag }}</v-chip>
              <v-chip v-if="(secret.tags || []).length > 3" size="x-small" variant="tonal">
                +{{ secret.tags.length - 3 }}
              </v-chip>
              <span v-if="!secret.tags || secret.tags.length === 0" class="text-medium-emphasis">-</span>
            </td>
            <td>
              <v-chip v-if="secret.project_name" size="x-small" variant="tonal" color="info">
                {{ secret.project_name }}
              </v-chip>
              <span v-else class="text-medium-emphasis">-</span>
            </td>
            <td class="text-body-2 text-medium-emphasis">{{ formatDateTime(secret.created_at) }}</td>
            <td class="text-center">
              <v-btn icon="mdi-eye-outline" size="small" variant="text" title="查看密钥值" @click="viewSecret(secret)" />
              <v-btn icon="mdi-pencil-outline" size="small" variant="text" title="编辑" @click="editSecret(secret)" />
              <v-btn icon="mdi-history" size="small" variant="text" title="审计日志" @click="showAuditLogs(secret)" />
              <v-btn icon="mdi-delete-outline" size="small" variant="text" color="error" title="删除" @click="deleteSecret(secret)" />
            </td>
          </tr>
        </tbody>
      </v-table>

      <!-- 空状态 -->
      <div v-if="secrets.length === 0 && !loading" class="text-center py-12">
        <v-icon size="64" color="grey-lighten-1">mdi-key-remove</v-icon>
        <p class="text-body-1 text-medium-emphasis mt-4">暂无密钥</p>
        <v-btn color="primary" class="mt-2" @click="openCreateDialog">创建第一个密钥</v-btn>
      </div>

      <v-divider v-if="totalPages > 1" />
      <div v-if="totalPages > 1" class="d-flex justify-center py-3">
        <v-pagination v-model="page" :length="totalPages" density="compact" @update:model-value="fetchSecrets" />
      </div>
    </v-card>

    <!-- 创建/编辑对话框 -->
    <v-dialog v-model="showFormDialog" max-width="640" persistent>
      <v-card class="rounded-lg">
        <v-card-title>{{ editingSecret ? '编辑密钥' : '创建密钥' }}</v-card-title>
        <v-divider />
        <v-card-text>
          <v-text-field
            v-model="form.name"
            label="密钥名称"
            :rules="[v => !!v || '必填']"
            placeholder="如 GITHUB_TOKEN"
          />
          <v-textarea
            v-model="form.value"
            label="密钥值"
            :rules="[v => !!v || '必填']"
            rows="3"
            placeholder="明文密钥值"
            :append-inner-icon="showFormValue ? 'mdi-eye-off' : 'mdi-eye'"
            :type="showFormValue ? 'text' : 'password'"
            @click:append-inner="showFormValue = !showFormValue"
          />
          <v-textarea
            v-model="form.description"
            label="描述（可选）"
            rows="2"
            placeholder="对此密钥的说明"
          />
          <v-combobox
            v-model="form.tags"
            label="标签（可选）"
            multiple
            chips
            closable-chips
            placeholder="输入后按回车添加"
            hide-details
            class="mb-3"
          />
          <v-select
            v-model="form.project_id"
            :items="projectOptions"
            label="关联项目（可选）"
            clearable
          />
        </v-card-text>
        <v-divider />
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="closeFormDialog">取消</v-btn>
          <v-btn color="primary" :loading="saving" @click="saveSecret">
            {{ editingSecret ? '更新' : '创建' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 查看密钥值对话框 -->
    <v-dialog v-model="showValueDialog" max-width="560">
      <v-card class="rounded-lg">
        <v-card-title class="d-flex align-center">
          <v-icon color="warning" class="mr-2">mdi-key-variant</v-icon>
          {{ viewingSecret?.name }}
        </v-card-title>
        <v-divider />
        <v-card-text>
          <div class="text-caption text-medium-emphasis mb-2">密钥值</div>
          <v-sheet rounded="lg" color="grey-darken-4" class="pa-3 d-flex align-center">
            <code class="flex-grow-1 text-body-2" style="word-break: break-all; color: #a5f3fc;">
              {{ showPlainValue ? secretValue : '•'.repeat(Math.min(secretValue.length, 40)) }}
            </code>
            <v-btn
              :icon="showPlainValue ? 'mdi-eye-off' : 'mdi-eye'"
              size="small"
              variant="text"
              color="white"
              class="ml-2"
              @click="showPlainValue = !showPlainValue"
            />
            <v-btn
              icon="mdi-content-copy"
              size="small"
              variant="text"
              color="white"
              class="ml-1"
              @click="copyValue"
            />
          </v-sheet>
          <div v-if="viewingSecret?.description" class="mt-4">
            <div class="text-caption text-medium-emphasis mb-1">描述</div>
            <p class="text-body-2">{{ viewingSecret.description }}</p>
          </div>
          <div v-if="viewingSecret?.tags?.length" class="mt-3">
            <div class="text-caption text-medium-emphasis mb-1">标签</div>
            <v-chip
              v-for="tag in viewingSecret.tags"
              :key="tag"
              size="small"
              variant="tonal"
              color="primary"
              class="mr-1"
            >{{ tag }}</v-chip>
          </div>
        </v-card-text>
        <v-divider />
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="showValueDialog = false">关闭</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 审计日志对话框 -->
    <v-dialog v-model="showAuditDialog" max-width="800">
      <v-card class="rounded-lg">
        <v-card-title class="d-flex align-center">
          <v-icon class="mr-2">mdi-history</v-icon>
          {{ auditTitle }}
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-0">
          <v-table density="compact" hover>
            <thead>
              <tr>
                <th>时间</th>
                <th>操作</th>
                <th>用户</th>
                <th>IP 地址</th>
                <th>详情</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="log in auditLogs" :key="log.id">
                <td class="text-body-2 text-no-wrap">{{ formatDateTime(log.created_at) }}</td>
                <td>
                  <v-chip :color="getAuditActionColor(log.action)" size="x-small" variant="flat">
                    {{ getAuditActionLabel(log.action) }}
                  </v-chip>
                </td>
                <td class="text-body-2">{{ log.username }}</td>
                <td class="text-body-2 text-medium-emphasis">{{ log.ip_address || '-' }}</td>
                <td class="text-body-2 text-medium-emphasis">{{ log.detail || '-' }}</td>
              </tr>
              <tr v-if="auditLogs.length === 0">
                <td colspan="5" class="text-center py-8 text-medium-emphasis">暂无审计记录</td>
              </tr>
            </tbody>
          </v-table>
          <div v-if="auditTotalPages > 1" class="d-flex justify-center py-3">
            <v-pagination
              v-model="auditPage"
              :length="auditTotalPages"
              density="compact"
              @update:model-value="fetchAuditLogs"
            />
          </div>
        </v-card-text>
        <v-divider />
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="showAuditDialog = false">关闭</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 复制成功 Snackbar -->
    <v-snackbar v-model="snackbar" :timeout="2000" color="success" location="top">
      {{ snackbarText }}
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { SecretBrief, Secret, SecretAuditLog, Project } from '@/types'
import { secretApi } from '@/api/secrets'
import { projectApi } from '@/api/projects'
import { formatDateTime } from '@/utils/time'

const secrets = ref<SecretBrief[]>([])
const loading = ref(false)
const saving = ref(false)
const page = ref(1)
const totalPages = ref(0)

const showFormDialog = ref(false)
const showFormValue = ref(false)
const editingSecret = ref<SecretBrief | null>(null)
const form = reactive({
  name: '',
  value: '',
  description: '',
  tags: [] as string[],
  project_id: null as string | null,
})

const showValueDialog = ref(false)
const viewingSecret = ref<SecretBrief | null>(null)
const secretValue = ref('')
const showPlainValue = ref(false)

const showAuditDialog = ref(false)
const auditTitle = ref('审计日志')
const auditLogs = ref<SecretAuditLog[]>([])
const auditSecretId = ref<string | null>(null)
const auditPage = ref(1)
const auditTotalPages = ref(0)

const snackbar = ref(false)
const snackbarText = ref('')

const filters = reactive({ name: '', tag: '', project_id: '' })
const projectOptions = ref<{ title: string; value: string }[]>([])

let debounceTimer: ReturnType<typeof setTimeout> | null = null

function debouncedFetch() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => fetchSecrets(), 300)
}

async function fetchSecrets() {
  loading.value = true
  try {
    const params: Record<string, any> = { page: page.value, page_size: 20 }
    if (filters.name) params.name = filters.name
    if (filters.tag) params.tag = filters.tag
    if (filters.project_id) params.project_id = filters.project_id
    const resp = await secretApi.list(params)
    secrets.value = resp.data || []
    totalPages.value = resp.meta?.total_pages || 0
  } catch { /* ignore */ } finally {
    loading.value = false
  }
}

async function fetchProjects() {
  try {
    const resp = await projectApi.list({ page_size: 100 })
    projectOptions.value = (resp.data || []).map((p: Project) => ({
      title: p.title,
      value: p.id,
    }))
  } catch { /* ignore */ }
}

function openCreateDialog() {
  editingSecret.value = null
  Object.assign(form, { name: '', value: '', description: '', tags: [], project_id: null })
  showFormValue.value = false
  showFormDialog.value = true
}

function editSecret(secret: SecretBrief) {
  editingSecret.value = secret
  Object.assign(form, {
    name: secret.name,
    value: '',
    description: secret.description,
    tags: [...(secret.tags || [])],
    project_id: secret.project_id || null,
  })
  showFormValue.value = false
  showFormDialog.value = true
}

function closeFormDialog() {
  showFormDialog.value = false
  editingSecret.value = null
}

async function saveSecret() {
  if (!form.name) return
  if (!editingSecret.value && !form.value) return

  saving.value = true
  try {
    const payload: Record<string, any> = {
      name: form.name,
      description: form.description,
      tags: form.tags,
      project_id: form.project_id || undefined,
    }

    if (editingSecret.value) {
      if (form.value) payload.value = form.value
      await secretApi.update(editingSecret.value.id, payload)
    } else {
      payload.value = form.value
      await secretApi.create(payload)
    }
    closeFormDialog()
    fetchSecrets()
  } catch { /* ignore */ } finally {
    saving.value = false
  }
}

async function viewSecret(secret: SecretBrief) {
  viewingSecret.value = secret
  showPlainValue.value = false
  secretValue.value = ''
  showValueDialog.value = true
  try {
    const resp = await secretApi.getValue(secret.id)
    secretValue.value = resp.data?.value || ''
  } catch {
    secretValue.value = '(加载失败)'
  }
}

async function copyValue() {
  try {
    await navigator.clipboard.writeText(secretValue.value)
    snackbarText.value = '密钥值已复制到剪贴板'
    snackbar.value = true
  } catch {
    snackbarText.value = '复制失败'
    snackbar.value = true
  }
}

async function deleteSecret(secret: SecretBrief) {
  if (!confirm(`确定要删除密钥「${secret.name}」吗？`)) return
  try {
    await secretApi.delete(secret.id)
    fetchSecrets()
  } catch { /* ignore */ }
}

function showAuditLogs(secret: SecretBrief) {
  auditSecretId.value = secret.id
  auditTitle.value = `审计日志 — ${secret.name}`
  auditPage.value = 1
  auditLogs.value = []
  showAuditDialog.value = true
  fetchAuditLogs()
}

function showAllAuditLogs() {
  auditSecretId.value = null
  auditTitle.value = '全部密钥审计日志'
  auditPage.value = 1
  auditLogs.value = []
  showAuditDialog.value = true
  fetchAuditLogs()
}

async function fetchAuditLogs() {
  try {
    const params: Record<string, any> = { page: auditPage.value, page_size: 20 }
    let resp
    if (auditSecretId.value) {
      resp = await secretApi.getAuditLogs(auditSecretId.value, params)
    } else {
      resp = await secretApi.getAllAuditLogs(params)
    }
    auditLogs.value = resp.data || []
    auditTotalPages.value = resp.meta?.total_pages || 0
  } catch { /* ignore */ }
}

function getAuditActionLabel(action: string): string {
  const map: Record<string, string> = {
    created: '创建',
    read: '查看',
    updated: '更新',
    deleted: '删除',
    value_read: '读取值',
    listed: '列表',
  }
  return map[action] || action
}

function getAuditActionColor(action: string): string {
  const map: Record<string, string> = {
    created: 'success',
    read: 'info',
    updated: 'warning',
    deleted: 'error',
    value_read: 'primary',
    listed: 'grey',
  }
  return map[action] || 'grey'
}

onMounted(() => {
  fetchSecrets()
  fetchProjects()
})
</script>
