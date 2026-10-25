import { test, expect } from '@playwright/test';

// #48: il widget nella barra segue il build mentre gira, senza ricaricare la pagina.
test('il widget di stato segue il build', async ({ page }) => {
  await page.goto('admin/foto/9');
  const widget = page.locator('#widget-stato');
  await expect(widget).toContainText(/Mem \d+/);
  await page.getByLabel('Fonte').fill(`Widget ${Date.now()}`);
  await page.getByRole('button', { name: 'Salva' }).click();
  await expect(page.getByRole('status')).toContainText('Salvato');
  await expect(widget).toContainText('Build in corso');
  await expect(widget).toContainText(/Build ✓ \d\d:\d\d/, { timeout: 150_000 });
  // I dettagli restano nella pagina dello stato.
  await widget.getByText(/Mem/).click();
  await expect(page.getByRole('heading', { name: 'Stato del server' })).toBeVisible();
});
