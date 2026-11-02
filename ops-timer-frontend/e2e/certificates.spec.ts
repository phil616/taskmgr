import { test, expect, type Page } from '@playwright/test'
import { execFileSync } from 'node:child_process'
import { mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

let dir: string, certificate: Buffer, privateKey: Buffer
const name = `certificate-${Date.now()}.example.com`

async function api(page: Page, path: string, method = 'GET', data?: unknown) {
  return page.evaluate(async ({ path, method, data }) => {
    const response = await fetch('/api/v1' + path, { method,
      headers: { 'Content-Type': 'application/json', 'X-Taskmgr-Browser': '1' },
      body: data === undefined ? undefined : JSON.stringify(data) })
    if (!response.ok) throw new Error(`${response.status}: ${await response.text()}`)
    return response.status === 204 ? null : response.json()
  }, { path, method, data })
}

test.beforeAll(() => {
  dir = mkdtempSync(join(tmpdir(), 'taskmgr-cert-e2e-'))
  execFileSync('openssl', ['req', '-x509', '-newkey', 'ec', '-pkeyopt', 'ec_paramgen_curve:P-256', '-nodes',
    '-keyout', join(dir, 'private.pem'), '-out', join(dir, 'public.pem'), '-days', '2', '-subj', `/CN=${name}`,
    '-addext', `subjectAltName=DNS:${name}`], { stdio: 'ignore' })
  certificate = readFileSync(join(dir, 'public.pem'))
  privateKey = readFileSync(join(dir, 'private.pem'))
})
test.afterAll(() => { if (dir) rmSync(dir, { recursive: true, force: true }) })

test('certificate upload, metadata, download, countdown and attachment associations', async ({ page }) => {
  await page.goto('/login')
  await page.getByLabel('用户名', { exact: true }).fill('admin')
  await page.getByLabel('密码', { exact: true }).fill(process.env.TASK_MANAGER_E2E_PASSWORD || 'BrowserTestPassword123')
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await expect(page).toHaveURL(/dashboard/)
  const project = (await api(page, '/projects', 'POST', { title: 'Certificate attachment E2E' })).data
  let attachmentID = '', unitID = ''
  try {
    await page.goto(`/projects/${project.id}`)
    const panel = page.locator('.v-card').filter({ has: page.getByRole('button', { name: '上传证书', exact: true }) }).first()
    await panel.locator('input[type=file]').setInputFiles({ name: 'public.pem', mimeType: 'application/x-pem-file', buffer: certificate })
    await expect(page.getByText('证书已上传并解析', { exact: true })).toBeVisible()
    const row = page.getByRole('row').filter({ hasText: name })
    await expect(row).toContainText('X.509')
    await row.getByRole('button', { name: '详情', exact: true }).click()
    await expect(page.getByRole('dialog')).toContainText('SHA-256 指纹')
    await expect(page.getByRole('dialog')).toContainText(`CN=${name}`)
    await expect(page.getByRole('dialog')).toContainText('DNS 名称')
    await page.getByRole('button', { name: '关闭', exact: true }).click()
    const saved = (await api(page, `/attachments?owner_type=project&owner_id=${project.id}`)).data[0]
    attachmentID = saved.id
    expect(saved.certificates[0].common_name).toBe(name)
    const downloadPromise = page.waitForEvent('download')
    await row.getByRole('button', { name: '下载', exact: true }).click()
    const download = await downloadPromise
    expect(download.suggestedFilename()).toBe('public.pem')
    const downloaded = await download.path()
    expect(readFileSync(downloaded!)).toEqual(certificate)
    await row.getByRole('button', { name: '到期倒计时', exact: true }).click()
    await expect(page).toHaveURL(/units\//)
    unitID = page.url().split('/units/')[1]
    const unit = (await api(page, `/units/${unitID}`)).data
    expect(new Date(unit.target_time).getTime()).toBe(new Date(saved.certificates[0].expires_at).getTime())
    expect(unit.project_id).toBe(project.id)
    expect(unit.remind_before_days).toEqual([30, 7, 1])
    const unitRow = page.getByRole('row').filter({ hasText: name })
    await expect(unitRow).toContainText('public.pem')
    await page.reload()
    await expect(page.getByRole('row').filter({ hasText: name })).toBeVisible()
    // Private-key attempts do not create another attachment.
    await page.locator('input[type=file]').setInputFiles({ name: 'private.pem', mimeType: 'application/x-pem-file', buffer: privateKey })
    await expect(page.getByText('只接受公钥证书，不能上传私钥或包含私钥的文件', { exact: true })).toBeVisible()
    expect((await api(page, `/attachments?owner_type=unit&owner_id=${unitID}`)).data).toHaveLength(1)
    await page.getByRole('row').filter({ hasText: name }).getByRole('button', { name: '移除关联', exact: true }).click()
    await page.getByRole('button', { name: '确认移除', exact: true }).click()
    await expect(page.getByText('暂无证书', { exact: true })).toBeVisible()
    await page.getByRole('button', { name: '关联已有证书', exact: true }).click()
    await page.getByLabel('选择证书附件', { exact: true }).press('Enter')
    await page.getByRole('option', { name: 'public.pem' }).click()
    await page.getByRole('button', { name: '关联', exact: true }).click()
    await expect(page.getByRole('row').filter({ hasText: name })).toBeVisible()
    // Opening a different countdown while already in UnitDetail must refresh
    // the reused Vue view, rather than leaving the previous unit on screen.
    const host = (await api(page, '/units', 'POST', { title: 'Certificate host unit', type: 'time_countdown', target_time: new Date(Date.now() + 86400000).toISOString() })).data
    try {
      await api(page, `/attachments/${attachmentID}/links`, 'POST', { owner_type: 'unit', owner_id: host.id })
      await page.goto(`/units/${host.id}`)
      await expect(page.getByRole('heading', { name: 'Certificate host unit', exact: true })).toBeVisible()
      await page.getByRole('row').filter({ hasText: name }).getByRole('button', { name: '到期倒计时', exact: true }).click()
      await expect(page).toHaveURL(new RegExp(`/units/${unitID}$`))
      await expect(page.getByRole('heading', { name: `${name} 证书到期`, exact: true })).toBeVisible()
    } finally { await api(page, `/units/${host.id}`, 'DELETE') }
    await page.goto('/certificates')
    await page.getByLabel('搜索文件名、CN、身份或指纹', { exact: true }).fill(name)
    await page.getByRole('button', { name: '搜索', exact: true }).click()
    const managed = page.getByRole('row').filter({ hasText: name })
    await managed.getByRole('button', { name: '到期倒计时', exact: true }).click()
    await expect(page).toHaveURL(new RegExp(`/units/${unitID}$`))
    await page.goto('/certificates')
    await page.getByRole('row').filter({ hasText: name }).getByRole('button', { name: '删除', exact: true }).click()
    await page.getByRole('button', { name: '确认删除', exact: true }).click()
    await expect(page.getByRole('row').filter({ hasText: name })).toHaveCount(0)
    attachmentID = ''
    expect((await api(page, `/units/${unitID}`)).data.id).toBe(unitID)
  } finally {
    if (attachmentID) await api(page, `/attachments/${attachmentID}`, 'DELETE')
    if (unitID) await api(page, `/units/${unitID}`, 'DELETE')
    await api(page, `/projects/${project.id}`, 'DELETE')
  }
})
