<template>
  <v-card class="rounded-lg mb-4">
    <v-card-title class="d-flex align-center flex-wrap ga-2">
      <v-icon class="mr-1">mdi-certificate-outline</v-icon>
      {{ owner ? '证书附件' : '证书管理' }}
      <v-spacer />
      <v-btn v-if="owner" variant="text" size="small" @click="openLink">关联已有证书</v-btn>
      <v-btn color="primary" prepend-icon="mdi-upload" :loading="uploading" @click="fileInput?.click()">上传证书</v-btn>
      <input ref="fileInput" type="file" hidden accept=".pem,.crt,.cer,.der,.asc,.gpg,.pgp" @change="upload" />
    </v-card-title>
    <v-card-text>
      <p class="text-body-2 text-medium-emphasis mb-3">支持 X.509（PEM / DER、证书链）和 GPG 公钥证书，单文件最多 4 MiB。仅提取证书信息，不接受私钥。</p>
      <v-alert v-if="error" type="error" closable class="mb-3" @click:close="error = ''">{{ error }}</v-alert>
      <v-alert v-if="success" type="success" closable class="mb-3" @click:close="success = ''">{{ success }}</v-alert>
      <div class="d-flex flex-wrap ga-3">
        <v-text-field v-model="search" label="搜索文件名、CN、身份或指纹" prepend-inner-icon="mdi-magnify" clearable hide-details density="compact" @keyup.enter="reload" @click:clear="reload" />
        <v-select v-model="type" :items="typeItems" label="证书类型" hide-details density="compact" style="max-width: 180px" @update:model-value="reload" />
        <v-btn variant="tonal" @click="reload">搜索</v-btn>
      </div>
    </v-card-text>
    <v-progress-linear v-if="loading" indeterminate />
    <v-table>
      <thead><tr><th>文件 / 身份</th><th>类型</th><th>到期时间</th><th>剩余时间</th><th>操作</th></tr></thead>
      <tbody>
        <template v-for="attachment in attachments" :key="attachment.id">
          <tr v-for="(cert, index) in attachment.certificates" :key="cert.id">
            <td class="py-3" style="max-width: 320px; overflow-wrap: anywhere">
              <div class="font-weight-medium">{{ cert.common_name || cert.subject || '未命名证书' }}</div>
              <div class="text-caption text-medium-emphasis">{{ attachment.filename }}{{ attachment.certificates.length > 1 ? ` · ${index + 1}/${attachment.certificates.length}` : '' }}</div>
            </td>
            <td>{{ cert.type === 'x509' ? 'X.509' : 'GPG' }}</td>
            <td class="text-no-wrap">{{ cert.expires_at ? formatDateTime(cert.expires_at) : '永不过期' }}</td>
            <td class="text-no-wrap"><v-chip :color="state(cert).color" size="small">{{ state(cert).text }}</v-chip></td>
            <td class="py-2">
              <div class="d-flex flex-wrap ga-1">
                <v-btn size="small" variant="text" @click="selected = { attachment, cert }">详情</v-btn>
                <v-btn size="small" variant="text" :disabled="!cert.expires_at || cert.revoked" :loading="busy === cert.id" @click="countdown(cert)">到期倒计时</v-btn>
                <template v-if="index === 0">
                  <v-btn size="small" variant="text" @click="download(attachment)">下载</v-btn>
                  <v-btn size="small" variant="text" color="error" @click="pendingDelete = attachment">{{ owner ? '移除关联' : '删除' }}</v-btn>
                </template>
              </div>
            </td>
          </tr>
        </template>
        <tr v-if="!loading && !attachments.length"><td colspan="5" class="text-center text-medium-emphasis py-8">暂无证书</td></tr>
      </tbody>
    </v-table>
    <v-pagination v-if="totalPages > 1" v-model="page" :length="totalPages" :total-visible="5" @update:model-value="load" />
    <v-dialog :model-value="!!selected" max-width="800" @update:model-value="selected = null">
      <v-card v-if="selected" title="证书详情">
        <v-card-text>
          <v-alert type="info" variant="tonal" class="mb-4">此处展示证书声明的信息，不代表已验证其信任链。GPG 到期时间取自主公钥的自签名。</v-alert>
          <dl class="certificate-details">
            <template v-for="field in detailFields" :key="field.label"><dt>{{ field.label }}</dt><dd>{{ field.value || '—' }}</dd></template>
          </dl>
        </v-card-text>
        <v-card-actions><v-spacer /><v-btn @click="selected = null">关闭</v-btn></v-card-actions>
      </v-card>
    </v-dialog>
    <v-dialog :model-value="!!pendingDelete" max-width="440" @update:model-value="pendingDelete = null">
      <v-card :title="owner ? '移除证书关联' : '删除证书附件'">
        <v-card-text>{{ owner ? '从当前对象移除关联，证书仍保留在证书管理中。' : '将删除原始证书文件及所有附件关联。已创建的倒计时会保留。' }}</v-card-text>
        <v-card-actions><v-spacer /><v-btn @click="pendingDelete = null">取消</v-btn><v-btn color="error" :loading="deleting" @click="remove">确认{{ owner ? '移除' : '删除' }}</v-btn></v-card-actions>
      </v-card>
    </v-dialog>
    <v-dialog v-model="linkDialog" max-width="600">
      <v-card title="关联已有证书">
        <v-card-text>
          <v-text-field v-model="linkSearch" label="搜索已有证书" @keyup.enter="loadChoices" />
          <v-btn variant="tonal" class="mb-3" :loading="choicesLoading" @click="loadChoices">搜索</v-btn>
          <v-select v-model="linkID" :items="choices" item-title="filename" item-value="id" label="选择证书附件" :loading="choicesLoading" />
          <p class="text-caption text-medium-emphasis">最多显示 100 个结果，可搜索缩小范围。</p>
          <v-alert v-if="linkError" type="error" class="mt-3">{{ linkError }}</v-alert>
        </v-card-text>
        <v-card-actions><v-spacer /><v-btn @click="linkDialog = false">取消</v-btn><v-btn color="primary" :disabled="!linkID" :loading="linking" @click="link">关联</v-btn></v-card-actions>
      </v-card>
    </v-dialog>
  </v-card>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { certificateApi, type AttachmentOwner, type Certificate, type CertificateAttachment } from '@/api/certificates'
import { formatDateTime, formatDuration } from '@/utils/time'

const props = defineProps<{ owner?: AttachmentOwner }>()
const router = useRouter()
const fileInput = ref<HTMLInputElement | null>(null)
const attachments = ref<CertificateAttachment[]>([])
const search = ref(''), type = ref(''), page = ref(1), totalPages = ref(0)
const loading = ref(false), uploading = ref(false), deleting = ref(false), busy = ref('')
const error = ref(''), success = ref('')
const now = ref(Date.now())
let ticker: ReturnType<typeof setInterval> | undefined
let requestVersion = 0
const selected = ref<{ attachment: CertificateAttachment; cert: Certificate } | null>(null)
const pendingDelete = ref<CertificateAttachment | null>(null)
const typeItems = [{ title: '全部', value: '' }, { title: 'X.509', value: 'x509' }, { title: 'GPG', value: 'gpg' }]
const linkDialog = ref(false), linkID = ref(''), linkSearch = ref(''), linkError = ref('')
const choices = ref<CertificateAttachment[]>([]), choicesLoading = ref(false), linking = ref(false)
const message = (e: any) => e.response?.data?.message || '操作失败，请重试'

function state(c: Certificate) {
  if (c.revoked) return { color: 'error', text: '已撤销' }
  if (new Date(c.not_before).getTime() > now.value) return { color: 'info', text: '尚未生效' }
  if (!c.expires_at) return { color: 'success', text: '永不过期' }
  const seconds = (new Date(c.expires_at).getTime() - now.value) / 1000
  if (seconds <= 0) return { color: 'error', text: '已过期' }
  return { color: seconds <= 30 * 86400 ? 'warning' : 'success', text: formatDuration(seconds) }
}

const detailFields = computed(() => {
  if (!selected.value) return []
  const { attachment: a, cert: c } = selected.value
  return [
    { label: '原始文件', value: `${a.filename} (${a.size} 字节)` },
    { label: '类型', value: c.type === 'x509' ? 'X.509' : 'GPG / OpenPGP' },
    { label: 'CN / 名称', value: c.common_name }, { label: '主体', value: c.subject },
    { label: '签发者', value: c.issuer }, { label: '序列号 / Key ID', value: c.serial_number },
    { label: c.type === 'x509' ? 'SHA-256 指纹' : 'OpenPGP 指纹', value: c.fingerprint },
    { label: '公钥算法', value: c.algorithm }, { label: 'DNS 名称', value: c.dns_names?.join(', ') },
    { label: '邮箱', value: c.emails?.join(', ') }, { label: '用户 ID', value: c.user_ids?.join('\n') },
    { label: '生效 / 创建时间', value: formatDateTime(c.not_before) },
    { label: '到期时间', value: c.expires_at ? formatDateTime(c.expires_at) : '永不过期' },
    { label: '状态', value: state(c).text },
  ]
})

async function load() {
  const version = ++requestVersion
  loading.value = true; error.value = ''
  try {
    const resp = await certificateApi.list({ ...props.owner, q: search.value || undefined, type: type.value || undefined, page: page.value, page_size: 20 })
    if (version !== requestVersion) return
    attachments.value = resp.data || []; totalPages.value = resp.meta?.total_pages || 0
    if (page.value > 1 && page.value > totalPages.value) { page.value = Math.max(1, totalPages.value); await load() }
  } catch (e) { if (version === requestVersion) error.value = message(e) }
  finally { if (version === requestVersion) loading.value = false }
}
function reload() { page.value = 1; return load() }
async function upload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  error.value = ''; success.value = ''
  if (file.size > 4 * 1024 * 1024 || file.size === 0) { error.value = '证书文件不能为空且不能超过 4 MiB'; input.value = ''; return }
  uploading.value = true
  try { await certificateApi.upload(file, props.owner); search.value = ''; type.value = ''; await reload(); success.value = '证书已上传并解析' }
  catch (e) { error.value = message(e) }
  finally { uploading.value = false; input.value = '' }
}
async function download(a: CertificateAttachment) {
  try { await certificateApi.download(a) } catch (e) { error.value = message(e) }
}
async function countdown(c: Certificate) {
  busy.value = c.id; error.value = ''
  try { const resp = await certificateApi.countdown(c.id); await router.push(`/units/${resp.data.id}`) }
  catch (e) { error.value = message(e) }
  finally { busy.value = '' }
}
async function remove() {
  if (!pendingDelete.value) return
  deleting.value = true; error.value = ''
  try {
    if (props.owner) await certificateApi.unlink(pendingDelete.value.id, props.owner)
    else await certificateApi.delete(pendingDelete.value.id)
    pendingDelete.value = null; await load()
  } catch (e) { pendingDelete.value = null; error.value = message(e) }
  finally { deleting.value = false }
}
function openLink() { linkDialog.value = true; linkID.value = ''; linkError.value = ''; linkSearch.value = ''; loadChoices() }
async function loadChoices() {
  choicesLoading.value = true; linkError.value = ''
  try { choices.value = (await certificateApi.list({ q: linkSearch.value || undefined, page_size: 100 })).data || [] }
  catch (e) { linkError.value = message(e) }
  finally { choicesLoading.value = false }
}
async function link() {
  if (!props.owner || !linkID.value) return
  linking.value = true; linkError.value = ''
  try { await certificateApi.link(linkID.value, props.owner); linkDialog.value = false; await reload() }
  catch (e) { linkError.value = message(e) }
  finally { linking.value = false }
}
watch(() => [props.owner?.owner_id, props.owner?.owner_type], () => { attachments.value = []; reload() })
onMounted(() => { load(); ticker = setInterval(() => { now.value = Date.now() }, 1000) })
onUnmounted(() => { if (ticker) clearInterval(ticker); requestVersion++ })
</script>

<style scoped>
.certificate-details { display: grid; grid-template-columns: minmax(100px, 140px) minmax(0, 1fr); gap: 12px; }
.certificate-details dt { opacity: 0.7; }
.certificate-details dd { overflow-wrap: anywhere; white-space: pre-wrap; }
@media (max-width: 500px) { .certificate-details { grid-template-columns: 1fr; gap: 6px; } .certificate-details dd { margin-bottom: 10px; } }
</style>
