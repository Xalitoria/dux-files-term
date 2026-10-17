import assert from "node:assert/strict";
import test from "node:test";
import vm from "node:vm";
import { lift, source } from "./qml-script.mjs";

const settingsSearch = source("../Common/SettingsSearch.qml");

const plain = value => JSON.parse(JSON.stringify(value));

function context() {
    const ctx = vm.createContext({});
    for (const name of ["tokens", "score"])
        lift(settingsSearch, name, ctx);
    return ctx;
}

test("a title prefix outranks a word start, which outranks a keyword", () => {
    const ctx = context();
    const title = ctx.score("sort", "Sort by", "", "");
    const word = ctx.score("desc", "Sort descending", "", "");
    const keyword = ctx.score("order", "Sort by", "", "order name size");
    const subtitle = ctx.score("current", "Show hidden files", "Ctrl+H still toggles them for the current window.", "");
    assert.ok(title > word && word > keyword && keyword > subtitle && subtitle > 0);
});

test("every word of the query has to match somewhere", () => {
    const ctx = context();
    assert.equal(ctx.score("hidden files", "Show hidden files", "", "dotfiles"), 6);
    assert.equal(ctx.score("hidden xyz", "Show hidden files", "", "dotfiles"), 0);
    assert.equal(ctx.score("dot sort", "Show hidden files", "", "dotfiles"), 0);
    assert.equal(ctx.score("Dot HID", "Show hidden files", "", "dotfiles"), 5);
});

test("blank queries and empty rows score nothing", () => {
    const ctx = context();
    assert.equal(ctx.score("", "Sort by", "", "order"), 0);
    assert.equal(ctx.score("   ", "Sort by", "", "order"), 0);
    assert.equal(ctx.score("sort", "", "", ""), 0);
    assert.deepEqual(plain(ctx.tokens("  Two   words ")), ["two", "words"]);
});
