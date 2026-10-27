<template>
  <v-card class="rounded-lg mt-4">
    <v-card-title>MCP 与 OAuth 管理</v-card-title>
    <v-divider />
    <v-card-text>
      <p class="text-body-2 text-medium-emphasis mb-4">仅管理员可修改。设置保存后立即生效，供所有客户端使用。</p>
      <v-progress-linear v-if="loading" indeterminate color="primary" class="mb-4" />
      <v-alert v-if="message" :type="success ? 'success' : 'error'" variant="tonal" class="mb-4" role="status">{{ message }}</v-alert>
      <v-btn v-if="!form && !loading" variant="tonal" @click="load">重新加载设置</v-btn>
      <v-form v-if="form" :disabled="loading || saving" @submit.prevent="save">
        <v-row>
          <v-col cols="12" md="6">
            <h3 class="text-subtitle-1 mb-2">MCP 服务</h3>
            <v-switch v-model="form.enabled" label="启用 MCP 服务" color="primary" hide-details />
            <v-text-field v-model="form.path" label="接入路径" hint="例如 /mcp 或 /agents/mcp；修改后需更新客户端地址，外部代理须转发该路径" persistent-hint class="my-3" />
            <v-text-field v-model="form.external_url" label="外部服务地址" placeholder="https://tasks.example.com" hint="填写不带路径的后端地址；留空时从请求地址生成 API Token 配置" persistent-hint class="mb-3" />
            <v-text-field v-model="form.server_name" label="服务名称" maxlength="100" />
            <v-text-field v-model="form.server_version" label="服务版本" maxlength="50" />
            <v-text-field v-model.number="form.timeout_seconds" label="请求超时（秒）" type="number" :min="1" :max="300" />
          </v-col>
          <v-col cols="12" md="6">
            <h3 class="text-subtitle-1 mb-2">认证方式</h3>
            <v-switch v-model="form.api_token_enabled" label="允许 API Token 接入 MCP" color="primary" hide-details />
            <p class="text-body-2 text-medium-emphasis mb-3">API Token 拥有账号的完整业务权限。关闭此项只影响 MCP 接入。</p>
            <v-switch v-model="form.generic_tools_enabled" label="启用通用 API 代理工具" color="primary" hide-details />
            <v-switch v-model="form.sensitive_tools_enabled" label="允许凭据操作和密钥明文读取工具" color="primary" hide-details />
            <p class="text-body-2 text-medium-emphasis mb-3">两项默认关闭。通用代理仍受敏感操作开关和 OAuth 权限约束。</p>
            <v-switch v-model="form.oauth_enabled" label="启用 OAuth 授权" color="primary" hide-details />
            <p class="text-body-2 text-medium-emphasis mb-3">智能体通过浏览器确认授权，使用独立、可撤销且受权限范围约束的令牌。</p>
            <v-text-field v-model="form.oauth_issuer" label="OAuth 授权服务器地址（issuer）" placeholder="https://tasks.example.com" hint="必须为后端的 HTTPS 来源，不带路径；本机开发允许 HTTP" persistent-hint class="mb-3" />
            <v-switch v-model="form.oauth_dynamic_registration" :disabled="!form.oauth_enabled" label="允许 OAuth 客户端动态注册" color="primary" hide-details />
            <v-alert type="info" variant="tonal" density="compact" class="mt-4">启用 MCP 时须保留至少一种认证方式。关闭 OAuth 会停止令牌认证；变更 issuer 或 MCP 路径后，已有 OAuth 授权需重新建立。</v-alert>
          </v-col>
        </v-row>
        <v-btn type="submit" color="primary" :loading="saving" :disabled="loading">保存 MCP 与认证设置</v-btn>
      </v-form>
      <template v-if="saved">
        <v-divider class="my-5" />
        <h3 class="text-subtitle-1 mb-2">已保存的 MCP 客户端配置</h3>
        <p v-if="!saved.enabled" class="text-body-2">MCP 服务已关闭，启用并保存后可获取配置。</p>
        <template v-else>
          <v-select v-if="saved.oauth_enabled && saved.api_token_enabled" v-model="configMode" :items="configModes" label="客户端认证方式" class="mt-3" />
          <v-textarea :model-value="clientConfig" label="MCP JSON 配置" readonly auto-grow rows="6" class="mt-3" />
          <p v-if="effectiveMode === 'token'" class="text-body-2 mb-3">将占位符替换为“API Token”区域中的个人令牌，请勿分享真实令牌。</p>
          <v-btn variant="tonal" prepend-icon="mdi-content-copy" @click="copyConfig">复制配置</v-btn>
          <p v-if="copyMessage" class="text-body-2 mt-2" role="status">{{ copyMessage }}</p>
        </template>
      </template>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { mcpSettingsApi, type MCPSettings } from '@/api/mcpSettings'
import { copyText } from '@/utils/clipboard'

const form = ref<MCPSettings | null>(null)
const saved = ref<MCPSettings | null>(null)
const loading = ref(false)
const saving = ref(false)
const message = ref('')
const success = ref(false)
const copyMessage = ref('')
const configMode = ref('oauth')
const configModes = [{ title: 'OAuth 浏览器授权', value: 'oauth' }, { title: 'API Token', value: 'token' }]
const effectiveMode = computed(() => saved.value?.oauth_enabled ? (saved.value.api_token_enabled ? configMode.value : 'oauth') : 'token')
const clientConfig = computed(() => {
  const s = saved.value
  if (!s) return ''
  const fallbackOrigin = new URL(import.meta.env.VITE_API_BASE_URL || '/api/v1', window.location.origin).origin
  const endpoint = effectiveMode.value === 'oauth' ? s.oauth_issuer : (s.external_url || fallbackOrigin)
  const entry = effectiveMode.value === 'oauth'
    ? { url: endpoint + s.path }
    : { url: endpoint + s.path, headers: { 'X-API-Token': '<your-api-token>' } }
  return JSON.stringify({ mcpServers: { [s.server_name]: entry } }, null, 2)
})

async function load() {
  loading.value = true
  message.value = ''
  try {
    const response = await mcpSettingsApi.get()
    form.value = { ...response.data }
    saved.value = { ...response.data }
  } catch (error: any) {
    success.value = false
    message.value = error.response?.data?.message || '读取 MCP 设置失败'
  } finally { loading.value = false }
}

async function save() {
  if (!form.value) return
  saving.value = true
  message.value = ''
  copyMessage.value = ''
  try {
    const response = await mcpSettingsApi.save({ ...form.value, timeout_seconds: Number(form.value.timeout_seconds) })
    form.value = { ...response.data }
    saved.value = { ...response.data }
    success.value = true
    message.value = 'MCP 与认证设置已保存并生效'
  } catch (error: any) {
    success.value = false
    message.value = error.response?.data?.message || '保存失败，请检查输入后重试'
  } finally { saving.value = false }
}

async function copyConfig() {
  try { await copyText(clientConfig.value); copyMessage.value = '配置已复制' }
  catch (error: any) { copyMessage.value = error.message || '复制失败，请手动复制' }
}

onMounted(load)
</script>
