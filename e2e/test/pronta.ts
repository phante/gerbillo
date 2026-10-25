import { Page } from '@playwright/test';

// pronta aspetta che htmx e Alpine siano partiti (data-pronto, da backend.js). Serve dopo
// una navigazione fatta con un clic o un invio: sull'anteprima gli script arrivano dopo
// l'HTML, e un clic troppo rapido farebbe un POST normale invece di quello di htmx.
export async function pronta(page: Page) {
  await page.locator('html[data-pronto]').waitFor({ state: 'attached' });
}
