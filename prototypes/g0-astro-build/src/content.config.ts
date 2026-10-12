import { defineCollection } from 'astro:content';
import { glob } from 'astro/loaders';

// ponytail: niente schema zod, il prototipo misura solo la build
export const collections = {
  foto: defineCollection({ loader: glob({ pattern: '*.yaml', base: './content/foto' }) }),
};
