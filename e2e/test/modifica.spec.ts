import { test, expect } from '@playwright/test';

// #58: «Modifica» sulla pagina pubblica della Foto, solo per i Moderatori.
test.describe('Modifica dalla Gallery pubblica', () => {
  test('un visitatore anonimo non vede il link', async ({ browser, baseURL }) => {
    const ctx = await browser.newContext({ baseURL, httpCredentials: undefined });
    const page = await ctx.newPage();
    await page.goto('f/8/');
    await expect(page.getByRole('button', { name: /Copia per il forum/ })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Modifica' })).toBeHidden();
    await ctx.close();
  });

  test('un Moderatore lo vede e apre la scheda', async ({ page }) => {
    await page.goto('admin/'); // l'accesso al Backend mette il cookie segnale
    await page.goto('f/8/');
    await page.getByRole('link', { name: 'Modifica' }).click();
    await expect(page).toHaveURL(/admin\/foto\/8$/);
    await expect(page.getByRole('heading', { name: 'Foto 8' })).toBeVisible();
    // Uscendo il link sparisce.
    await page.goto('admin/esci');
    await page.goto('f/8/');
    await expect(page.getByRole('link', { name: 'Modifica' })).toBeHidden();
  });

  test('il cookie messo a mano non dà accesso al Backend', async ({ browser, baseURL }) => {
    const ctx = await browser.newContext({ baseURL, httpCredentials: undefined });
    await ctx.addCookies([{ name: 'gerbillo_moderatore', value: '1', url: baseURL! }]);
    const page = await ctx.newPage();
    await page.goto('f/8/');
    await expect(page.getByRole('link', { name: 'Modifica' })).toBeVisible();
    const r = await page.request.get('admin/foto/8');
    expect(r.status()).toBe(401);
    await ctx.close();
  });
});
