// What the Go side evaluates: one script leaving the changes page renderer and the theme
// script behind as globals, since go-spidermonkey has no module system to hand them to.
import { THEME_SCRIPT, renderChangesPageJSON } from "@octocov/ui/ssr";

globalThis.renderChangesPageJSON = renderChangesPageJSON;
globalThis.THEME_SCRIPT = THEME_SCRIPT;
