import assert from "node:assert/strict";
import test from "node:test";
import vm from "node:vm";
import { lift, source } from "./qml-script.mjs";

const view = source("../Modules/TerminalView.qml");

const terminal = (cols, rows) => {
    const ctx = vm.createContext({ Qt: { point: (x, y) => ({ x, y }) } });
    vm.runInContext(`var cols = ${cols}; var lineStore = ${JSON.stringify(rows)}; var selStart = null; var selEnd = null; var selAnchor = null; var behavior = {};`, ctx);
    for (const name of ["rowCells", "isWordCell", "select", "clearSelection", "selectWord", "selectLine", "selectedText"])
        lift(view, name, ctx);
    return ctx;
};

const run = (x, t, w = t.length) => ({ x, t, w });
const point = (x, y) => ({ x, y });
const call = (ctx, code) => JSON.parse(JSON.stringify(vm.runInContext(code, ctx)));

test("a drag selects the cells between its ends, whichever way it goes", () => {
    const ctx = terminal(10, [[run(0, "hello wor")]]);
    vm.runInContext("select({x: 4, y: 0}, {x: 0, y: 0})", ctx);
    assert.equal(call(ctx, "selectedText()"), "hello");
});

test("a selection over several rows keeps the line breaks and drops trailing blanks", () => {
    const ctx = terminal(8, [[run(0, "one")], [run(0, "two")], [run(0, "three")]]);
    vm.runInContext("select({x: 1, y: 0}, {x: 2, y: 2})", ctx);
    assert.equal(call(ctx, "selectedText()"), "ne\ntwo\nthr");
});

test("double click picks the word, including path and url characters", () => {
    const ctx = terminal(30, [[run(0, "cd ~/Documents/a-b.txt now")]]);
    vm.runInContext("selectWord({x: 8, y: 0})", ctx);
    assert.equal(call(ctx, "selectedText()"), "~/Documents/a-b.txt");
    vm.runInContext("selectWord({x: 2, y: 0})", ctx);
    assert.equal(call(ctx, "selectedText()"), "", "a blank copies nothing");
});

test("triple click picks the whole row", () => {
    const ctx = terminal(20, [[run(0, "  indented text")]]);
    vm.runInContext("selectLine({x: 5, y: 0})", ctx);
    assert.equal(call(ctx, "selectedText()"), "  indented text");
});

test("gaps between runs read as spaces and wide glyphs count once", () => {
    const ctx = terminal(8, [[run(0, "a"), run(3, "漢", 2), run(5, "b")]]);
    vm.runInContext("select({x: 0, y: 0}, {x: 7, y: 0})", ctx);
    assert.equal(call(ctx, "selectedText()"), "a  漢b");
});

test("clearing drops the selection", () => {
    const ctx = terminal(5, [[run(0, "abc")]]);
    vm.runInContext("selectLine({x: 0, y: 0})", ctx);
    vm.runInContext("clearSelection()", ctx);
    assert.equal(call(ctx, "selectedText()"), "");
});

test("a wide emoji stays whole when the selection ends on it", () => {
    const ctx = terminal(6, [[run(0, "a"), run(1, "😀", 2), run(3, "b")]]);
    vm.runInContext("select({x: 0, y: 0}, {x: 1, y: 0})", ctx);
    assert.equal(call(ctx, "selectedText()"), "a😀");
});

test("a terminal that lists delimiters ends words at those characters only", () => {
    const ctx = terminal(30, [[run(0, "git commit -m'fix' ~/a.txt")]]);
    vm.runInContext("behavior = {wordDelims: \" '\"}", ctx);
    vm.runInContext("selectWord({x: 12, y: 0})", ctx);
    assert.equal(call(ctx, "selectedText()"), "-m", "the quote ends the word");
    vm.runInContext("selectWord({x: 22, y: 0})", ctx);
    assert.equal(call(ctx, "selectedText()"), "~/a.txt");
});

test("a terminal that lists word characters adds them to letters and digits", () => {
    const ctx = terminal(30, [[run(0, "see a:b-c/d.e now")]]);
    vm.runInContext("behavior = {wordChars: \"-.\"}", ctx);
    vm.runInContext("selectWord({x: 4, y: 0})", ctx);
    assert.equal(call(ctx, "selectedText()"), "a", "the colon is not a word character here");
    vm.runInContext("selectWord({x: 6, y: 0})", ctx);
    assert.equal(call(ctx, "selectedText()"), "b-c", "the slash is not either");
});
