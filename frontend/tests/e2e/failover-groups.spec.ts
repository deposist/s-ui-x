import { expect, test } from '@playwright/test'

import { login } from './helpers'

// Creates an Auto/Failover group through the outbound editor and confirms it
// lands in the list. The failover engine itself is covered by Go unit/
// integration tests; this only exercises the editor + persistence path.
test('nexus outbounds: create a failover group', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)
  await page.goto('outbounds')

  await page.getByRole('button', { name: 'Add', exact: true }).first().click()
  const drawer = page.getByRole('dialog')
  await expect(drawer).toContainText('Add Outbound')

  // Pick the Failover type; its dedicated editor appears.
  await drawer.getByLabel('Type').click()
  await page.getByRole('option', { name: 'Failover', exact: true }).click()
  await expect(drawer).toContainText('Failover Group')

  const tag = `fo-${Date.now()}`
  await drawer.getByLabel('Tag').fill(tag)

  // Add the primary member (the seeded "direct" outbound is always present).
  await drawer.getByRole('button', { name: 'Add member' }).click()
  await drawer.getByLabel('Primary').click()
  await page.getByRole('option', { name: 'direct', exact: true }).click()

  await drawer.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(page.getByText(tag)).toBeVisible()
  await expect(page.locator('.nexus-drawer.v-navigation-drawer--active')).toHaveCount(0)
})
