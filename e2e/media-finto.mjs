// Server finto dei file di Coppermine per i test end-to-end: al posto di progettodighe.it,
// che la CI non deve interrogare (#93). A ogni percorso risponde con lo stesso PNG di
// 681 × 1024 px: basta l'intestazione, perché la scheda legge solo dimensioni e peso.
import http from 'node:http';
import zlib from 'node:zlib';

function chunk(tipo, dati) {
  const lun = Buffer.alloc(4);
  lun.writeUInt32BE(dati.length);
  const td = Buffer.concat([Buffer.from(tipo), dati]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(zlib.crc32(td));
  return Buffer.concat([lun, td, crc]);
}

const ihdr = Buffer.alloc(13);
ihdr.writeUInt32BE(681, 0);
ihdr.writeUInt32BE(1024, 4);
ihdr[8] = 8; // 8 bit per canale, RGB, il resto a zero
ihdr[9] = 2;
const png = Buffer.concat([
  Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
  chunk('IHDR', ihdr),
  Buffer.alloc(2048), // un peso in kB, come un file vero
]);

const porta = Number(process.env.PORTA_MEDIA || 8092);
http.createServer((req, res) => {
  res.writeHead(200, { 'Content-Type': 'image/png', 'Content-Length': png.length });
  res.end(req.method === 'HEAD' ? undefined : png);
}).listen(porta, '127.0.0.1');
