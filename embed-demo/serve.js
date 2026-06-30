// Tiny static server for the embed demo (no deps). Simulates a clinic's website
// on a different origin so the cross-origin JS embed can be tested.
//   node serve.js        # serves ./ on port 8090
//   PORT=9000 node serve.js
const http = require("http");
const fs = require("fs");
const path = require("path");

const port = process.env.PORT || 8090;
const root = __dirname;
const types = { ".html": "text/html", ".js": "application/javascript", ".css": "text/css" };

http
  .createServer((req, res) => {
    let p = decodeURIComponent(req.url.split("?")[0]);
    if (p === "/") p = "/index.html";
    const file = path.join(root, p);
    if (!file.startsWith(root)) {
      res.writeHead(403);
      return res.end("forbidden");
    }
    fs.readFile(file, (err, data) => {
      if (err) {
        res.writeHead(404);
        return res.end("not found");
      }
      res.writeHead(200, { "content-type": types[path.extname(file)] || "text/plain" });
      res.end(data);
    });
  })
  .listen(port, () => console.log("embed demo serving on http://localhost:" + port));
