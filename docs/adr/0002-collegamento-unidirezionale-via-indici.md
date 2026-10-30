# Gallery e Sito principale si parlano solo tramite due indici JSON

Corretto il 30/10/2026 (ADR 0019).

La Gallery pubblica è un sito separato dal Sito principale, e ciascuno si costruisce e si rilascia da solo. Si parlano esclusivamente con due file JSON: l'Indice delle Entità (dal Sito principale alla gallery), che porta le pagine del sito con le loro schede, e l'Indice delle Foto (dalla gallery al Sito principale), che elenca le Foto per Slug. Il Backend non scrive mai nel repository del Sito principale.

L'Indice delle Entità è una delle due fonti delle Entità della Gallery, non l'unica: l'altra sono i file di `entita/` (ADR 0019).
