import { test, expect } from '@playwright/test';

// ADR 0018: dighe/verzasca di dati/arricchimento/entita/, con le Foto 143, 144 e 145. Nella
// Gallery pubblica è una pagina come le altre: solo il link al sito non c'è.
test('un\'Entità senza pagina sul sito è una pagina come le altre', async ({ page }) => {
  await page.goto('dighe/verzasca/');
  await expect(page.getByRole('heading', { name: 'Verzasca' })).toBeVisible();
  await expect(page.getByText('3 foto · Ticino, Svizzera')).toBeVisible();
  await expect(page.locator('main ul li')).toHaveCount(3);
  await expect(page.getByRole('link', { name: /sul sito/ })).toHaveCount(0);
  await expect(page.locator('body')).not.toContainText('non ancora sul sito');
  await expect(page.locator('footer')).toContainText('OpenStreetMap');

  await page.locator('main ul li a').first().click();
  await expect(page).toHaveURL(/f\/143\/$/);
  await expect(page.getByRole('navigation', { name: 'Percorso di navigazione' }).getByRole('link', { name: 'Verzasca' })).toBeVisible();

  await page.goto('dighe/');
  await expect(page.locator('a[href$="dighe/verzasca/"]')).toBeVisible();

  await page.goto('cerca/?q=Verzasca');
  await expect(page.locator('#pagine a[href$="dighe/verzasca/"]')).toBeVisible();

  // Il Backend la propone come le altre: i file seme si spostano in entita/ all'avvio.
  const risposta = await page.request.get('admin/suggerimenti/slug?q=Verzasca');
  expect(await risposta.text()).toContain('dighe/verzasca');
});
