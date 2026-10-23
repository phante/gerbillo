# Copie dal Sito principale

File copiati byte per byte dal tema del sito (`phante/progettodighe`,
`themes/progettodighe`), come fa il tema Coppermine: Hugo mette un hash nel nome
del CSS, quindi l'URL cambia a ogni build, e la Gallery deve restare in piedi anche
se il sito non risponde (#25).

| Qui | Nel tema del sito |
|---|---|
| `tachyons.css`, `main.css`, `map.css`, `search.css` | `assets/asset/css/` |
| `map.control.verticalexpand.css` | `assets/asset/css/` |
| `mappa.js`, `leaflet.css` | costruiti da `scripts/mappa-sito.sh`: `map*.js` del sito più Leaflet alle versioni del suo `package-lock.json` |
| `PDLogo.svg` | `static/asset/brand/` |
| `icone/cc.svg`, `by.svg`, `sa.svg` | `static/asset/brand/cc/` |
| `icone/<nome>.svg` | `layouts/_partials/svg/simbols/` |
| `icone/<social>.svg` | `layouts/_partials/svg/brands/<social>-simpleicons.org.svg` |

`gerbillo build` unisce i CSS in `statico/stile.css` saltando gli `@import`, poi
aggiunge `../stile.css` con le regole proprie della Gallery. `TestCopieSito`
confronta le copie con il repo del sito, se è clonato accanto a questo
(`../progettodighe`), e si salta altrimenti. Quando divergono si ricopiano.
