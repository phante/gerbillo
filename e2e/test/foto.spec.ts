import { test, expect } from '@playwright/test';
import { pronta } from './pronta';

// #56: login, apertura della Foto, salvataggio di un campo, esito visibile senza ricaricare.
test('salva un campo della Foto senza ricaricare la pagina', async ({ page }) => {
  await page.goto('admin/');
  await expect(page.locator('.utente')).toHaveText('admin');
  // Il nome è Gerbillo; branch e commit dell'anteprima stanno in fondo, come nella Gallery.
  await expect(page.locator('.marchio')).toHaveText('Gerbillo');
  await expect(page).toHaveTitle('Gerbillo · Gerbillo');
  await expect(page.locator('footer.piede-anteprima')).toContainText('[ANTEPRIMA]');
  await page.locator('.barra').getByLabel('Cerca una Foto').fill('8');
  await page.locator('.barra').getByLabel('Cerca una Foto').press('Enter');
  await expect(page).toHaveURL(/admin\/foto\/8$/);
  await pronta(page);
  await expect(page.getByRole('heading', { name: 'Foto 8' })).toBeVisible();

  // Un segno nella pagina: se il salvataggio la ricaricasse, sparirebbe.
  await page.evaluate(() => ((window as any).segno = true));
  const fonte = `Archivio di prova ${Date.now()}`;
  await page.getByLabel('Fonte').fill(fonte);
  await expect(page.getByText('Modifiche non salvate')).toBeVisible();
  await page.getByRole('button', { name: 'Salva' }).click();
  await expect(page.getByRole('status')).toContainText('Salvato: fonte');
  await expect(page.getByLabel('Fonte')).toHaveValue(fonte);
  expect(await page.evaluate(() => (window as any).segno)).toBe(true);
  await expect(page.getByText('Modifiche non salvate')).toBeHidden();

  // Riaprendo la scheda il valore è quello salvato.
  await page.reload();
  await expect(page.getByLabel('Fonte')).toHaveValue(fonte);
  await expect(page.getByText('Importato: (vuoto)').first()).toBeVisible();
});

test('un valore non valido non si salva e l\'errore si vede', async ({ page }) => {
  await page.goto('admin/foto/8');
  await page.getByLabel(/Punto geografico/).fill('46,2672');
  await page.getByRole('button', { name: 'Salva' }).click();
  await expect(page.getByRole('alert')).toContainText('Non salvato');
  await expect(page.getByLabel(/Punto geografico/)).toHaveValue('46,2672');
});
