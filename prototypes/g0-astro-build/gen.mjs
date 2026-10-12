// PROTOTYPE G0: genera N YAML sintetici di Foto in ./content/foto (wipe me).
import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { stringify } from 'yaml';

const N = Number(process.argv[2] ?? 25000);
const ENTITA = 3000;
const IMPIANTI = 400;
const TEMI = ['Tecnica', 'Meeting', 'Costruzione', 'Scarico', 'Opere di presa', 'Paramento', 'Coronamento', 'Lago', 'Centrale', 'Documenti', 'Storiche', 'Neve', 'Svaso', 'Panorama', 'Interni'];
const REGIONI = ['Piemonte', 'Lombardia', 'Trentino', 'Veneto', 'Friuli', 'Liguria', 'Emilia', 'Toscana', 'Umbria', 'Marche', 'Lazio', 'Abruzzo', 'Molise', 'Campania', 'Puglia', 'Basilicata', 'Calabria', 'Sicilia', 'Sardegna', "Valle d'Aosta"];
const AUTORI = Array.from({ length: 250 }, (_, i) => `utente${i}`);
const LOREM = 'Vista del paramento di valle con lo scarico di superficie in funzione dopo le piogge di novembre; sullo sfondo la casa di guardia e la strada di accesso al coronamento. ';

let seed = 42;
const rnd = () => ((seed = (seed * 1103515245 + 12345) % 2 ** 31) / 2 ** 31);
const pick = (a) => a[Math.floor(rnd() * a.length)];

const dir = new URL('./content/foto/', import.meta.url);
rmSync(dir, { recursive: true, force: true });
mkdirSync(dir, { recursive: true });

for (let pid = 1; pid <= N; pid++) {
  const nEnt = rnd() < 0.85 ? 1 : rnd() < 0.8 ? 2 : 0;
  const entita = Array.from({ length: nEnt }, () => `diga-${Math.floor(rnd() ** 2 * ENTITA)}`);
  const anno = 1900 + Math.floor(rnd() * 125);
  const foto = {
    pid,
    titolo: `Foto ${pid} ${pick(TEMI).toLowerCase()}`,
    descrizione: LOREM.repeat(1 + Math.floor(rnd() * 3)).trim(),
    entita: [...new Set(entita)],
    ...(nEnt === 0 ? { impianto: `impianto-${Math.floor(rnd() * IMPIANTI)}` } : {}),
    regione: pick(REGIONI),
    temi: [...new Set([pick(TEMI), ...(rnd() < 0.3 ? [pick(TEMI)] : [])])],
    datazione: rnd() < 0.7 ? { precisione: 'giorno', valore: `${anno}-0${1 + Math.floor(rnd() * 9)}-1${Math.floor(rnd() * 9)}` } : { precisione: 'decennio', valore: `${anno - (anno % 10)}s` },
    autore: pick(AUTORI),
    contributore: pick(AUTORI),
    ...(rnd() < 0.1 ? { fonte: 'Archivio storico ENEL' } : {}),
    formato: rnd() < 0.97 ? 'immagine' : 'pdf',
    caricata: `20${String(5 + Math.floor(rnd() * 21)).padStart(2, '0')}-06-01`,
    larghezza: 1600,
    altezza: 1200,
  };
  writeFileSync(new URL(`${pid}.yaml`, dir), stringify(foto));
}
console.log(`generati ${N} YAML in ${dir.pathname}`);
