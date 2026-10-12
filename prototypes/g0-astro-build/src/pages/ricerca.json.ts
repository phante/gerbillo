import { indice } from '../lib/dati.mjs';

// indice per la ricerca lato client (ADR 0001)
export async function GET() {
  return new Response(JSON.stringify(indice()));
}
