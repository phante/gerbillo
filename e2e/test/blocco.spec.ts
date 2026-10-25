import { test, expect } from '@playwright/test';

// #50: selezione su più pagine, anteprima obbligatoria, conferma con un commit solo.
test('modifica in blocco di Foto scelte su più pagine', async ({ page }) => {
  await page.goto('admin/foto/?s=da-collegare');
  await page.getByRole('button', { name: 'Seleziona', exact: true }).click();
  const caselle = page.locator('.griglia .casella');
  const pid1 = await page.locator('.griglia .pid').nth(0).innerText();
  const pid2 = await page.locator('.griglia .pid').nth(1).innerText();
  await caselle.nth(0).check();
  await page.locator('.griglia li a').nth(1).click(); // in selezione il tocco sceglie, non apre
  await expect(page.locator('.barra-selezione')).toContainText('2 selezionate');
  await page.getByRole('link', { name: 'Successive →' }).click();
  await expect(page.locator('.barra-selezione')).toContainText('2 selezionate');
  const pid3 = await page.locator('.griglia .pid').nth(0).innerText();
  await caselle.nth(0).check();
  await expect(page.locator('.barra-selezione')).toContainText('3 selezionate');
  await page.getByRole('button', { name: 'Modifica…' }).click();

  await expect(page.getByRole('heading', { name: 'Modifica in blocco di 3 Foto' })).toBeVisible();
  await page.locator('#togli-tags-ts-control').fill('da-coll');
  await page.locator('.ts-dropdown .option', { hasText: 'da-collegare' }).first().click();
  await page.locator('#aggiungi-slug-ts-control').fill('vajont');
  await page.locator('.ts-dropdown .option', { hasText: 'dighe/vajont' }).first().click();
  await page.getByLabel('Cosa fare').nth(1).selectOption('imposta'); // Fonte
  await page.locator('#valore-fonte').fill('Archivio in blocco');
  await page.getByRole('button', { name: 'Anteprima' }).click();

  const anteprima = page.locator('#anteprima-blocco');
  await expect(anteprima).toContainText('Cambieranno 3 Foto su 3');
  await expect(anteprima).toContainText('slug: 3');
  await anteprima.getByText('Le Foto che cambiano').click();
  for (const p of [pid1, pid2, pid3]) await expect(anteprima.getByRole('link', { name: p, exact: true })).toBeVisible();
  await anteprima.getByRole('button', { name: 'Conferma su 3 Foto' }).click();
  await expect(anteprima.getByRole('status')).toContainText('Cambiate 3 Foto con un commit solo');

  await page.goto('admin/operazioni');
  const op = page.locator('article.operazione').first();
  await expect(op).toContainText('In blocco');
  await expect(op).toContainText('3 Foto');
  await page.goto(`admin/foto/${pid2}`);
  await expect(page.getByLabel('Fonte')).toHaveValue('Archivio in blocco');
});

test('tutte le N del filtro, con la conferma esplicita oltre 500', async ({ page }) => {
  await page.goto('admin/foto/?s=da-collegare');
  const n = await page.locator('main p b').first().innerText();
  expect(Number(n)).toBeGreaterThan(500);
  await page.evaluate(() => sessionStorage.clear());
  await page.reload();
  await page.getByRole('button', { name: 'Seleziona', exact: true }).click();
  await page.getByRole('button', { name: `Seleziona tutte le ${n}` }).click();
  await expect(page.locator('.barra-selezione')).toContainText(`${n} selezionate`);
  await page.getByRole('button', { name: 'Modifica…' }).click();
  await expect(page.getByRole('heading', { name: `Modifica in blocco di ${n} Foto` })).toBeVisible();
  await page.getByLabel('Cosa fare').nth(1).selectOption('imposta');
  await page.locator('#valore-fonte').fill('Tutte');
  await page.getByRole('button', { name: 'Anteprima' }).click();
  const anteprima = page.locator('#anteprima-blocco');
  await expect(anteprima).toContainText(`su ${n}`, { timeout: 60_000 });
  await anteprima.getByRole('button', { name: /Conferma su/ }).click();
  await expect(anteprima.getByRole('alert')).toContainText(`scrivi ${n}`);
});
