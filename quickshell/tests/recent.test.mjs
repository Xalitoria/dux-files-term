import assert from "node:assert/strict";
import test from "node:test";
import vm from "node:vm";
import { lift, source } from "./qml-script.mjs";

const snapshotModel = source("../Modules/SnapshotModel.qml");
const recentModel = source("../Modules/RecentModel.qml");
const windowActions = source("../Modules/WindowActions.qml");

function modelContext() {
    const ctx = vm.createContext({
        FilePaths: { parentOf: path => path.substring(0, path.lastIndexOf("/")) || "/" }
    });
    ctx.root = ctx;
    for (const name of ["sorted", "sortBy", "setSort"])
        lift(snapshotModel, name, ctx);
    for (const name of ["compare", "rowChanged"])
        lift(recentModel, name, ctx);
    ctx.descendingKeys = ["used", "mtime"];
    ctx._all = [];
    ctx._refilter = () => {};
    return ctx;
}

const item = (name, usedMs, extra = {}) => ({ name, path: "/home/me/docs/" + name, usedMs, mtimeMs: 0, size: 1, ...extra });

test("the last used file comes first and ties fall back to the name", () => {
    const ctx = modelContext();
    const names = ctx.sorted([item("b", 10), item("c", 30), item("a", 10)], "used", true).map(entry => entry.name);
    assert.deepEqual(names, ["c", "a", "b"]);
    const byLocation = ctx.sorted([item("b", 1, { path: "/z/b" }), item("a", 2, { path: "/m/a" })], "location", false).map(entry => entry.name);
    assert.deepEqual(byLocation, ["a", "b"]);
});

test("picking a time column starts newest first, a name column starts ascending", () => {
    const ctx = modelContext();
    ctx.sortKey = "used";
    ctx.sortDescending = true;
    ctx.sortBy("name");
    assert.equal(ctx.sortDescending, false);
    ctx.sortBy("mtime");
    assert.equal(ctx.sortDescending, true);
    ctx.sortBy("mtime");
    assert.equal(ctx.sortDescending, false);
});

test("a row is rewritten when it was used again or the file changed underneath", () => {
    const ctx = modelContext();
    assert.equal(ctx.rowChanged(item("a", 1), item("a", 1)), false);
    assert.equal(ctx.rowChanged(item("a", 1), item("a", 2)), true);
    assert.equal(ctx.rowChanged(item("a", 1), item("a", 1, { size: 9 })), true);
});

test("the recent view keeps file actions and drops the ones that need a real folder", () => {
    const ctx = vm.createContext({ mainWindow: { selectedPaths: () => [] } });
    lift(windowActions, "recentAction", ctx);
    for (const id of ["open", "openWith", "openContainingFolder", "cut", "copy", "trash", "deletePermanently", "moveTo", "copyTo", "removeFromRecent", "clearRecent", "selectAll", "reload"])
        assert.equal(ctx.recentAction(id), id, id);
    for (const id of ["rename", "paste", "newFolder", "newDocument", "openTerminal", "compress", "duplicate", "bookmark"])
        assert.equal(ctx.recentAction(id), "", id);
    for (const id of ["properties", "copyPath", "pasteInto"])
        assert.equal(ctx.recentAction(id), "", id);
    ctx.mainWindow.selectedPaths = () => ["/home/me/a.txt"];
    for (const id of ["properties", "copyPath", "pasteInto"])
        assert.equal(ctx.recentAction(id), id, id);
    ctx.mainWindow.selectedPaths = () => ["/home/me/a.txt", "/home/me/b.txt"];
    assert.equal(ctx.recentAction("pasteInto"), "");
});
