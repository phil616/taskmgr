<template>
  <div>
    <h2 class="text-h5 font-weight-bold mb-4">系统设置</h2>

    <v-row>
      <!-- 左列 -->
      <v-col cols="12" md="6">
        <!-- 个人信息 -->
        <v-card class="rounded-lg mb-4">
          <v-card-title>个人信息</v-card-title>
          <v-divider />
          <v-card-text>
            <v-text-field v-model="profileForm.username" label="用户名" class="mb-1" />
            <v-text-field v-model="profileForm.display_name" label="显示名称" class="mb-1" />
            <v-alert v-if="profileMsg" :type="profileSuccess ? 'success' : 'error'" variant="tonal" density="compact" class="mb-3">
              {{ profileMsg }}
            </v-alert>
            <v-btn color="primary" :loading="savingProfile" @click="saveProfile">保存</v-btn>
          </v-card-text>
        </v-card>

        <!-- 修改密码 -->
        <v-card class="rounded-lg">
          <v-card-title>修改密码</v-card-title>
          <v-divider />
          <v-card-text>
            <v-text-field v-model="passwordForm.old_password" label="当前密码" type="password" class="mb-1" />
            <v-text-field v-model="passwordForm.new_password" label="新密码" type="password" class="mb-1" />
            <v-text-field
              v-model="passwordForm.confirm_password"
              label="确认新密码"
              type="password"
              :rules="[v => v === passwordForm.new_password || '密码不一致']"
              class="mb-1"
            />
            <v-alert v-if="passwordMsg" :type="passwordSuccess ? 'success' : 'error'" variant="tonal" density="compact" class="mb-3">
              {{ passwordMsg }}
            </v-alert>
            <v-btn color="primary" :loading="savingPassword" @click="changePassword">修改密码</v-btn>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- 右列 -->
      <v-col cols="12" md="6">
        <!-- 邮件通知 -->
        <v-card class="rounded-lg mb-4">
          <v-card-title class="d-flex align-center">
            邮件通知
            <v-chip
              size="x-small"
              :color="smtpEnabled ? 'success' : 'default'"
              variant="flat"
              class="ml-2"
            >
              {{ smtpEnabled ? 'SMTP 已配置' : 'SMTP 未配置' }}
            </v-chip>
          </v-card-title>
          <v-divider />
          <v-card-text>
            <div v-if="!smtpEnabled" class="mb-4">
              <v-alert type="info" variant="tonal" density="compact">
                邮件通知未启用。请通过 <code>TASK_MANAGER_SMTP_*</code> 环境变量配置 SMTP。
              </v-alert>
            </div>

            <v-text-field
              v-model="profileForm.email"
              label="通知邮箱"
              placeholder="your@email.com"
              prepend-inner-icon="mdi-email-outline"
              :hint="smtpEnabled ? '系统将在计时单元触发通知时发送邮件至此地址' : '配置 SMTP 后生效'"
              persistent-hint
              :disabled="!smtpEnabled"
              class="mb-3"
            />

            <div class="d-flex ga-3 align-center">
              <v-btn
                color="primary"
                :loading="savingProfile"
                :disabled="!smtpEnabled"
                @click="saveProfile"
              >
                保存邮箱
              </v-btn>
              <v-btn
                variant="tonal"
                color="primary"
                prepend-icon="mdi-email-fast-outline"
                :loading="testingEmail"
                :disabled="!smtpEnabled || !profileForm.email"
                @click="testEmail"
              >
                发送测试邮件
              </v-btn>
            </div>

            <v-alert
              v-if="emailTestMsg"
              :type="emailTestSuccess ? 'success' : 'error'"
              variant="tonal"
              density="compact"
              class="mt-3"
            >
              {{ emailTestMsg }}
            </v-alert>
          </v-card-text>
        </v-card>

        <!-- API Token -->
        <v-card class="rounded-lg">
          <v-card-title>API Token</v-card-title>
          <v-divider />
          <v-card-text>
            <p class="text-body-2 text-medium-emphasis mb-3">
              API Token 可用于脚本和第三方集成调用任务管理器 API。请妥善保管。
            </p>
            <v-text-field
              :model-value="apiToken"
              label="API Token"
              readonly
              :append-inner-icon="showToken ? 'mdi-eye-off' : 'mdi-eye'"
              :type="showToken ? 'text' : 'password'"
              @click:append-inner="toggleShowToken"
            />
            <v-btn color="warning" variant="tonal" :loading="regenerating" @click="regenerateToken">
              重新生成 Token
            </v-btn>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api/auth'

const auth = useAuthStore()
const profileForm = reactive({ username: '', display_name: '', email: '' })
const passwordForm = reactive({ old_password: '', new_password: '', confirm_password: '' })
const apiToken = ref('')
const showToken = ref(false)
const savingProfile = ref(false)
const savingPassword = ref(false)
const regenerating = ref(false)
const testingEmail = ref(false)
const smtpEnabled = ref(false)

const profileMsg = ref('')
const profileSuccess = ref(false)
const passwordMsg = ref('')
const passwordSuccess = ref(false)
const emailTestMsg = ref('')
const emailTestSuccess = ref(false)

async function loadProfile() {
  profileForm.username = auth.user?.username || ''
  profileForm.display_name = auth.user?.display_name || ''
  profileForm.email = auth.user?.email || ''
}

async function loadToken() {
  try {
    const resp = await authApi.getToken()
    apiToken.value = resp.data.api_token
  } catch { /* ignore */ }
}

async function loadSmtpStatus() {
  try {
    const resp = await authApi.smtpStatus()
    smtpEnabled.value = resp.data.enabled
  } catch { /* ignore */ }
}

function toggleShowToken() {
  showToken.value = !showToken.value
}

async function saveProfile() {
  savingProfile.value = true
  profileMsg.value = ''
  try {
    await authApi.updateProfile({
      username: profileForm.username,
      display_name: profileForm.display_name,
      email: profileForm.email,
    })
    await auth.fetchProfile()
    profileMsg.value = '个人信息已保存'
    profileSuccess.value = true
  } catch (e: any) {
    profileMsg.value = e.response?.data?.message || '保存失败'
    profileSuccess.value = false
  } finally {
    savingProfile.value = false
  }
}

async function changePassword() {
  if (passwordForm.new_password !== passwordForm.confirm_password) {
    passwordMsg.value = '两次密码不一致'
    passwordSuccess.value = false
    return
  }
  savingPassword.value = true
  passwordMsg.value = ''
  try {
    await authApi.changePassword(passwordForm.old_password, passwordForm.new_password)
    passwordMsg.value = '密码修改成功'
    passwordSuccess.value = true
    Object.assign(passwordForm, { old_password: '', new_password: '', confirm_password: '' })
  } catch (e: any) {
    passwordMsg.value = e.response?.data?.message || '修改失败'
    passwordSuccess.value = false
  } finally {
    savingPassword.value = false
  }
}

async function testEmail() {
  testingEmail.value = true
  emailTestMsg.value = ''
  try {
    const resp = await authApi.testEmail()
    emailTestMsg.value = resp.data.message
    emailTestSuccess.value = true
  } catch (e: any) {
    emailTestMsg.value = e.response?.data?.message || '测试失败，请检查 SMTP 配置'
    emailTestSuccess.value = false
  } finally {
    testingEmail.value = false
  }
}

async function regenerateToken() {
  if (!confirm('重新生成后，旧 Token 将立即失效。确定继续？')) return
  regenerating.value = true
  try {
    const resp = await authApi.regenerateToken()
    apiToken.value = resp.data.api_token
  } catch { /* ignore */ } finally {
    regenerating.value = false
  }
}

onMounted(() => { loadProfile(); loadToken(); loadSmtpStatus() })
</script>
