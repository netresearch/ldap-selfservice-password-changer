// @ts-check
const Uglify = require("uglify-js");
const FS = require("node:fs");

const base = "./internal/web/static/js/";

const nameCache = {};

const files = FS.readdirSync(base)
  .filter((f) => f.endsWith(".js"))
  .map((f) => base + f);

for (const f of files) {
  const raw = FS.readFileSync(f, "utf8");

  const res = Uglify.minify(raw, {
    mangle: true,
    compress: {},
    module: true,
    toplevel: true,
    sourceMap: false,
    nameCache
  });

  if (res.error) throw new Error(`err: an error occurred during minification of ${f}: ${res.error}`);

  FS.writeFileSync(f, res.code);
}
