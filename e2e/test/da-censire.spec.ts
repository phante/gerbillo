import { test, expect } from '@playwright/test';

// #61, ADR 0018: l'Entità da censire 1 di dati/arricchimento/entita/, con le Foto 143, 144 e 145.
test('un\'Entità da censire ha la sua pagina, ed è nell\'albero e nella ricerca', async ({ page }) => {
  await page.goto('da-censire/1/');
  await expect(page.getByRole('heading', { name: 'Verzasca' })).toBeVisible();
  await expect(page.getByText('3 foto · Diga · Ticino, Svizzera · non ancora sul sito')).toBeVisible();
  await expect(page.locator('main ul li')).toHaveCount(3);
  await expect(page.locator('footer')).toContainText('OpenStreetMap');
  await expect(page.locator('footer')).toContainText('Nominatim');

  await page.locator('main ul li a').first().click();
  await expect(page).toHaveURL(/f\/143\/$/);
  await expect(page.getByRole('navigation', { name: 'Percorso di navigazione' }).getByRole('link', { name: 'Verzasca' })).toBeVisible();

  await page.goto('dighe/');
  const voce = page.locator('a[href$="da-censire/1/"]');
  await expect(voce).toContainText('non ancora sul sito');

  await page.goto('cerca/?q=Verzasca');
  await expect(page.locator('#pagine a[href$="da-censire/1/"]')).toBeVisible();

  // Nella Bussola sta sotto «Dove», e sparisce scegliendo un «Cosa».
  await page.goto('');
  const riga = page.locator('#bussola > div', { hasText: 'Dove' });
  const altre = riga.getByRole('button', { name: /^Altre/ });
  if (await altre.count()) await altre.click();
  await riga.getByRole('button', { name: /^Svizzera/ }).click();
  await expect(page.locator('#risultati a[href$="da-censire/1/"]')).toBeVisible();
  await page.getByRole('button', { name: /^Dighe/ }).click();
  await expect(page.locator('#risultati a[href$="da-censire/1/"]')).toHaveCount(0);
});
