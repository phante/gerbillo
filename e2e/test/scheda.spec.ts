import { test, expect } from '@playwright/test';

// #55: la Foto in alto e le informazioni tecniche, con Originale, Schermo e Mini.
test('la scheda mostra la Foto e i suoi file', async ({ page }) => {
  await page.goto('admin/foto/8');
  const img = page.locator('.foto-scheda img');
  await expect(img).toHaveAttribute('src', /albums\/userpics\/10025\/2005-condotta_forzata\.jpg$/);
  await expect(page.locator('.foto-scheda').getByRole('link', { name: 'Pagina pubblica della Foto' })).toHaveAttribute('href', '../f/8/');

  await page.getByText('Informazioni tecniche').click();
  const tecniche = page.locator('.tecniche');
  await expect(tecniche).toContainText('immagine · 681 × 1024 px');
  await expect(tecniche).toContainText('Anof0A (25)');
  await expect(tecniche).toContainText('27/12/2006');
  await expect(tecniche).toContainText(/Album Coppermine.*\(3\)/);
  const file = tecniche.locator('.file-foto');
  await expect(file.getByRole('link', { name: 'Originale' })).toBeVisible({ timeout: 20_000 });
  await expect(file.getByRole('link', { name: 'Mini' })).toHaveAttribute('href', /thumb_2005-condotta_forzata\.jpg$/);
  // Dimensioni e peso dal file vero, su progettodighe.it.
  await expect(file.locator('tr', { hasText: 'Originale' })).toContainText(/681 × 1024.*kB/);
});
