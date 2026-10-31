import { test, expect } from '@playwright/test';

// #54: mappa per il Punto geografico, sincronizzata con il campo nei due sensi.
test('Punto geografico dalla mappa e dal campo', async ({ page }) => {
  await page.goto('admin/foto/8'); // dighe/ca-selva, con il Punto della diga
  const campo = page.getByLabel('Punto geografico');
  const mappa = page.locator('.mappa-punto');
  await expect(mappa.locator('.leaflet-tile-pane')).toBeAttached();
  await expect(campo).toHaveValue('');
  await expect(page.locator('.leaflet-interactive')).toBeVisible(); // il Punto ereditato

  // Un tocco sulla mappa mette il pin e scrive il campo.
  await mappa.click({ position: { x: 60, y: 60 } });
  await expect(page.locator('.pin-punto')).toBeVisible();
  await expect(campo).toHaveValue(/^\d+\.\d+, \d+\.\d+$/);

  // Le coordinate incollate spostano il pin.
  const prima = await page.locator('.pin-punto').boundingBox();
  await campo.fill(`46°17'26"N 8°17'55"E`);
  await expect(page.locator('.esito-punto')).toHaveText('= 46.290556, 8.298611');
  await expect.poll(async () => (await page.locator('.pin-punto').boundingBox())?.x).not.toBe(prima?.x);
  await page.getByRole('button', { name: 'Salva' }).click();
  await expect(page.getByRole('status')).toContainText('Salvato: geolocation.coords');
  await expect(campo).toHaveValue('46.290556, 8.298611');

  // «Usa quello della diga» svuota il Punto: la Foto torna a ereditarlo.
  await page.getByRole('button', { name: 'Usa quello della diga' }).click();
  await expect(campo).toHaveValue('');
  await expect(page.locator('.pin-punto')).toHaveCount(0);
  await page.getByRole('button', { name: 'Salva' }).click();
  await expect(page.getByRole('status')).toContainText('Salvato: geolocation.coords');
  await expect(campo).toHaveValue('');
});

// #82: il nome dell'Entità nel tooltip del Punto ereditato è testo, non HTML.
test('Il tooltip del Punto ereditato non interpreta il nome', async ({ page }) => {
  await page.route('**/admin/foto/8', async (route) => {
    const r = await route.fetch();
    await route.fulfill({ response: r, body: (await r.text()).replace(/data-diga="[^"]*"/, 'data-diga="&lt;b&gt;Selva&lt;/b&gt;"') });
  });
  await page.goto('admin/foto/8');
  await page.locator('.leaflet-interactive').hover();
  await expect(page.locator('.leaflet-tooltip')).toHaveText('Punto di <b>Selva</b>');
  await expect(page.locator('.leaflet-tooltip b')).toHaveCount(0);
});
