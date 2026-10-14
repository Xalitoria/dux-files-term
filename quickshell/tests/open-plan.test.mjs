import assert from "node:assert/strict";
import test from "node:test";
import vm from "node:vm";
import { lift, source } from "./qml-script.mjs";

const mainWindow = source("../Modules/MainWindow.qml");

function context() {
    const ctx = vm.createContext({ FilesFormat: { extractable: mime => mime === "application/zip" } });
    for (const name of ["openPlan", "openCount"])
        lift(mainWindow, name, ctx);
    return ctx;
}

const folder = path => ({ path, isDir: true, mime: "inode/directory" });
const file = (path, mime = "text/plain") => ({ path, isDir: false, mime });

test("the first folder opens in the pane and the rest in tabs", () => {
    const ctx = context();
    const plan = ctx.openPlan([file("/a.txt"), folder("/one"), folder("/two"), folder("/three")], false);
    assert.equal(plan.folder, "/one");
    assert.deepEqual([...plan.tabs], ["/two", "/three"]);
    assert.deepEqual([...plan.files], ["/a.txt"]);
    assert.equal(ctx.openCount(plan), 3);
});

test("archives are unpacked only when extract on open is on", () => {
    const ctx = context();
    const entries = [file("/a.zip", "application/zip"), file("/b.txt")];
    const extracting = ctx.openPlan(entries, true);
    assert.deepEqual([...extracting.archives], ["/a.zip"]);
    assert.deepEqual([...extracting.files], ["/b.txt"]);
    const opening = ctx.openPlan(entries, false);
    assert.deepEqual([...opening.archives], []);
    assert.deepEqual([...opening.files], ["/a.zip", "/b.txt"]);
});

test("files alone leave the pane where it is", () => {
    const ctx = context();
    const plan = ctx.openPlan([file("/a.txt"), file("/b.txt")], false);
    assert.equal(plan.folder, "");
    assert.deepEqual([...plan.tabs], []);
    assert.equal(ctx.openCount(plan), 2);
});
