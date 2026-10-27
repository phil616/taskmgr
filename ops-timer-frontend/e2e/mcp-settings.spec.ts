import { expect, test, type Page } from '@playwright/test'
import type { MCPSettings } from '../src/api/mcpSettings'

async function mockSettings(page: Page, isAdmin = true) {
  let saved: MCPSettings = {
    enabled: true, path: '/mcp', external_url: 'https://tasks.example.com',
    server_name: 'ops-timer-mcp', server_version: '2.0.0', timeout_seconds: 30,
    generic_tools_enabled: false, sensitive_tools_enabled: false, api_token_enabled: true, oauth_enabled: false, oauth_issuer: '', oauth_dynamic_registration: true,
  }
  let requests = 0
  let failSave = false
  await page.route('**/api/v1/**', async route => {
    const path = new URL(route.request().url()).pathname
    let data: unknown = {}
    if (path === '/api/v1/auth/profile') data = { id: 'user-1', username: 'admin', display_name: 'Admin', email: '', is_admin: isAdmin }
    if (path === '/api/v1/auth/token') data = { api_token: 'personal-token-must-not-be-copied' }
    if (path === '/api/v1/auth/smtp-status') data = { enabled: false }
    if (path === '/api/v1/notifications/unread-count') data = { count: 0 }
    if (path === '/api/v1/admin/mcp-settings') {
      requests++
      if (route.request().method() === 'PUT') {
        if (failSave) return route.fulfill({ status: 400, json: { message: '启用 OAuth 时必须设置授权服务器地址' } })
        saved = route.request().postDataJSON()
      }
      data = saved
    }
    return route.fulfill({ json: { code: 0, message: 'success', data } })
  })
  return { settings: () => saved, requests: () => requests, failSave: () => { failSave = true } }
}

test('administrator saves MCP settings, copies saved config, and sees failed saves without losing edits', async ({ page, context }) => {
  const state = await mockSettings(page)
  await context.grantPermissions(['clipboard-read', 'clipboard-write'])
  await page.goto('/settings')
  await expect(page.getByText('MCP 与 OAuth 管理', { exact: true })).toBeVisible()
  await expect(page.getByLabel("启用通用 API 代理工具", { exact: true })).not.toBeChecked()
  await expect(page.getByLabel("允许凭据操作和密钥明文读取工具", { exact: true })).not.toBeChecked()
  const config = page.getByLabel('MCP JSON 配置', { exact: true })
  await expect(config).toHaveValue(/https:\/\/tasks.example.com\/mcp/)
  await page.getByLabel('接入路径', { exact: true }).fill('/agents/mcp')
  await page.getByLabel('服务名称', { exact: true }).fill('agents')
  await page.getByLabel("启用通用 API 代理工具", { exact: true }).check()
  await page.getByLabel("允许凭据操作和密钥明文读取工具", { exact: true }).check()
  // Unsaved edits must not be presented as the currently active configuration.
  await expect(config).toHaveValue(/https:\/\/tasks.example.com\/mcp/)
  await page.getByRole('button', { name: '保存 MCP 与认证设置' }).click()
  await expect(page.getByText('MCP 与认证设置已保存并生效')).toBeVisible()
  expect(state.settings().generic_tools_enabled).toBe(true)
  expect(state.settings().sensitive_tools_enabled).toBe(true)
  await expect(config).toHaveValue(/https:\/\/tasks.example.com\/agents\/mcp/)
  await page.getByRole('button', { name: '复制配置', exact: true }).click()
  await expect(page.getByText('配置已复制', { exact: true })).toBeVisible()
  const copied = await page.evaluate(() => navigator.clipboard.readText())
  expect(JSON.parse(copied).mcpServers.agents.headers['X-API-Token']).toBe('<your-api-token>')
  expect(copied).not.toContain('personal-token-must-not-be-copied')
  await page.getByLabel('启用 OAuth 授权', { exact: true }).check()
  await page.getByLabel('OAuth 授权服务器地址（issuer）', { exact: true }).fill('https://oauth.example.com')
  await page.getByRole('button', { name: '保存 MCP 与认证设置' }).click()
  await expect(config).toHaveValue(/https:\/\/oauth.example.com\/agents\/mcp/)
  await expect(config).not.toHaveValue(/X-API-Token/)
  state.failSave()
  await page.getByLabel('OAuth 授权服务器地址（issuer）', { exact: true }).fill('')
  await page.getByRole('button', { name: '保存 MCP 与认证设置' }).click()
  await expect(page.getByText('启用 OAuth 时必须设置授权服务器地址', { exact: true })).toBeVisible()
  await expect(page.getByLabel('OAuth 授权服务器地址（issuer）', { exact: true })).toHaveValue('')
  await expect(config).toHaveValue(/https:\/\/oauth.example.com\/agents\/mcp/)
  await page.screenshot({ path: 'test-results/mcp-settings-desktop.png', fullPage: true })
  await page.setViewportSize({ width: 390, height: 844 })
  await expect(page.getByRole('button', { name: '保存 MCP 与认证设置' })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await page.screenshot({ path: 'test-results/mcp-settings-mobile.png', fullPage: true })
})

test('ordinary users do not see or request administrator settings', async ({ page }) => {
  const state = await mockSettings(page, false)
  await page.goto('/settings')
  await expect(page.getByText('个人信息', { exact: true })).toBeVisible()
  await expect(page.getByText('MCP 与 OAuth 管理', { exact: true })).toHaveCount(0)
  expect(state.requests()).toBe(0)
})
