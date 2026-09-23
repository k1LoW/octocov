import { copyFile } from "node:fs/promises";
import { createRequire } from "node:module";
import { build } from "esbuild";

await build({
  entryPoints: ["entry.js"],
  outfile: "../bundle.js",
  bundle: true,
  format: "iife",
  // react-dom/server resolves to React's Node build under the node platform, and the
  // engine has no Node built-ins to give it.
  platform: "browser",
  minify: true,
  define: {
    "process.env.NODE_ENV": JSON.stringify("production"),
  },
});

// Compiled by the package's own build, since there is no Tailwind to run here.
const require = createRequire(import.meta.url);
await copyFile(require.resolve("@octocov/ui/ssr.css"), "../ssr.css");

// The paths and anchors the package checks its own fileAnchor against, which FileAnchor is
// checked against too.
await copyFile(
  new URL("node_modules/@octocov/ui/src/fileAnchor.vectors.json", import.meta.url),
  "../testdata/fileAnchor.vectors.json"
);
