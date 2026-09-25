// What the Go side evaluates: one script leaving the changes page renderer and the theme
// script behind as globals, since go-spidermonkey has no module system to hand them to.
import { THEME_SCRIPT, renderChangesPageJSON } from "@octocov/ui/ssr";
import { createPageHighlighter, languageOf } from "@octocov/ui/highlight";
import { createJavaScriptRawEngine } from "shiki/engine/javascript";
// The grammars octocov.dev colours, so a card reads the same in the artifact as on the
// site. Precompiled, with the raw engine: the regex engine translates each pattern on its
// first match, and in this engine that was most of what a first tokenize took, 7.0 s
// against 0.93 s over the first seventeen of these. Which languages the page colours is this list, and
// a file of any other is drawn plain.
import bash from "@shikijs/langs-precompiled/bash";
import c from "@shikijs/langs-precompiled/c";
import cpp from "@shikijs/langs-precompiled/cpp";
import csharp from "@shikijs/langs-precompiled/csharp";
import css from "@shikijs/langs-precompiled/css";
import go from "@shikijs/langs-precompiled/go";
import html from "@shikijs/langs-precompiled/html";
import java from "@shikijs/langs-precompiled/java";
import javascript from "@shikijs/langs-precompiled/javascript";
import json from "@shikijs/langs-precompiled/json";
import jsx from "@shikijs/langs-precompiled/jsx";
import markdown from "@shikijs/langs-precompiled/markdown";
import php from "@shikijs/langs-precompiled/php";
import python from "@shikijs/langs-precompiled/python";
import ruby from "@shikijs/langs-precompiled/ruby";
import rust from "@shikijs/langs-precompiled/rust";
import sql from "@shikijs/langs-precompiled/sql";
import tsx from "@shikijs/langs-precompiled/tsx";
import typescript from "@shikijs/langs-precompiled/typescript";
import xml from "@shikijs/langs-precompiled/xml";
import yaml from "@shikijs/langs-precompiled/yaml";

let highlighter = null;

// Built on the first text there is to colour rather than on evaluation. Building it takes
// about two seconds in this engine, and a page whose files are in none of these languages
// has nothing to spend them on.
function forPage() {
  let page = null;
  return (code, filename) => {
    if (languageOf(filename) === null) return null;
    highlighter ??= createPageHighlighter({
      langs: [
        bash, c, cpp, csharp, css, go, html, java, javascript, json, jsx, markdown, php,
        python, ruby, rust, sql, tsx, typescript, xml, yaml,
      ],
      engine: createJavaScriptRawEngine(),
    });
    page ??= highlighter.forPage();
    return page(code, filename);
  };
}

// The same name and the same string each way as before, so the Go side is unchanged. Each
// render gets a page budget of its own.
globalThis.renderChangesPageJSON = (inputJSON) =>
  renderChangesPageJSON(inputJSON, { highlight: forPage() });
globalThis.THEME_SCRIPT = THEME_SCRIPT;
