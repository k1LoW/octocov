// What build.mjs and credits.mjs agree on about the TextMate grammars, so the bundle that
// ships and the one credited are the same one.

// Grammars shiki carries that come under no licence at all, replaced in the bundle by a
// grammar of the same name that matches nothing. Without a licence they are not ours to
// redistribute, and every binary carries the bundle. None of them is a language the page
// colours on its own: each is pulled in as a language embedded in one it does, and shiki
// refuses to load a grammar whose embedded languages are missing, so the name stays and
// only the patterns go. Code in them is then drawn in its host language's plain colour.
//
// glsl is polym0rph/GLSL.tmbundle, which C++ embeds for shaders in raw string literals. The
// repository has no licence file and says nothing of one in its README or its info.plist.
export const unlicensedGrammars = { glsl: "source.glsl" };

// Grammars whose licence tm-grammars' NOTICE leaves out, credited by hand in _EXTRA_CREDITS
// under `TextMate grammars: <name>`, which credits.mjs checks is there.
//
// yaml is textmate/yaml.tmbundle, whose Syntaxes/YAML-license.txt is MIT. tm-grammars
// records no licence for it, going by the repository having no licence file at its root.
export const grammarsCreditedByHand = ["yaml"];

export const stubUnlicensedGrammars = {
  name: "stub-unlicensed-grammars",
  setup(build) {
    const names = Object.keys(unlicensedGrammars).join("|");
    build.onResolve({ filter: new RegExp(`/(${names})\\.mjs$`) }, (args) => {
      if (!args.importer.includes("@shikijs/langs-precompiled")) return undefined;
      const name = args.path.match(/([^/]+)\.mjs$/)[1];
      return { path: name, namespace: "unlicensed-grammar" };
    });
    build.onLoad({ filter: /.*/, namespace: "unlicensed-grammar" }, (args) => ({
      contents: `export default [${JSON.stringify({
        name: args.path,
        scopeName: unlicensedGrammars[args.path],
        patterns: [],
        repository: {},
      })}]`,
      loader: "js",
    }));
  },
};
