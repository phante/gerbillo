// Test end-to-end del Backend (ADR 0017), a larghezza telefono. Il server è e2e/avvia.sh:
// richiede dist/ e seme/ di scripts/anteprima-gallery.sh.
import { defineConfig, devices } from '@playwright/test';

const porta = process.env.PORTA || '8091';

export default defineConfig({
  testDir: 'test',
  timeout: 90_000,
  expect: { timeout: 10_000 },
  // I test cambiano lo stesso Repository dei contenuti: uno alla volta.
  workers: 1,
  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: `http://127.0.0.1:${porta}/gerbillo/`,
    httpCredentials: { username: 'gerbillo', password: 'prova' },
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'telefono', use: { ...devices['Pixel 7'] } }],
  webServer: {
    command: './avvia.sh',
    url: `http://127.0.0.1:${porta}/gerbillo/admin/statico/backend.css`,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    env: { PORTA: porta },
  },
});
