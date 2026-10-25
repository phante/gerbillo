// Comportamenti comuni del Backend (ADR 0017), scritti a mano.
// htmx: anche le risposte con un errore (4xx, 5xx) sostituiscono il bersaglio, così l'esito si vede.
document.addEventListener("htmx:beforeSwap", function (e) {
  if (e.detail.xhr.status >= 400) { e.detail.shouldSwap = true; e.detail.isError = false; }
});

// Widget di stato nella barra (#48): build, memoria vera del container e riavvii, da
// stato.json ogni 5 secondi. Ogni aggiornamento manda anche l'evento gerbillo:stato
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
    fetch("stato.json", { cache: "no-store" }).then(function (r) {
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
  function tutti(radice) { (radice.querySelectorAll ? radice : document).querySelectorAll("select[data-chip]").forEach(chip); }
  // All'apertura della pagina, e per i frammenti che htmx inserisce.
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", function () { tutti(document); });
  else tutti(document);
  htmx.onLoad(tutti);
})();
