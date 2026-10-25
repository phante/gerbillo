import { test, expect } from '@playwright/test';
import { pronta } from './pronta';

async function salvaFonte(page, pid: number, fonte: string) {
  await page.goto(`admin/foto/${pid}`);
  await page.getByLabel('Fonte').fill(fonte);
  await page.getByRole('button', { name: 'Salva' }).click();
  await expect(page.getByRole('status')).toContainText('Salvato: fonte');
}

// #59: Operazioni recenti, modifica e Annulla.
test('modifica e annulla', async ({ page }) => {
  page.on('dialog', (d) => d.accept());
  await salvaFonte(page, 11, 'Da annullare');
  await page.goto('admin/');
  await page.getByRole('link', { name: 'Operazioni recenti' }).click();
  await pronta(page);
  const op = page.locator('article.operazione').first();
  await expect(op).toContainText('Modifica');
  await expect(op).toContainText('Campi: fonte');
  await expect(op.getByRole('link', { name: '11' })).toBeVisible();
  await op.getByRole('button', { name: 'Annulla' }).click();
  await expect(page.locator('article.operazione', { hasText: 'Annullata.' })).toBeVisible();
  await page.reload();
  await expect(page.locator('article.operazione').first()).toContainText('Annullamento');
  await page.goto('admin/foto/11');
  await expect(page.getByLabel('Fonte')).toHaveValue('');
});

test('non annulla se la Foto è cambiata dopo', async ({ page }) => {
  page.on('dialog', (d) => d.accept());
  await salvaFonte(page, 12, 'Prima');
  await salvaFonte(page, 12, 'Dopo');
  await page.goto('admin/operazioni');
  await pronta(page);
  const vecchia = page.locator('article.operazione').nth(1);
  await expect(vecchia).toContainText('12');
  await vecchia.getByRole('button', { name: 'Annulla' }).click();
  const esito = page.getByRole('alert');
  await expect(esito).toContainText('Non annullata');
  await expect(esito.getByRole('link', { name: '12' })).toBeVisible();
  await page.goto('admin/foto/12');
  await expect(page.getByLabel('Fonte')).toHaveValue('Dopo');
});
