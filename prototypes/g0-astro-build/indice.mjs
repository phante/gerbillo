// PROTOTYPE G0: indice leggero di tutte le Foto, letto un YAML alla volta.
// Serve alle pagine aggregate (Entità, Tema, ricerca) senza caricare i metadati completi.
import { readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { parse } from 'yaml';

const dir = './content/foto/';
const righe = [];
for (const nome of readdirSync(dir)) {
  const f = parse(readFileSync(dir + nome, 'utf8'));
  righe.push([f.pid, f.titolo, f.entita, f.temi, f.autore]);
}
righe.sort((a, b) => a[0] - b[0]);
writeFileSync('./indice.json', JSON.stringify(righe));
console.log(`indice: ${righe.length} foto`);
