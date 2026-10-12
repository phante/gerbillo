# PROTOTYPE G0 — build Astro con 25.000 Foto sintetiche (throwaway)

Domanda (issue #8, ADR 0001): la Gallery pubblica in Astro + Svelte regge la dimensione
dell'archivio entro ~5 minuti di build e con memoria compatibile con il server Lightsail?

```sh
./bench.sh [N=25000] [CPU=2] [MEM=2g]   # richiede podman
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

## Verdetto

- **Tempo: gate superato con ampio margine** (~20 s contro 5 min). Anche una CPU del server
  2–3 volte più lenta resta sotto il minuto.
- **Memoria: serve ~1 GB libero durante la build.** Sotto i 768 MB il processo viene
  ucciso; limitare l'heap V8 non aiuta, quindi la memoria non è old-space JS.
- Su un piano Lightsail da 2 GB condiviso con forum, MySQL e Backend il margine è stretto;
  su 1 GB la build sul server non è praticabile. Alternativa: build fuori dal server
  (CI o macchina del manutentore) e rsync di `dist/` (solo HTML, le Varianti stanno fuori).
