import { test, expect, Page } from '@playwright/test';

// #66: nell'anteprima, dopo la password, si sceglie con chi entrare.
test.use({ storageState: { cookies: [], origins: [] } });

async function entra(page: Page, chi: string) {
  await page.goto('admin/');
  await page.getByRole('button', { name: chi, exact: true }).click();
}

test.describe('Scelta dell\'utente', () => {
  test('il Moderatore entra nel Backend, ma non ha le azioni dell\'amministratore', async ({ page }) => {
    await entra(page, 'Moderatore');
    await expect(page.locator('.utente')).toHaveText('moderatore');
    await expect(page.getByRole('button', { name: 'Ricalcola' })).toBeHidden();
  });

  test('l\'amministratore ha anche le azioni sull\'intera Gallery', async ({ page }) => {
    await entra(page, 'Amministratore');
    await expect(page.locator('.utente')).toHaveText('admin');
    await expect(page.getByRole('button', { name: 'Ricalcola' })).toBeVisible();
  });

  for (const [chi, gruppo] of [['Gallery unlimited', 'Gallery unlimited'], ['Utente nuovo', 'NEWLY_REGISTERED']]) {
    test(`${chi} non entra nel Backend`, async ({ page }) => {
      await page.goto('admin/');
      const risposta = page.waitForResponse(r => r.request().method() === 'GET' && r.url().includes('admin/'));
      await page.getByRole('button', { name: chi, exact: true }).click();
      expect((await risposta).status()).toBe(403);
      await expect(page.getByText(gruppo)).toBeVisible();
    });
  }

  test('uscendo si sceglie di nuovo', async ({ page }) => {
    await entra(page, 'Utente nuovo');
    await page.getByRole('link', { name: 'Esci' }).click();
    await page.getByRole('link', { name: 'Rientra nel Backend' }).click();
    await entra(page, 'Moderatore');
    await expect(page.locator('.utente')).toHaveText('moderatore');
  });

  test('la scelta torna alla pagina richiesta', async ({ page }) => {
    await page.goto('admin/foto/8');
    await page.getByRole('button', { name: 'Moderatore', exact: true }).click();
    await expect(page.getByRole('heading', { name: 'Foto 8' })).toBeVisible();
  });
});
