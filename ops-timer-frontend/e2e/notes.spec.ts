import { test, expect, type Page } from '@playwright/test'

const content = '# 恢复测试\n\n**粗体** 与 *斜体*\n\n```javascript\nconst answer = 42;\nconsole.log(answer);\n```\n\n- [ ] 待办\n\n$x^2 + y^2$\n'

async function api(page: Page, path: string, method = 'GET', data?: unknown) {
  return page.evaluate(async ({ path, method, data }) => {
    const response = await fetch('/api/v1' + path, { method,
      headers: { 'Content-Type': 'application/json', 'X-Taskmgr-Browser': '1' },
      body: data === undefined ? undefined : JSON.stringify(data) })
    if (!response.ok) throw new Error(`${response.status}: ${await response.text()}`)
    return response.status === 204 ? null : response.json()
  }, { path, method, data })
}

test.beforeEach(async ({ page }) => {
  // Match Qt's external resource isolation: all editor dependencies must be local.
  await page.route('**/*', route => {
    const url = new URL(route.request().url())
    return url.hostname === '127.0.0.1' ? route.continue() : route.abort()
  })
  await page.goto('/login')
  await page.getByLabel('用户名', { exact: true }).fill('admin')
  await page.getByLabel('密码', { exact: true }).fill(process.env.TASK_MANAGER_E2E_PASSWORD || 'BrowserTestPassword123')
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await expect(page).toHaveURL(/dashboard/)
  await page.goto('/notes')
  await expect(page.getByRole('button', { name: '新建笔记', exact: true })).toBeVisible()
})

test('offline editor resources, incomplete draft, reload recovery, full copy, failed save and atomic publish', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write'])
  await page.getByRole('button', { name: '新建笔记', exact: true }).click()
  const editor = page.locator('#note-editor')
  const input = editor.locator('.cm-content')
  await input.fill(content)
  await expect(editor.locator('.md-editor-preview .hljs-keyword').first()).toBeVisible()
  await expect(editor.locator('.katex').first()).toBeVisible()
  const icon = editor.locator('.md-editor-toolbar svg').first()
  await expect(icon).toBeVisible()
  expect(await icon.evaluate(svg => {
    const use = svg.querySelector('use')!
    const target = document.querySelector(use.getAttribute('href') || use.getAttribute('xlink:href')!)
    return !!target?.querySelector('path') && getComputedStyle(svg).fill !== 'none'
  })).toBe(true)
  await page.getByRole('button', { name: '保存草稿', exact: true }).click()
  await expect(page.getByText('草稿已保存', { exact: true })).toBeVisible()
  const saved = (await api(page, '/note-drafts')).data.find((draft: any) => draft.content === content)
  expect(saved.title).toBe('')
  expect((await api(page, '/notes')).data).toHaveLength(0)
  await page.getByRole('button', { name: '全文复制', exact: true }).click()
  await expect.poll(() => page.evaluate(async () => (await navigator.clipboard.readText()).replace(/\r\n/g, '\n'))).toBe(content)
  await page.getByRole('button', { name: '关闭', exact: true }).click()
  await page.reload()
  await page.getByRole('button', { name: /草稿箱/ }).click()
  await page.getByRole('button', { name: '恢复', exact: true }).click()
  await expect.poll(() => input.locator('.cm-line').evaluateAll(lines => lines.map(line => line.textContent).join('\n'))).toBe(content)
  await page.getByLabel('标题', { exact: true }).fill('跨客户端草稿')
  await page.route(`**/api/v1/note-drafts/${saved.id}`, route => route.request().method() === 'PUT'
    ? route.fulfill({ status: 500, json: { message: '草稿保存失败' } }) : route.continue())
  await page.getByRole('button', { name: '保存草稿', exact: true }).click()
  await expect(page.getByText('草稿保存失败', { exact: true })).toBeVisible()
  await expect(page.getByLabel('标题', { exact: true })).toHaveValue('跨客户端草稿')
  await page.unroute(`**/api/v1/note-drafts/${saved.id}`)
  await page.getByRole('button', { name: '关闭', exact: true }).click()
  await expect(page.getByText('保留未保存的修改？', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: '继续编辑', exact: true }).click()
  await page.getByRole('button', { name: '存草稿并关闭', exact: true }).click()
  await expect(editor).not.toBeVisible()
  expect((await api(page, '/note-drafts')).data).toHaveLength(1)
  await page.getByRole('button', { name: /草稿箱/ }).click()
  await page.getByRole('button', { name: '恢复', exact: true }).click()
  await page.getByRole('button', { name: '创建', exact: true }).click()
  await expect(page.getByText('笔记已保存，草稿已提交', { exact: true })).toBeVisible()
  const notes = (await api(page, '/notes')).data
  expect(notes[0].content).toBe(content)
  expect((await api(page, '/note-drafts')).data).toHaveLength(0)
  await page.getByRole('button', { name: '复制笔记全文', exact: true }).click()
  await expect.poll(() => page.evaluate(async () => (await navigator.clipboard.readText()).replace(/\r\n/g, '\n'))).toBe(content)
  await page.screenshot({ path: 'test-results/notes-preview.png', fullPage: true })
  await api(page, `/notes/${notes[0].id}`, 'DELETE')
})

test('Markdown list continuation, two-space Tab, Shift+Tab, undo and preview colors', async ({ page }) => {
  await page.getByRole('button', { name: '新建笔记', exact: true }).click()
  const editor = page.locator('#note-editor')
  const input = editor.locator('.cm-content')
  await input.fill('- first')
  await input.press('End')
  await input.press('Enter')
  await page.keyboard.insertText('second')
  await expect.poll(() => input.innerText()).toBe('- first\n- second')
  await input.press('ControlOrMeta+End')
  await input.press('Tab')
  await expect.poll(() => input.innerText()).toBe('- first\n  - second')
  await input.press('Shift+Tab')
  await expect.poll(() => input.innerText()).toBe('- first\n- second')
  await input.fill(content)
  await expect(editor.locator('.md-editor-preview .hljs-keyword').first()).toBeVisible()
  const colors = await editor.locator('.md-editor-preview .hljs-keyword').first().evaluate(token =>
    [getComputedStyle(token).color, getComputedStyle(token.closest('code')!).color])
  expect(colors[0]).not.toBe(colors[1])
  await page.screenshot({ path: 'test-results/notes-editor.png', fullPage: true })
  await page.getByRole('button', { name: '关闭', exact: true }).click()
  await page.getByRole('button', { name: '丢弃修改', exact: true }).click()
})
