<template>
  <div>
    <div class="d-flex align-center mb-4">
      <h2 class="text-h5 font-weight-bold">项目管理</h2>
      <v-spacer />
      <v-btn color="primary" prepend-icon="mdi-plus" @click="showDialog = true">创建项目</v-btn>
    </div>

    <v-row>
      <v-col v-for="project in projects" :key="project.id" cols="12" sm="6" md="4">
        <v-card class="rounded-lg" :style="project.color ? { borderTop: `3px solid ${project.color}` } : {}">
          <v-card-text>
            <div class="d-flex align-center mb-2">
              <v-icon v-if="project.icon" class="mr-2">{{ project.icon }}</v-icon>
              <router-link :to="`/projects/${project.id}`" class="text-subtitle-1 font-weight-bold text-decoration-none text-high-emphasis">
                {{ project.title }}
              </router-link>
              <v-spacer />
              <v-chip size="x-small" :color="getStatusColor(project.status)" variant="flat">
                {{ getStatusLabel(project.status) }}
              </v-chip>
            </div>
            <p class="text-body-2 text-medium-emphasis mb-3" v-if="project.description">
              {{ stripMd(project.description).slice(0, 100) }}{{ stripMd(project.description).length > 100 ? '...' : '' }}
            </p>
            <div class="d-flex ga-4" v-if="project.unit_stats">
              <div class="text-center">
                <div class="text-h6 font-weight-bold text-success">{{ project.unit_stats.active_count }}</div>
                <div class="text-caption text-medium-emphasis">活跃</div>
              </div>
              <div class="text-center">
                <div class="text-h6 font-weight-bold text-info">{{ project.unit_stats.completed_count }}</div>
                <div class="text-caption text-medium-emphasis">完成</div>
              </div>
              <div class="text-center">
                <div class="text-h6 font-weight-bold">{{ project.unit_stats.total_count }}</div>
                <div class="text-caption text-medium-emphasis">总计</div>
              </div>
            </div>
            <!-- 预算进度 -->
            <div v-if="project.max_budget > 0 && project.budget_stats" class="mt-3">
              <div class="d-flex justify-space-between mb-1">
                <span class="text-caption text-medium-emphasis">预算</span>
                <span class="text-caption">
                  ¥{{ fmtAmt(project.budget_stats.total_expense) }} / ¥{{ fmtAmt(project.max_budget) }}
                </span>
              </div>
              <v-progress-linear
                :model-value="Math.min(project.budget_stats.usage_rate * 100, 100)"
                :color="project.budget_stats.usage_rate >= 1 ? 'error' : project.budget_stats.usage_rate >= 0.8 ? 'warning' : 'success'"
                height="6"
                rounded
              />
            </div>
          </v-card-text>
          <v-divider />
          <v-card-actions>
            <v-btn size="small" variant="text" :to="`/projects/${project.id}`">查看详情</v-btn>
            <v-spacer />
            <v-btn icon="mdi-pencil" size="small" variant="text" @click="editProject(project)" />
            <v-btn icon="mdi-delete" size="small" variant="text" color="error" @click="deleteProject(project.id)" />
          </v-card-actions>
        </v-card>
      </v-col>
    </v-row>

    <div v-if="projects.length === 0 && !loading" class="text-center py-12">
      <v-icon size="64" color="grey-lighten-1">mdi-folder-open-outline</v-icon>
      <p class="text-body-1 text-medium-emphasis mt-4">暂无项目</p>
      <v-btn color="primary" class="mt-2" @click="showDialog = true">创建第一个项目</v-btn>
    </div>

    <v-dialog v-model="showDialog" max-width="600" persistent>
      <v-card class="rounded-lg">
        <v-card-title>{{ editingProject ? '编辑项目' : '创建项目' }}</v-card-title>
        <v-divider />
        <v-card-text>
          <v-text-field v-model="form.title" label="项目标题" :rules="[v => !!v || '必填']" />
          <v-textarea v-model="form.description" label="项目介绍 (Markdown)" rows="5" />
          <v-row dense>
            <v-col cols="6">
              <v-text-field v-model="form.color" label="颜色" placeholder="#1565C0" />
            </v-col>
            <v-col cols="6">
              <v-text-field v-model="form.icon" label="图标" placeholder="mdi-folder" />
            </v-col>
          </v-row>
          <v-text-field
            v-model.number="form.max_budget"
            label="最大预算（可选）"
            type="number"
            prefix="¥"
            hint="设置为 0 或留空表示不限预算"
            persistent-hint
          />
        </v-card-text>
        <v-divider />
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="closeDialog">取消</v-btn>
          <v-btn color="primary" :loading="saving" @click="saveProject">{{ editingProject ? '更新' : '创建' }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { Project } from '@/types'
import { projectApi } from '@/api/projects'
import { getStatusColor, getStatusLabel } from '@/utils/time'

const projects = ref<Project[]>([])
const loading = ref(false)
const saving = ref(false)
const showDialog = ref(false)
const editingProject = ref<Project | null>(null)
const form = reactive({ title: '', description: '', color: '', icon: '', max_budget: 0 })

function fmtAmt(n: number) {
  return n.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

/** 去除常见 Markdown 语法符号，用于卡片摘要预览 */
function stripMd(text: string): string {
  return text
    .replace(/#{1,6}\s*/g, '')       // 标题
    .replace(/(\*\*|__)(.*?)\1/g, '$2') // 加粗
    .replace(/(\*|_)(.*?)\1/g, '$2')    // 斜体
    .replace(/`{1,3}[^`]*`{1,3}/g, '') // 代码
    .replace(/!\[.*?\]\(.*?\)/g, '')    // 图片
    .replace(/\[([^\]]+)\]\(.*?\)/g, '$1') // 链接
    .replace(/^[-*+]\s+/gm, '')        // 无序列表
    .replace(/^\d+\.\s+/gm, '')        // 有序列表
    .replace(/^>\s*/gm, '')            // 引用
    .replace(/[-]{3,}/g, '')           // 分割线
    .replace(/\n+/g, ' ')             // 换行转空格
    .trim()
}

async function fetchProjects() {
  loading.value = true
  try {
    const resp = await projectApi.list({ page_size: 100 })
    projects.value = resp.data || []
  } catch { /* ignore */ } finally {
    loading.value = false
  }
}

function editProject(project: Project) {
  editingProject.value = project
  Object.assign(form, { title: project.title, description: project.description, color: project.color, icon: project.icon, max_budget: project.max_budget || 0 })
  showDialog.value = true
}

function closeDialog() {
  showDialog.value = false
  editingProject.value = null
  Object.assign(form, { title: '', description: '', color: '', icon: '', max_budget: 0 })
}

async function saveProject() {
  if (!form.title) return
  saving.value = true
  try {
    if (editingProject.value) {
      await projectApi.update(editingProject.value.id, form)
    } else {
      await projectApi.create(form)
    }
    closeDialog()
    fetchProjects()
  } catch { /* ignore */ } finally {
    saving.value = false
  }
}

async function deleteProject(id: string) {
  if (!confirm('确定要删除此项目吗？')) return
  try { await projectApi.delete(id); fetchProjects() } catch { /* ignore */ }
}

onMounted(fetchProjects)
</script>
