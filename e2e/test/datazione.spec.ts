import { test, expect } from '@playwright/test';

// #51: anteprima in italiano mentre si scrive, valori non validi rifiutati, «Usa» da user3.
test('Datazione validata e spiegata', async ({ page }) => {
  await page.goto('admin/foto/18890'); // senza Datazione, user3 «luglio 2011»
  const campo = page.getByLabel('Datazione', { exact: true });
  const anteprima = page.locator('.anteprima-datazione');

  await campo.fill('195X');
  await expect(anteprima).toHaveText("= anni '50");
  await campo.fill('maggio 2006');
  await expect(anteprima).toContainText('= maggio 2006');
  await expect(anteprima).toContainText('si salva come 2006-05');
  await campo.fill('Pippo');
  await expect(anteprima).toContainText('Non valida');
  await page.getByRole('button', { name: 'Salva' }).click();
  await expect(page.getByRole('alert')).toContainText('non valida');

  await page.getByRole('button', { name: '?' }).click();
  await expect(page.locator('#esempi-datazione')).toContainText('1938/1940');

  await expect(page.getByText('Su Coppermine (user3): «luglio 2011»')).toBeVisible();
  await page.getByRole('button', { name: 'Usa 2011-07' }).click();
  await expect(campo).toHaveValue('2011-07');
  await expect(anteprima).toHaveText('= luglio 2011');
  await page.getByRole('button', { name: 'Salva' }).click();
  await expect(page.getByRole('status')).toContainText('Salvato: datazione');
});
