import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";

const here = new URL("./", import.meta.url);

export const source = path => readFileSync(new URL(path, here), "utf8");

// Lifts one top-level `function name(...) { ... }` out of a QML source into
// the context, so pure helpers run without Qt.
export function lift(text, name, context) {
    const match = text.match(new RegExp(`^    function ${name}\\([^\\n]*\\) \\{\\n[\\s\\S]*?^    \\}`, "m"));
    assert.ok(match, `function ${name} not found`);
    vm.runInContext(match[0], context);
}
