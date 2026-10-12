# PROTOTYPE G0 — build Astro con 25.000 Foto sintetiche (throwaway)

Domanda (issue #8, ADR 0001): la Gallery pubblica in Astro + Svelte regge la dimensione
dell'archivio entro ~5 minuti di build e con memoria compatibile con il server Lightsail?

```sh
./profile.sh [N] [MEM] ["./build-blocchi.sh K"]   # build a blocchi, memoria anon vs page cache
./bench.sh [N=25000] [CPU=2] [MEM=2g] [SWAP=MEM]   # richiede podman; SWAP = memoria+swap totale
```

Il sito generato è volutamente realistico ma minimo: pagina per Foto (con isola Svelte
`client:visible`), pagina per Entità, pagine per Tema paginate a 100, indice JSON per la
ricerca lato client. YAML per Foto come da ADR 0004. Astro 7.3.8, Svelte 5.57.2, Node 22.
Il container ha swap disattivato. `picco_MB` è `memory.peak` del cgroup (include page cache).

## Risultati (MacBook, podman applehv, build da zero senza cache `.astro`)

| CPU | Memoria | NODE_OPTIONS              | Esito      | Secondi | Pagine |
|-----|---------|---------------------------|------------|---------|--------|
| 2   | 2 GB    | —                         | ok         | 16      | 26.144 |
| 2   | 1 GB    | —                         | ok         | 21      | 26.144 |
| 1   | 1 GB    | —                         | ok         | 20      | 26.144 |
| 1   | 768 MB  | —                         | OOM (137)  | —       | —      |
| 1   | 768 MB  | --max-old-space-size=512  | OOM (137)  | —       | —      |
| 2   | 512 MB  | —                         | OOM (137)  | —       | —      |
| 1   | 1100 MB | — (disponibile sul server) | ok        | 22      | 26.144 |
| 1   | 900 MB  | — (swap 2 GB)             | ok         | 27      | 26.144 |
| 1   | 768 MB  | — (swap 2 GB)             | OOM (137)  | —       | —      |

Lo swap del container probabilmente non ha effetto: la VM podman non ha swap propria.

## Server reale (Lightsail)

2 vCPU (1 core fisico in hyperthreading, Xeon Platinum 8259CL 2.5 GHz), 1905 MB di RAM,
~1174 MB disponibili, swap 2 GB di cui ~728 MB già in uso.

## Verdetto: G0 superato, con margine di memoria stretto

- **Tempo: superato** con ampio margine (~20–27 s sul Mac). Il core Xeon è più lento:
  stima 1 minuto circa, comunque lontano dai 5 minuti.
- **Memoria: compatibile ma stretta.** La build serve fra 768 e 900 MB, con picco ~1050 MB
  quando ce n'è. Sul server ne sono disponibili ~1170, e lo swap è già usato. Durante la build
  il forum e MySQL possono finire in swap. Limitare l'heap V8 non aiuta.
- Da decidere nelle fasi successive: build sul server in orari tranquilli (`nice`), oppure
  build fuori dal server (CI o macchina del manutentore) e rsync di `dist/`.

## Seconda iterazione: build a blocchi (memoria limitata)

La build unica cresce linearmente: ~340 MB fissi + ~20 MB ogni 1000 Foto
(anon: 5k 476 MB, 25k 733 MB, 50k 1291 MB). Prima o poi il limite fisico arriva.

`build-blocchi.sh` toglie le content collections:

1. `indice.mjs` legge i YAML uno alla volta e scrive `indice.json` (pid, titolo, Entità, Temi, Autore).
2. N esecuzioni di `astro build` con `BLOCCO=da-a`: ognuna legge solo i propri K YAML e genera le pagine Foto.
3. Un'esecuzione `BLOCCO=indici` genera home, Entità, Temi e `ricerca.json` dall'indice leggero.
4. Unione delle cartelle `dist-*` in `dist/`.

| Foto    | K    | Limite | Esito | Secondi | anon max |
|---------|------|--------|-------|---------|----------|
| 25.000  | 5000 | 2 GB   | ok    | 30      | 354 MB   |
| 25.000  | 1000 | 2 GB   | ok    | 57      | 323 MB   |
| 50.000  | 5000 | 2 GB   | ok    | 55      | 353 MB   |
| 50.000  | 2000 | 512 MB | ok    | 84      | 316 MB   |
| 100.000 | 2000 | 512 MB | ok    | 165     | 378 MB   |

La memoria non dipende più dal numero di Foto, salvo l'indice leggero (~60 MB in più fra 25k e 100k).
Il prezzo è il tempo: ~1,6 s ogni 1000 Foto sul Mac, quindi 2–3 volte tanto sul server.
Con i blocchi si può anche rigenerare solo il blocco toccato da una Pubblicazione più gli indici
(non provato).
