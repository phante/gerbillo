import { test, expect, Page } from '@playwright/test';

// Il campo di testo di Tom Select per il campo con id dato.
const scrivi = (page: Page, id: string) => page.locator(`#${id}-ts-control`);
const chips = (page: Page, id: string) => page.locator(`#${id}`).locator('xpath=following-sibling::div[contains(@class,"ts-wrapper")]').locator('.chip');

// #53 e #52: campi a chip con i suggerimenti, Categorie dal vocabolario.
test('chip per slug, Tag, Autori e Categorie', async ({ page }) => {
  await page.goto('admin/foto/10');

  // slug: dall'Indice delle Entità, per titolo. È il campo per collegare la Foto a una centrale.
  await expect(page.getByRole('combobox', { name: 'Dighe, centrali e altre pagine del sito' })).toBeVisible();
  await scrivi(page, 'slug').fill('barc');
  await page.locator('.ts-dropdown .option', { hasText: 'dighe/barcis-o-del-cellina' }).first().click();
  await expect(chips(page, 'slug').filter({ hasText: 'Barcis o del Cellina' })).toHaveCount(1);

  // Tag: «ciao come stai» è un Tag solo, nuovo; la virgola ne chiude due.
  const tag = scrivi(page, 'tags');
  await tag.fill('ciao come stai');
  await tag.press('Enter');
  await expect(chips(page, 'tags').filter({ hasText: 'ciao come stai' })).toContainText('nuovo');
  await tag.pressSequentially('uno,due,');
  await expect(chips(page, 'tags').filter({ hasText: /^uno/ })).toHaveCount(1);
  await expect(chips(page, 'tags').filter({ hasText: /^due/ })).toHaveCount(1);
  await chips(page, 'tags').filter({ hasText: /^due/ }).locator('.remove').click();
  await expect(chips(page, 'tags').filter({ hasText: /^due/ })).toHaveCount(0);

  // I tag interni hanno un colore diverso.
  await tag.fill('da-coll');
  await page.locator('.ts-dropdown .option', { hasText: 'da-collegare' }).first().click();
  await expect(chips(page, 'tags').filter({ hasText: 'da-collegare' })).toHaveClass(/chip-interno/);

  // Autori: da quelli già usati. La Foto 10 ne ha già uno.
  await expect(chips(page, 'authors')).toHaveCount(1);
  await scrivi(page, 'authors').fill('Ente di prova');
  await scrivi(page, 'authors').press('Enter');
  await expect(chips(page, 'authors')).toHaveCount(2);

  // Categorie: dal vocabolario, raggruppate per percorso.
  await scrivi(page, 'categories').click();
  await expect(page.locator('.ts-dropdown .optgroup-header', { hasText: 'tecnica' })).toBeVisible();
  await scrivi(page, 'categories').fill('pelton');
  await page.locator('.ts-dropdown .option', { hasText: 'pelton' }).click();

  await page.getByRole('button', { name: 'Salva' }).click();
  await expect(page.getByRole('status')).toContainText('Salvato:');
  for (const campo of ['slug', 'categories', 'tags', 'authors']) {
    await expect(page.getByRole('status')).toContainText(campo);
  }
  // Dopo il salvataggio i chip restano, con i valori salvati.
  await expect(chips(page, 'tags').filter({ hasText: 'ciao come stai' })).toHaveCount(1);
  await expect(chips(page, 'categories').filter({ hasText: 'pelton' })).toHaveCount(1);
});
