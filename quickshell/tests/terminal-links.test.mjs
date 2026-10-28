import assert from "node:assert/strict";
import test from "node:test";
import vm from "node:vm";
import { lift, source } from "./qml-script.mjs";

const view = source("../Modules/TerminalView.qml");

const terminal = (cols, rows) => {
    const ctx = vm.createContext({});
    vm.runInContext(`var cols = ${cols}; var lineStore = ${JSON.stringify(rows)}; var opened = [];
        var DuxFilesService = { openUri: url => opened.push(url) };`, ctx);
    for (const name of ["rowCells", "linkAt", "openLink"])
        lift(view, name, ctx);
    return ctx;
};

const run = (x, t, extra = {}) => ({ x, t, w: t.length, ...extra });
const at = (ctx, x, y = 0) => vm.runInContext(`linkAt({x: ${x}, y: ${y}})`, ctx);

test("a hyperlink from the program wins over the text it covers", () => {
    const ctx = terminal(30, [[run(0, "see "), run(4, "docs", { l: "https://example.com/docs" })]]);
    assert.equal(at(ctx, 5), "https://example.com/docs");
    assert.equal(at(ctx, 1), "");
});

test("plain URLs are found without trailing punctuation", () => {
    const ctx = terminal(40, [[run(0, "go to https://example.com/a?b=1, now")]]);
    assert.equal(at(ctx, 10), "https://example.com/a?b=1");
    assert.equal(at(ctx, 2), "");
    assert.equal(at(ctx, 33), "");
});

test("only web, file and mail links are opened", () => {
    const ctx = terminal(10, [[]]);
    for (const url of ["https://a.b", "file:///tmp/x", "mailto:a@b.c", "javascript:alert(1)", "ssh://host", "x-terminal:run"])
        vm.runInContext(`openLink(${JSON.stringify(url)})`, ctx);
    assert.deepEqual(JSON.parse(vm.runInContext("JSON.stringify(opened)", ctx)), ["https://a.b", "file:///tmp/x", "mailto:a@b.c"]);
});
