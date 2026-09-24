// Writes the license notices of the npm packages bundled into ../bundle.js into
// _EXTRA_CREDITS, which the release appends to the CREDITS gocredits writes for the Go
// modules. gocredits knows nothing of what is bundled here, and the bundle is embedded in
// every binary and written into every page it renders.
//
// An entry this script writes is the one whose URL is the package's page on npm, so each
// run removes those and writes them again from what is bundled now. Running it twice
// leaves the file as the first run did, and the entry of a package the bundle no longer
// takes is gone after the next run. The entries written by hand are kept byte for byte.
import { readFile, readdir, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { build } from "esbuild";

const creditsPath = new URL("../../../_EXTRA_CREDITS", import.meta.url);
const npmURL = "https://www.npmjs.com/package/";
const dash = "-".repeat(64);
const separator = "\n" + "=".repeat(64) + "\n";

// The same build as build.mjs, asked only which packages it takes in, so a dependency
// that esbuild tree-shakes away gets no entry and one it pulls in transitively gets one.
const result = await build({
  entryPoints: ["entry.js"],
  bundle: true,
  format: "iife",
  platform: "browser",
  write: false,
  metafile: true,
  define: {
    "process.env.NODE_ENV": JSON.stringify("production"),
  },
});
const packages = new Set();
for (const input of Object.keys(result.metafile.inputs)) {
  const m = input.match(/node_modules\/((?:@[^/]+\/)?[^/]+)\//);
  if (m) packages.add(m[1]);
}

const entries = [];
for (const name of [...packages].sort()) {
  const dir = join("node_modules", name);
  const pkg = JSON.parse(await readFile(join(dir, "package.json"), "utf8"));
  entries.push(`${name}\n${npmURL}${name}\n${dash}\n${await licenseText(dir, pkg)}\n`);
}

// Split on the separator so that joining with it again gives back the file exactly.
const current = await readFile(creditsPath, "utf8");
const kept = current
  .split(separator)
  .filter((entry) => entry !== "" && !isGenerated(entry));
const blocks = [...kept, ...entries.map((entry, i) => (kept.length + i === 0 ? entry : "\n" + entry))];
await writeFile(creditsPath, blocks.join(separator) + separator);

function isGenerated(entry) {
  const url = entry.replace(/^\n/, "").split("\n")[1] ?? "";
  return url.startsWith(npmURL);
}

// The license file the package ships, and where it ships none, what its package.json
// declares, since a copyright line nobody wrote is not something to write on its behalf.
async function licenseText(dir, pkg) {
  const files = (await readdir(dir)).filter((f) => /^licen[cs]e(\.(md|txt))?$/i.test(f)).sort();
  if (files.length > 0) {
    return (await readFile(join(dir, files[0]), "utf8")).trimEnd();
  }
  return `Licensed under ${pkg.license}, as declared in its package.json. The package ships no license file.`;
}
