import { defineConfig } from 'astro/config';
import svelte from '@astrojs/svelte';

export default defineConfig({
  base: '/gallery',
  outDir: process.env.OUT ?? './dist',
  integrations: [svelte()],
});
