import { test, expect } from '@playwright/test';

// ADR 0019, #74: nella Bussola il Cosa viene dalla sezione dello Slug, e la Gallery pubblica
// non parla solo di dighe né racconta il lavoro da fare.
test('la Bussola distingue dighe e centrali', async ({ page }) => {
  await page.goto('./');
  const cosa = page.locator('#bussola > div').first();
  const risultati = page.locator('#risultati');

  await cosa.getByRole('button', { name: /^Dighe/ }).click();
  await expect(page.locator('#esito')).toContainText('soggett');
  await expect(risultati.locator('a[href*="/dighe/"]').first()).toBeVisible();
  await expect(risultati.locator('a[href*="/centrali/"]')).toHaveCount(0);

  await cosa.getByRole('button', { name: 'Dighe: togli il filtro' }).click();
  await cosa.getByRole('button', { name: /^Centrali/ }).click();
  await expect(risultati.locator('a[href*="/centrali/"]').first()).toBeVisible();
  await expect(risultati.locator('a[href*="/dighe/"]')).toHaveCount(0);

  await expect(page.locator('body')).not.toContainText(/da classificare/i);
  await expect(page.getByRole('link', { name: /Tutti i soggetti/ })).toBeVisible();
  await page.getByRole('link', { name: /Altre foto/ }).click();
  await expect(page.getByRole('heading', { name: 'Altre foto' })).toBeVisible();
  await expect(page.locator('body')).not.toContainText(/da classificare|collegate a una pagina/i);
});
