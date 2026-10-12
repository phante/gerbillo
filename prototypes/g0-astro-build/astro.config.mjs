import { defineConfig } from 'astro/config';
import svelte from '@astrojs/svelte';

export default defineConfig({
  base: '/gallery',
  integrations: [svelte()],
});
