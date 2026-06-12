import { expect, test, type Page } from '@playwright/test'
import fs from 'node:fs'

import { csrfToken, login, serverStatePath } from './helpers'

const chooseSelectOption = async (page: Page, testId: string, value: string) => {
  const select = page.getByTestId(testId)
  await select.scrollIntoViewIfNeeded()
  await select.locator('.v-field').click()
  await page.getByRole('option', { name: value, exact: true }).last().click()
}

const createClient = async (page: Page, name: string) => {
  const token = await csrfToken(page)
  const response = await page.request.post('api/save', {
    headers: { 'X-CSRF-Token': token },
    form: {
      object: 'clients',
      action: 'new',
      data: JSON.stringify({
        name,
        enable: true,
        inbounds: [],
        links: [
          { type: 'external', remark: 'external-test', uri: 'https://example.com/subscription' },
        ],
      }),
    },
  })
  const body = await response.json()
  expect(body.success).toBe(true)
}

test('personal ops pack doctor presets delivery and client diagnosis smoke', async ({ page }) => {
  test.setTimeout(90_000)
  const startedAt = Date.now()

  await expect.poll(() => {
    if (!fs.existsSync(serverStatePath)) return false
    return fs.statSync(serverStatePath).mtimeMs >= startedAt - 30_000
  }).toBe(true)

  await login(page)

  await page.goto('')
  await expect(page.getByText('Config Doctor').first()).toBeVisible()
  await page.getByRole('button', { name: 'Run Doctor' }).first().click()
  await expect(page.getByText(/Build sing-box config|Dry config check|sing-box core/).first()).toBeVisible()

  await page.goto('rules')
  await expect(page.getByText('RU/ZH Routing and DNS Presets')).toBeVisible()
  await chooseSelectOption(page, 'preset-proxy-outbound', 'direct')
  await chooseSelectOption(page, 'preset-direct-outbound', 'direct')
  await expect(page.getByText('Preview diff')).toBeVisible()
  await page.getByRole('button', { name: 'Apply to local config' }).click()
  await expect(page.getByText(/rule-set|dns server|experimental.cache_file/).first()).toBeVisible()

  const clientName = `ops-${Date.now()}`
  await createClient(page, clientName)
  await page.goto('clients')
  await expect(page.getByText(clientName)).toBeVisible()
  const row = page.locator('.nexus-data-table__row').filter({ hasText: clientName })

  await row.getByRole('button', { name: 'Diagnose' }).click()
  const diagnosis = page.getByRole('dialog').filter({ hasText: 'Client Diagnosis' })
  await expect(diagnosis).toBeVisible()
  await expect(diagnosis.getByText(/Client enabled|Client inbounds|Subscription formats/).first()).toBeVisible()
  await page.keyboard.press('Escape')

  await row.getByRole('button', { name: 'Config' }).click()
  const delivery = page.getByRole('dialog').filter({ hasText: 'Delivery' })
  await expect(delivery).toBeVisible()
  await expect(delivery.getByRole('tab', { name: 'Sing-box' })).toBeVisible()
  await expect(delivery.getByLabel('Subscription URL')).toBeVisible()
})
