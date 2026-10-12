# PROTOTYPE G0 — build Astro con 25.000 Foto sintetiche (throwaway)

Domanda (issue #8, ADR 0001): la Gallery pubblica in Astro + Svelte regge la dimensione
dell'archivio entro ~5 minuti di build e con memoria compatibile con il server Lightsail?

```sh
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
