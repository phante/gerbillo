import { getCollection } from 'astro:content';

// indice per la ricerca lato client (ADR 0001)
export async function GET() {
  const foto = await getCollection('foto');
  return new Response(JSON.stringify(foto.map(({ data: f }) => [f.pid, f.titolo, f.entita, f.temi, f.autore])));
}
