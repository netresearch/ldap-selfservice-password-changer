// Runs tests/companion/index.html in headless Chromium and fails unless the page reports PASS.
// Usage: node tests/companion/run.mjs   (needs `playwright` resolvable, see the companion-checks workflow)
import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { extname, join, normalize, resolve, sep } from "node:path";
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

const server = createServer(async (req, res) => {
  const path = normalize(join(root, decodeURIComponent(req.url.split("?")[0])));
  if (path !== root && !path.startsWith(root + sep)) {
    res.writeHead(403).end();
    return;
  }
  try {
    const file = path.endsWith(sep) ? join(path, "index.html") : path;
    res.writeHead(200, { "Content-Type": types[extname(file)] ?? "application/octet-stream" });
    res.end(await readFile(file));
  } catch {
    res.writeHead(404).end();
  }
});
await new Promise((done) => server.listen(0, "127.0.0.1", done));

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
