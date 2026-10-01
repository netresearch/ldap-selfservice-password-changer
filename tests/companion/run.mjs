// Runs tests/companion/index.html in headless Chromium and fails unless the page reports PASS.
// Usage: node tests/companion/run.mjs   (needs `playwright` resolvable, see the companion-checks workflow)
import { createServer } from "node:http";
import { readFile, readdir } from "node:fs/promises";
import { extname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";

const root = resolve(fileURLToPath(new URL("../..", import.meta.url)));
const types = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".webp": "image/webp",
  ".png": "image/png",
  ".svg": "image/svg+xml"
};

// Only files the fixture needs are served, looked up by URL path in a table built from the
// directory listing, so a request can never name a path outside it.
const served = new Map();
for (const dir of ["tests/companion", "internal/web/static/companion"]) {
  for (const entry of await readdir(join(root, dir), { recursive: true, withFileTypes: true })) {
    if (!entry.isFile() || entry.parentPath.includes("node_modules")) continue;
    const file = join(entry.parentPath, entry.name);
    served.set("/" + relative(root, file).split(sep).join("/"), file);
  }
}
served.set("/internal/web/static/logo.webp", join(root, "internal/web/static/logo.webp"));
served.set("/tests/companion/", served.get("/tests/companion/index.html"));

const server = createServer(async (req, res) => {
  let path;
  try {
    path = decodeURIComponent(req.url.split("?")[0]);
  } catch {
    // A malformed escape such as "/%" throws URIError, which would reject inside the async handler.
    res.writeHead(400).end();
    return;
  }
  const file = served.get(path);
  if (!file) {
    res.writeHead(404).end();
    return;
  }
  res.writeHead(200, { "Content-Type": types[extname(file)] ?? "application/octet-stream" });
  res.end(await readFile(file));
});
await new Promise((done) => server.listen(0, "127.0.0.1", done));

// A malformed escape must be answered with 400 instead of rejecting inside the handler.
const probe = await fetch(`http://127.0.0.1:${server.address().port}/%`);
if (probe.status !== 400) {
  console.error(`FAIL: GET /% answered ${probe.status}, expected 400`);
  process.exit(1);
}

// Software WebGL keeps the check runnable on GPU-less CI runners.
const browser = await chromium.launch({
  args: ["--use-angle=swiftshader", "--enable-unsafe-swiftshader", "--ignore-gpu-blocklist"]
});
let status = "no status";
try {
  const page = await browser.newPage();
  page.on("pageerror", (error) => console.error("pageerror:", error.message));
  await page.goto(`http://127.0.0.1:${server.address().port}/tests/companion/`);
  await page.waitForFunction(() => /^(PASS|FAIL)/.test(document.getElementById("status")?.textContent ?? ""), null, {
    timeout: 150000
  });
  status = await page.locator("#status").innerText();
  for (const row of await page.locator("#results li").allInnerTexts()) console.log(row);
} finally {
  await browser.close();
  server.close();
}
console.log(status);
process.exit(status.startsWith("PASS") ? 0 : 1);
