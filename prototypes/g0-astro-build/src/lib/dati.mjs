// PROTOTYPE G0: dati per la build a blocchi.
// BLOCCO="da-a" genera le pagine Foto con pid in [da, a); BLOCCO="indici" le pagine aggregate.
import { existsSync, readFileSync } from 'node:fs';
import { parse } from 'yaml';

const blocco = process.env.BLOCCO ?? 'indici';
export const faiIndici = blocco === 'indici';

export function fotoDelBlocco() {
  if (faiIndici) return [];
  const [da, a] = blocco.split('-').map(Number);
  const out = [];
  for (let pid = da; pid < a; pid++) {
    const p = `./content/foto/${pid}.yaml`;
    if (existsSync(p)) out.push(parse(readFileSync(p, 'utf8')));
  }
  return out;
}

// [pid, titolo, entita, temi, autore]
export const indice = () => (faiIndici ? JSON.parse(readFileSync('./indice.json', 'utf8')) : []);
