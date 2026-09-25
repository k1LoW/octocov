// Writes the license notices of the npm packages bundled into ../bundle.js into
// _EXTRA_CREDITS, which the release appends to the CREDITS gocredits writes for the Go
// modules. gocredits knows nothing of what is bundled here, and the bundle is embedded in
// every binary and written into every page it renders.
//
// The TextMate grammars are credited too, apart from the package that carries them.
// @shikijs/langs-precompiled is MIT, which is its own licence and not theirs: each grammar
// comes from a project of its own under the licence that project chose, which tm-grammars,
// where shiki takes them from, collects in its NOTICE. So each grammar the bundle takes is
// looked up there, and the notice that covers it is written out.
//
// An entry this script writes is the one whose URL is the package's page on npm, so each
// run removes those and writes them again from what is bundled now. Running it twice
// leaves the file as the first run did, and the entry of a package the bundle no longer
// takes is gone after the next run. The entries written by hand are kept byte for byte.
import { readFile, readdir, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { build } from "esbuild";
import { grammars } from "tm-grammars";
import { grammarsCreditedByHand, stubUnlicensedGrammars } from "./grammars.mjs";

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
  plugins: [stubUnlicensedGrammars],
});
const packages = new Set();
const bundledGrammars = new Set();
for (const input of Object.keys(result.metafile.inputs)) {
  const m = input.match(/node_modules\/((?:@[^/]+\/)?[^/]+)\//);
  if (m) packages.add(m[1]);
  const g = input.match(/node_modules\/@shikijs\/langs-precompiled\/dist\/([^/]+)\.mjs$/);
  if (g) bundledGrammars.add(g[1]);
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

for (const name of grammarsCreditedByHand) {
  if (!bundledGrammars.has(name)) continue;
  bundledGrammars.delete(name);
  const title = `TextMate grammars: ${name}`;
  if (!kept.some((entry) => entry.replace(/^\n/, "").split("\n")[0] === title)) {
    throw new Error(`_EXTRA_CREDITS has no entry "${title}", which grammars.mjs says it has`);
  }
}
entries.push(...(await grammarEntries(bundledGrammars)));
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

// One entry per notice in tm-grammars' NOTICE that covers a bundled grammar, naming the
// grammars it covers here. Under the package's npm page like every generated entry, so
// the next run replaces it along with the rest.
async function grammarEntries(names) {
  const notice = await readFile(join("node_modules", "tm-grammars", "NOTICE"), "utf8");
  const sections = new Map();
  for (const chunk of notice.split(/^=+$/m).slice(1)) {
    const [head, ...rest] = chunk.replace(/^\n/, "").split("\n");
    const files = head.match(/^Files:\s+(.*)$/);
    if (!files) continue;
    const section = { body: rest.join("\n").trimEnd(), names: [] };
    for (const file of files[1].split(/,\s*/)) sections.set(file.replace(/\.json$/, ""), section);
  }
  const used = new Set();
  for (const name of names) {
    // A file named for an alias, such as bash, re-exports the grammar it is an alias of,
    // which the bundle then takes too and is credited under its own name
    const grammar = grammars.find((g) => g.name === name || g.aliases?.includes(name));
    if (grammar && grammar.name !== name) continue;
    const section = sections.get(name);
    // Failed rather than skipped: a grammar with no notice is one this file would ship
    // without the licence it came under, and nothing after this would say so
    if (!section) throw new Error(`no notice in tm-grammars covers the grammar ${name}`);
    section.names.push(name);
    used.add(section);
  }
  return [...used]
    .map((section) => ({ ...section, names: section.names.sort() }))
    .sort((a, b) => a.names[0].localeCompare(b.names[0]))
    .map(
      (section) =>
        `TextMate grammars: ${section.names.join(", ")}\n${npmURL}tm-grammars\n${dash}\n${section.body}\n`
    );
}
