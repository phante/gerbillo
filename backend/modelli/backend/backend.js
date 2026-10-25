// Comportamenti comuni del Backend (ADR 0017), scritti a mano.
// data-pronto: htmx e Alpine sono partiti. Lo aspettano i test end-to-end: sull'anteprima
// gli script arrivano dopo l'HTML, e un clic troppo rapido farebbe un POST normale.
document.addEventListener("alpine:initialized", function () { document.documentElement.dataset.pronto = ""; });

// htmx: anche le risposte con un errore (4xx, 5xx) sostituiscono il bersaglio, così l'esito si vede.
document.addEventListener("htmx:beforeSwap", function (e) {
  if (e.detail.xhr.status >= 400) { e.detail.shouldSwap = true; e.detail.isError = false; }
});

// Widget di stato nella barra (#48): build, memoria vera del container e riavvii, da
// widget (JSON) ogni 5 secondi. Ogni aggiornamento manda anche l'evento gerbillo:stato
// con i dati, per le pagine che seguono il build.
(function () {
  var w = document.getElementById("widget-stato");
  if (!w) return;
  function voce(testo, classe, titolo) {
    var s = document.createElement("a");
    s.href = "stato";
    s.className = "voce-widget link " + (classe || "");
    s.textContent = testo;
    if (titolo) s.title = titolo;
    return s;
  }
  function aggiorna() {
    fetch("widget", { cache: "no-store" }).then(function (r) {
      if (!r.ok) throw new Error("HTTP " + r.status);
      return r.json();
    }).then(function (s) {
      var v = [], b = s.build;
      if (b) {
        if (b.inCorso) v.push(voce(b.inCoda ? "Build in corso, poi un altro" : "Build in corso", "in-corso"));
        else if (b.esito === "riuscito") v.push(voce("Build ✓ " + b.fine + " (" + b.durata + ")", "", "ultimo build riuscito"));
        else if (b.esito === "fallito") v.push(voce("Build ✗ " + b.fine, "guasto", b.errore));
      }
      var m = s.memoria;
      v.push(voce("Mem " + m.vera + (m.max ? "/" + m.max : "") + " MiB", m.max && m.vera > m.max * 0.85 ? "guasto" : "",
        "memoria vera del container, senza i " + m.cache + " MiB di cache"));
      if (s.riavvii > 0) v.push(voce("⚠ " + s.riavvii + (s.riavvii === 1 ? " riavvio" : " riavvii"), "guasto"));
      w.replaceChildren.apply(w, v);
      document.dispatchEvent(new CustomEvent("gerbillo:stato", { detail: s }));
    }).catch(function () {
      w.replaceChildren(voce("Stato non disponibile", "guasto"));
    });
  }
  aggiorna();
  setInterval(aggiorna, 5000);
})();

// Campi a chip (#53): Tom Select sui select con data-chip, anche quelli che htmx inserisce.
// Invio o virgola chiudono una voce, la × la toglie. slug e Categorie si scelgono soltanto;
// Tag e Autori accettano voci nuove, e un Tag che nessuno usa è segnato «nuovo».
(function () {
  var minimo = { slug: 2, tags: 1, authors: 1 };
  function chip(el) {
    if (el.tomselect || typeof TomSelect === "undefined") return;
    var campo = el.dataset.chip, nuove = campo === "tags" || campo === "authors";
    var opzioni = {
      plugins: ["remove_button"],
      dataAttr: "data-opzione",
      create: nuove ? function (v) { return { value: v, text: v, nuovo: campo === "tags" }; } : false,
      createOnBlur: nuove,
      delimiter: ",",
      persist: false,
      hideSelected: true,
      closeAfterSelect: true,
      maxOptions: 50,
      render: {
        item: function (d, esc) {
          var c = d.interno ? " chip-interno" : d.nuovo ? " chip-nuovo" : "";
          return '<div class="chip' + c + '">' + esc(d.text) + (d.nuovo ? " <small>nuovo</small>" : "") + "</div>";
        },
        option: function (d, esc) {
          return '<div class="' + (d.interno ? "chip-interno" : "") + '">' + esc(d.text) + "</div>";
        },
        option_create: function (d, esc) { return '<div class="create">Aggiungi «' + esc(d.input) + "»</div>"; },
        no_results: function () { return '<div class="no-results">Nessun suggerimento</div>'; },
      },
    };
    if (campo !== "categories") {
      opzioni.load = function (q, fatto) {
        fetch("suggerimenti/" + campo + "?q=" + encodeURIComponent(q))
          .then(function (r) { return r.json(); }).then(fatto).catch(function () { fatto(); });
      };
      opzioni.shouldLoad = function (q) { return q.length >= minimo[campo]; };
      opzioni.searchField = ["text", "value"];
    }
    new TomSelect(el, opzioni);
  }
  // Filtri dell'elenco (#57): una scelta sola, con la ricerca tra le opzioni.
  function scelta(el) {
    if (el.tomselect || typeof TomSelect === "undefined") return;
    new TomSelect(el, { allowEmptyOption: true, maxOptions: 600 }); // il change del select rimanda il modulo
  }
  function tutti(radice) {
    var r = radice.querySelectorAll ? radice : document;
    r.querySelectorAll("select[data-chip]").forEach(chip);
    r.querySelectorAll("select[data-scelta]").forEach(scelta);
  }
  // All'apertura della pagina, e per i frammenti che htmx inserisce.
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", function () { tutti(document); });
  else tutti(document);
  htmx.onLoad(tutti);
})();

// Mappa del Punto geografico (#54): Leaflet di mappa.js (window.L), con gli sfondi del sito
// più uno satellitare. Il campo e il pin si aggiornano a vicenda; i formati del campo li
// interpreta il server (punto?v=…), lo stesso codice del salvataggio.
(function () {
  function arrotonda(x) { return Math.round(x * 1e6) / 1e6; }
  function mappa(div) {
    if (div.mappaPunto || typeof L === "undefined") return;
    var campo = document.getElementById(div.dataset.campo);
    var esito = div.parentNode.querySelector(".esito-punto");
    var ered = div.dataset.ereditato ? div.dataset.ereditato.split(",").map(Number) : null;
    var osm = L.tileLayer("https://tile.openstreetmap.org/{z}/{x}/{y}.png", { maxZoom: 19,
      attribution: '<a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>', referrerPolicy: "strict-origin-when-cross-origin" });
    var sat = L.tileLayer("https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}", { maxZoom: 19,
      attribution: "Esri, Maxar, Earthstar Geographics" });
    var m = L.map(div, { layers: [osm], scrollWheelZoom: false });
    div.mappaPunto = m;
    // La mappa può nascere nascosta (modifica in blocco): si ridisegna quando compare.
    if (window.ResizeObserver) new ResizeObserver(function () { m.invalidateSize(); }).observe(div);
    // Aperto: l'icona di leaflet.css (images/layers.png) non è nel binario.
    L.control.layers({ Mappa: osm, Satellite: sat }, null, { collapsed: false }).addTo(m);
    L.control.scale().addTo(m);
    if (ered) L.circleMarker(ered, { radius: 8, color: "#555", weight: 2, dashArray: "4 3", fillOpacity: 0.1 })
      .addTo(m).bindTooltip("Punto di " + (div.dataset.diga || "della diga"));
    var icona = L.divIcon({ className: "pin-punto", html: "<span></span>", iconSize: [26, 26], iconAnchor: [13, 30] });
    var pin = null;
    function metti(lat, lon, centra) {
      if (!pin) {
        pin = L.marker([lat, lon], { draggable: true, icon: icona, title: "Punto geografico della Foto" }).addTo(m);
        pin.on("dragend", function () { var p = pin.getLatLng(); scrivi(p.lat, p.lng); });
      } else pin.setLatLng([lat, lon]);
      if (centra) m.setView([lat, lon], Math.max(m.getZoom() || 0, 14));
    }
    function togli() { if (pin) { m.removeLayer(pin); pin = null; } }
    function scrivi(lat, lon) {
      campo.value = arrotonda(lat) + ", " + arrotonda(lon);
      esito.textContent = "";
      campo.dispatchEvent(new Event("change", { bubbles: true })); // il modulo segna la modifica
    }
    m.on("click", function (e) { metti(e.latlng.lat, e.latlng.lng); scrivi(e.latlng.lat, e.latlng.lng); });
    var attesa;
    function leggi(centra) {
      var v = campo.value.trim();
      if (!v) { togli(); esito.textContent = ered ? "Vuoto: la Foto eredita il Punto della diga." : ""; return; }
      fetch("punto?v=" + encodeURIComponent(v)).then(function (r) { return r.json(); }).then(function (p) {
        if (p.errore) { esito.textContent = "Non valido: non si salva."; esito.classList.add("dark-red"); return; }
        esito.classList.remove("dark-red");
        esito.textContent = p.testo === v ? "" : "= " + p.testo;
        metti(p.lat, p.lon, centra);
      });
    }
    campo.addEventListener("input", function () { clearTimeout(attesa); attesa = setTimeout(function () { leggi(true); }, 400); });
    var usa = div.parentNode.querySelector(".usa-diga");
    if (usa && ered) usa.addEventListener("click", function () {
      campo.value = ""; togli(); leggi(false);
      campo.dispatchEvent(new Event("change", { bubbles: true }));
      m.setView(ered, 14);
    });
    if (campo.value.trim()) {
      var p = campo.value.split(",").map(Number);
      if (p.length === 2 && !isNaN(p[0]) && !isNaN(p[1])) { metti(p[0], p[1]); m.setView(p, 15); } else leggi(true);
    } else if (ered) m.setView(ered, 14);
    else m.setView([45.8, 10.5], 6);
  }
  function tutte(radice) { (radice.querySelectorAll ? radice : document).querySelectorAll(".mappa-punto").forEach(mappa); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", function () { tutte(document); });
  else tutte(document);
  htmx.onLoad(tutte);
})();

// Selezione multipla nell'elenco (#50): «Seleziona» mette la griglia in modalità selezione.
// La selezione resta cambiando pagina (sessionStorage, per filtro); «Seleziona tutte le N»
// prende tutte le Foto del filtro, anche delle altre pagine. «Modifica…» apre la modifica
// in blocco con l'elenco dei pid.
document.addEventListener("alpine:init", function () {
  Alpine.data("selezione", function (filtro, totale) {
    var chiave = "gerbillo:selezione:" + filtro;
    function leggi() { try { return JSON.parse(sessionStorage.getItem(chiave)) || {}; } catch (e) { return {}; } }
    var s = leggi();
    return {
      modo: !!s.modo, tutte: !!s.tutte, scelte: s.scelte || [], totale: totale, filtro: filtro,
      salva: function () {
        try { sessionStorage.setItem(chiave, JSON.stringify({ modo: this.modo, tutte: this.tutte, scelte: this.scelte })); } catch (e) {}
      },
      get quante() { return this.tutte ? this.totale : this.scelte.length; },
      scelta: function (pid) { return this.tutte || this.scelte.indexOf(pid) >= 0; },
      cambia: function (pid) {
        if (this.tutte) return;
        var i = this.scelte.indexOf(pid);
        if (i >= 0) this.scelte.splice(i, 1); else this.scelte.push(pid);
        this.salva();
      },
      apri: function () { this.modo = true; this.salva(); },
      tutteLe: function () { this.tutte = true; this.salva(); },
      annulla: function () { this.modo = false; this.tutte = false; this.scelte = []; this.salva(); },
    };
  });
});

// Tornando indietro, il browser può ridare la pagina dei filtri com'era al momento
// dell'invio, con i campi vuoti disabilitati: si riabilitano.
window.addEventListener("pageshow", function (e) {
  if (e.persisted) document.querySelectorAll(".filtri-modulo [disabled]").forEach(function (x) { x.disabled = false; });
});
