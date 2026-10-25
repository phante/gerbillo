// Test end-to-end del Backend (ADR 0017), a larghezza telefono. Il server è e2e/avvia.sh:
// richiede dist/ e seme/ di scripts/anteprima-gallery.sh. Con E2E_URL e E2E_PASSWORD i test
// girano su un Backend già avviato, per esempio l'anteprima:
//   E2E_URL=https://dev.progettodighe.it/gerbillo/ E2E_PASSWORD=… npx playwright test
import { defineConfig, devices } from '@playwright/test';

const porta = process.env.PORTA || '8091';
const remoto = process.env.E2E_URL;

export default defineConfig({
  testDir: 'test',
  timeout: remoto ? 180_000 : 90_000,
  expect: { timeout: 10_000 },
  // I test cambiano lo stesso Repository dei contenuti: uno alla volta.
  workers: 1,
  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: remoto || `http://127.0.0.1:${porta}/gerbillo/`,
    httpCredentials: { username: 'gerbillo', password: process.env.E2E_PASSWORD || 'prova' },
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'telefono', use: { ...devices['Pixel 7'] } }],
  webServer: remoto ? undefined : {
    command: './avvia.sh',
    url: `http://127.0.0.1:${porta}/gerbillo/admin/statico/backend-stile`,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    env: { PORTA: porta },
  },
});
