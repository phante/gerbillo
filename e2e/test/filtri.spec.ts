import { test, expect } from '@playwright/test';
import { pronta } from './pronta';

const totale = (page) => page.locator('main p b').first();

// #57: il campo della barra apre la Foto da un pid o da un URL pubblico, e l'elenco da un testo.
test('campo della barra', async ({ page }) => {
  const cerca = async (v: string) => {
    const campo = page.locator('.barra').getByLabel('Cerca una Foto');
    await campo.fill(v);
    await campo.press('Enter');
  };
  await page.goto('admin/');
  await cerca('8');
  await expect(page).toHaveURL(/admin\/foto\/8$/);
  await cerca('https://progettodighe.it/gallery/f/10/');
  await expect(page).toHaveURL(/admin\/foto\/10$/);
  await cerca('condotta forzata');
  await expect(page).toHaveURL(/admin\/foto\/\?q=condotta\+forzata$/);
  await expect(page.getByLabel('Testo in titolo e descrizione')).toHaveValue('condotta forzata');
  expect(Number(await totale(page).innerText())).toBeGreaterThan(0);
});

// Filtri combinati con i conteggi, ordinamento nell'URL, «successiva» che lo segue.
test('filtri, ordinamento e successiva', async ({ page }) => {
  await page.goto('admin/foto/?s=senza-datazione');
  const tutte = Number(await totale(page).innerText());
  await expect(page.locator('details.filtri').nth(1)).not.toHaveAttribute('open', '');
  await pronta(page);
  await page.getByText('Filtri e ordine').click();

  // Formato: documento, con il conteggio che rispetta «senza Datazione».
  await page.locator('#filtro-f + .ts-wrapper .ts-control').click();
  const opzione = page.locator('.ts-dropdown .option', { hasText: /^documento \(\d+\)$/ });
  const n = Number((await opzione.innerText()).match(/\((\d+)\)/)![1]);
  await opzione.click();
  await expect(page).toHaveURL(/f=documento/);
  await pronta(page);
  await expect(page).toHaveURL(/s=senza-datazione/);
  await expect(totale(page)).toHaveText(String(n));
  expect(n).toBeLessThan(tutte);
  await expect(page.getByText('Filtri e ordine (1 attivi)')).toBeVisible();

  // Ordine per pid: resta nell'URL, e la scheda passa alla successiva dello stesso ordine.
  await page.getByLabel('Ordine').selectOption('pid');
  await expect(page).toHaveURL(/o=pid/);
  // I campi vuoti non vanno nell'URL.
  expect(page.url()).not.toMatch(/[?&]q=&/);
  await pronta(page);
  const pid = await page.locator('.griglia .pid').allInnerTexts();
  expect(pid.map(Number)).toEqual([...pid.map(Number)].sort((a, b) => a - b));
  await page.locator('.griglia li a').first().click();
  await expect(page).toHaveURL(/o=pid/);
  await page.getByRole('link', { name: 'Foto successiva' }).click();
  await expect(page).toHaveURL(new RegExp(`foto/${pid[1]}\\?`));
  await page.goBack();
  await page.goBack();
  await expect(page).toHaveURL(/o=pid/);
  await expect(totale(page)).toHaveText(String(n));
});
