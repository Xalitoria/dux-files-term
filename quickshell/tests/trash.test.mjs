import assert from "node:assert/strict";
import test from "node:test";
import vm from "node:vm";
import { lift, source } from "./qml-script.mjs";

const trashModel = source("../Modules/TrashModel.qml");
const mainWindow = source("../Modules/MainWindow.qml");

function modelContext() {
    const ctx = vm.createContext({});
    for (const name of ["sorted", "matches", "syncInto"])
        lift(trashModel, name, ctx);
    return ctx;
}

const item = (name, deletedMs, extra = {}) => ({ name, path: "/t/files/" + name, deletedMs, originalPath: "/home/me/" + name, size: 1, displayName: "", thumbnail: "", ...extra });

class FakeList {
    constructor(rows) {
        this.rows = rows.map(row => ({ ...row }));
        this.writes = 0;
    }
    clear() {
        this.writes++;
        this.rows = [];
    }
    append(rows) {
        this.writes++;
        this.rows.push(...rows.map(row => ({ ...row })));
    }
    set(i, row) {
        this.writes++;
        this.rows[i] = { ...row };
    }
    setProperty(i, key, value) {
        this.writes++;
        this.rows[i][key] = value;
    }
    insert(i, row) {
        this.writes++;
        this.rows.splice(i, 0, { ...row });
    }
    remove(i, count = 1) {
        this.writes++;
        this.rows.splice(i, count);
    }
}

test("newest deleted comes first and ties fall back to the name", () => {
    const ctx = modelContext();
    const names = ctx.sorted([item("b", 10), item("c", 30), item("a", 10)], "deleted", true).map(entry => entry.name);
    assert.deepEqual(names, ["c", "a", "b"]);
    const byOrigin = ctx.sorted([item("b", 1, { originalPath: "/z/b" }), item("a", 2, { originalPath: "/m/a" })], "origin", false).map(entry => entry.name);
    assert.deepEqual(byOrigin, ["a", "b"]);
});

test("the filter matches the original name", () => {
    const ctx = modelContext();
    assert.equal(ctx.matches(item("Report.pdf", 1), "rep"), true);
    assert.equal(ctx.matches(item("Report.pdf", 1), "zzz"), false);
    assert.equal(ctx.matches(item("Report.pdf", 1), ""), true);
});

test("a refresh keeps the rows that stayed and lands on the new order", () => {
    const ctx = modelContext();
    const first = [item("a", 3, { thumbnail: "/thumbs/a.png" }), item("b", 2), item("c", 1)];
    const list = new FakeList([]);
    let shown = ctx.syncInto(list, [], first);
    list.writes = 0;

    shown = ctx.syncInto(list, shown, [item("a", 3), item("b", 2), item("c", 1)]);
    assert.equal(list.writes, 0);
    assert.equal(shown[0].thumbnail, "/thumbs/a.png");

    shown = ctx.syncInto(list, shown, [item("d", 4), item("a", 3), item("c", 1)]);
    assert.deepEqual(list.rows.map(row => row.name), ["d", "a", "c"]);
    assert.equal(list.writes, 2);

    shown = ctx.syncInto(list, shown, [item("c", 5), item("d", 4), item("a", 3)]);
    assert.deepEqual(list.rows.map(row => row.name), ["c", "d", "a"]);
    assert.equal(list.rows[0].deletedMs, 5);
    assert.deepEqual(shown.map(row => row.name), ["c", "d", "a"]);

    ctx.syncInto(list, shown, []);
    assert.deepEqual(list.rows, []);
});

test("restore sends one job per trash directory", () => {
    const calls = [];
    const ctx = vm.createContext({
        opScope: "window1",
        FileOpsService: { restore: (names, dir, scope) => calls.push({ names: [...names], dir, scope }) }
    });
    lift(mainWindow, "restoreEntries", ctx);
    ctx.restoreEntries([
        { trashDir: "/home/me/.local/share/Trash", trashName: "a.txt" },
        { trashDir: "/media/disk/.Trash-1000", trashName: "photo.jpg" },
        { trashDir: "/home/me/.local/share/Trash", trashName: "b.2.txt" }
    ]);
    assert.deepEqual(calls, [
        { names: ["a.txt", "b.2.txt"], dir: "/home/me/.local/share/Trash", scope: "window1" },
        { names: ["photo.jpg"], dir: "/media/disk/.Trash-1000", scope: "window1" }
    ]);
});
