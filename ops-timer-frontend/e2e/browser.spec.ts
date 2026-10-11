import { test, expect, type Page } from '@playwright/test'

const password = process.env.TASK_MANAGER_E2E_PASSWORD || 'BrowserTestPassword123'
async function login(page: Page, remember = false) {
  await page.goto('/login')
  await page.getByLabel('用户名', { exact: true }).fill('admin')
  await page.getByLabel('密码', { exact: true }).fill(password)
  if (remember) await page.getByRole('checkbox', { name: '记住我（14 天）' }).check()
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await expect(page).toHaveURL(/dashboard/)
}
async function api(page: Page, path: string, method = 'GET', body?: unknown) {
  return page.evaluate(async ({ path, method, body }) => {
    const response = await fetch(`/api/v1${path}`, { method, headers: { 'Content-Type': 'application/json', 'X-Taskmgr-Browser': '1' }, body: body === undefined ? undefined : JSON.stringify(body) })
    if (!response.ok) throw new Error(`API ${path}: ${response.status}`)
    return response.status === 204 ? null : response.json()
  }, { path, method, body })
}

test('remembered login restores, keeps credentials out of JS storage, and logout revokes it', async ({ page, browser }) => {
  await login(page, true)
  const cookies = await page.context().cookies()
  const session = cookies.find(cookie => cookie.name === 'taskmgr_session')!
  expect(session.httpOnly).toBe(true)
  expect(session.sameSite).toBe('Strict')
  expect(session.expires - Date.now() / 1000).toBeGreaterThan(14 * 86400 - 120)
  expect(session.expires - Date.now() / 1000).toBeLessThanOrEqual(14 * 86400)
  expect(await page.evaluate(() => ({ token: localStorage.getItem('token'), user: localStorage.getItem('user'), cookie: document.cookie }))).toEqual({ token: null, user: null, cookie: '' })
  // A fresh browser context has no Pinia state or localStorage, just the persistent cookie.
  const next = await browser.newContext()
  await next.addCookies(cookies.filter(cookie => cookie.expires > 0))
  const restored = await next.newPage()
  await restored.goto('http://127.0.0.1:5177/todos')
  await expect(restored.getByRole('heading', { name: '待办事项', exact: true })).toBeVisible()
  await restored.getByRole('button', { name: '账户菜单' }).click()
  await restored.getByText('退出登录', { exact: true }).click()
  await expect(restored).toHaveURL(/login/)
  await page.reload()
  await expect(page).toHaveURL(/login/)
  await next.close()
})

test('unchecked login uses a session cookie', async ({ page }) => {
  await login(page)
  const cookie = (await page.context().cookies()).find(cookie => cookie.name === 'taskmgr_session')!
  expect(cookie.expires).toBe(-1)
  await page.reload()
  await expect(page).toHaveURL(/dashboard/)
})

test('TODO matrix defaults, overlapping selection, drag persistence, keyboard and rollback', async ({ page }) => {
  await login(page)
  const first = (await api(page, '/todos', 'POST', { title: `坐标测试 A ${Date.now()}` })).data
  const second = (await api(page, '/todos', 'POST', { title: `坐标测试 B ${Date.now()}` })).data
  try {
    expect([first.importance, first.difficulty]).toEqual([0.5, 0.5])
    await page.goto('/todos')
    const select = page.getByLabel('待办评分坐标图').locator('.v-select')
    await select.click()
    await page.getByRole('option', { name: first.title, exact: true }).click()
    const point = page.locator(`.todo-point[data-id="${first.id}"]`)
    await expect(point).toHaveClass(/selected/)
    const plane = await page.locator('.matrix-plane').boundingBox()
    const box = await point.boundingBox()
    if (!plane || !box) throw new Error('matrix not visible')
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
    await page.mouse.down()
    await page.mouse.move(box.x + box.width / 2 + plane.width * .25, box.y + box.height / 2 - plane.height * .25, { steps: 15 })
    await page.mouse.up()
    await expect(page.getByText('位置已保存', { exact: true })).toBeVisible()
    let saved = (await api(page, `/todos/${first.id}`)).data
    expect(saved.importance).toBeCloseTo(.75, 1)
    expect(saved.difficulty).toBeCloseTo(.75, 1)
    expect((await api(page, `/todos/${second.id}`)).data.importance).toBe(.5)
    await page.reload()
    await point.focus()
    await point.press('ArrowRight')
    await point.press('Enter')
    await expect(page.getByText('位置已保存', { exact: true })).toBeVisible()
    saved = (await api(page, `/todos/${first.id}`)).data
    expect(saved.importance).toBeCloseTo(.76, 1)
    await page.route(`**/api/v1/todos/${first.id}`, async route => {
      if (route.request().method() === 'PUT') await route.fulfill({ status: 500, json: { message: 'test failure' } })
      else await route.continue()
    })
    await point.focus()
    await point.press('ArrowDown')
    await point.press('Enter')
    await expect(page.getByText('保存失败，已恢复原位置。请检查网络后重试。')).toBeVisible()
    await expect(point).toHaveAttribute('style', new RegExp(`bottom: ${saved.difficulty * 100}%`))
    await page.unroute(`**/api/v1/todos/${first.id}`)
    await page.screenshot({ path: 'test-results/todo-matrix-desktop.png', fullPage: true })
    await page.setViewportSize({ width: 390, height: 844 })
    await expect(point).toBeVisible()
    await expect.poll(async () => (await page.locator('.matrix-plane').boundingBox())?.width || 0).toBeGreaterThan(240)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
    await page.locator('.matrix-plane').scrollIntoViewIfNeeded()
    const touch = await page.context().newCDPSession(page)
    await touch.send('Emulation.setTouchEmulationEnabled', { enabled: true })
    const mobilePlane = (await page.locator('.matrix-plane').boundingBox())!
    const mobilePoint = (await point.boundingBox())!
    const x = mobilePoint.x + mobilePoint.width / 2
    const y = mobilePoint.y + mobilePoint.height / 2
    await touch.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x, y }] })
    for (let step = 1; step <= 10; step++) {
      await touch.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: x - mobilePlane.width * .01 * step, y: y + mobilePlane.height * .01 * step }] })
    }
    await touch.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] })
    await expect(page.getByText('位置已保存', { exact: true })).toBeVisible()
    const touchSaved = (await api(page, `/todos/${first.id}`)).data
    expect(touchSaved.importance).toBeCloseTo(saved.importance - .1, 1)
    expect(touchSaved.difficulty).toBeCloseTo(saved.difficulty - .1, 1)
    await page.screenshot({ path: 'test-results/todo-matrix-mobile.png', fullPage: true })
  } finally {
    await api(page, `/todos/${first.id}`, 'DELETE').catch(() => {})
    await api(page, `/todos/${second.id}`, 'DELETE').catch(() => {})
  }
})
