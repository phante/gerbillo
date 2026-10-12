<script>
  let { pid } = $props();
  let fatto = $state(false);
  async function copia() {
    const base = location.origin + '/gallery/f/' + pid;
    await navigator.clipboard.writeText(`[url=${base}/][img]${base}/miniatura.webp[/img][/url]`);
    fatto = true;
  }
</script>

<button onclick={copia}>{fatto ? 'Copiato' : 'Copia per il forum'}</button>
