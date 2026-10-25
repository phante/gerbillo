import { test, expect } from '@playwright/test';

// #49: dalla home all'elenco filtrato, alla scheda, alla successiva e di nuovo all'elenco.
test('filtro da-collegare, scheda e Foto successiva', async ({ page }) => {
  await page.goto('admin/');
  const voce = page.locator('.cruscotto a', { hasText: 'da-collegare' });
  const n = Number(await voce.locator('.numero').innerText());
  expect(n).toBeGreaterThan(60);
  await voce.click();
  await expect(page).toHaveURL(/foto\/\?s=da-collegare$/);
  await expect(page.locator('main')).toContainText(`${n} Foto`);
  await expect(page.locator('.scelta.attiva')).toContainText('da-collegare');
  await expect(page.locator('.griglia li')).toHaveCount(60);

  // La seconda pagina, poi indietro: il filtro resta nell'URL.
  await page.getByRole('link', { name: 'Successive →' }).click();
  await expect(page).toHaveURL(/s=da-collegare&p=2|p=2&s=da-collegare/);
  await page.goBack();
  await expect(page).toHaveURL(/foto\/\?s=da-collegare$/);

  const prima = page.locator('.griglia li').first();
  const pid = await prima.locator('.pid').innerText();
  const secondo = await page.locator('.griglia li').nth(1).locator('.pid').innerText();
  await prima.locator('a').click();
  await expect(page).toHaveURL(new RegExp(`foto/${pid}\\?s=da-collegare$`));
  await expect(page.getByText(`1 di ${n}`)).toBeVisible();
  await page.getByRole('link', { name: 'Foto successiva' }).click();
  await expect(page).toHaveURL(new RegExp(`foto/${secondo}\\?s=da-collegare$`));
  await page.getByRole('link', { name: '← Elenco' }).click();
  await expect(page).toHaveURL(/foto\/\?s=da-collegare$/);
});

// Un salvataggio che toglie da-collegare fa uscire la Foto dall'elenco a fine build, senza ricaricare a mano.
test('l\'elenco si aggiorna da solo a fine build', async ({ page }) => {
  await page.goto('admin/foto/?s=da-collegare');
  const totale = page.locator('main p b').first();
  const n = Number(await totale.innerText());
  const pid = await page.locator('.griglia li').first().locator('.pid').innerText();

  const scheda = await page.context().newPage();
  await scheda.goto(`admin/foto/${pid}`);
  const tag = scheda.getByLabel('Tag');
  await tag.fill((await tag.inputValue()).split('\n').filter((t) => t !== 'da-collegare').join('\n'));
  await scheda.getByLabel('Pagine del sito (slug)').fill('dighe/vajont');
  await scheda.getByRole('button', { name: 'Salva' }).click();
  await expect(scheda.getByRole('status')).toContainText('Salvato');
  await scheda.close();

  await expect(page.locator('#freschezza')).toContainText('Build in corso');
  await expect(totale).toHaveText(String(n - 1), { timeout: 60_000 });
  await expect(page.locator('.griglia .pid', { hasText: new RegExp(`^${pid}$`) })).toHaveCount(0);
});
