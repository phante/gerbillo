// Comportamenti comuni del Backend (ADR 0017), scritti a mano.
// htmx: anche le risposte con un errore (4xx, 5xx) sostituiscono il bersaglio, così l'esito si vede.
document.addEventListener("htmx:beforeSwap", function (e) {
  if (e.detail.xhr.status >= 400) { e.detail.shouldSwap = true; e.detail.isError = false; }
});
