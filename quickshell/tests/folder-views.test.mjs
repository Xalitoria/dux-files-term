import assert from "node:assert/strict";
import test from "node:test";
import vm from "node:vm";
import { lift, source } from "./qml-script.mjs";

const folderViews = source("../Common/FolderViews.qml");

const plain = value => JSON.parse(JSON.stringify(value));

function context() {
    const ctx = vm.createContext({});
    for (const name of ["keyFor", "resolve", "prune", "renameKeys", "hasKeysUnder", "stripMatching"])
        lift(folderViews, name, ctx);
    return ctx;
}

const devices = [
    { system: true, mounted: true, uuid: "root-uuid", mountPoint: "/" },
    { system: false, mounted: true, uuid: "usb-uuid", mountPoint: "/run/media/me/USB" },
    { system: false, mounted: true, uuid: "deep-uuid", mountPoints: ["/run/media/me/USB/nested"] },
    { system: false, mounted: false, uuid: "gone-uuid", mountPoint: "/run/media/me/USB/old" },
    { system: false, mounted: true, uuid: "", mountPoint: "/mnt/nouuid" }
];

test("folders on removable mounts are keyed by uuid and the path inside", () => {
    const ctx = context();
    assert.equal(ctx.keyFor("/run/media/me/USB/photos", devices), "uuid:usb-uuid:/photos");
    assert.equal(ctx.keyFor("/run/media/me/USB", devices), "uuid:usb-uuid:");
    assert.equal(ctx.keyFor("/run/media/me/USB/nested/a", devices), "uuid:deep-uuid:/a");
    assert.equal(ctx.keyFor("/run/media/me/USBstick/a", devices), "/run/media/me/USBstick/a");
});

test("the system device, unmounted devices and devices without a uuid key by path", () => {
    const ctx = context();
    assert.equal(ctx.keyFor("/home/me/docs", devices), "/home/me/docs");
    assert.equal(ctx.keyFor("/run/media/me/USB/old/x", devices), "uuid:usb-uuid:/old/x");
    assert.equal(ctx.keyFor("/mnt/nouuid/x", devices), "/mnt/nouuid/x");
    assert.equal(ctx.keyFor("/tmp", []), "/tmp");
});

test("the folder's own value wins, then the last used one, then the default", () => {
    const ctx = context();
    const defaults = { sortKey: "name", sortDescending: false, viewMode: "list" };
    assert.deepEqual(plain(ctx.resolve(null, null, defaults)), defaults);
    assert.deepEqual(plain(ctx.resolve(null, { sortKey: "size", viewMode: "grid" }, defaults)), { sortKey: "size", sortDescending: false, viewMode: "grid" });
    assert.deepEqual(plain(ctx.resolve({ sortKey: "mtime", sortDescending: true }, { sortKey: "size", viewMode: "grid" }, defaults)), { sortKey: "mtime", sortDescending: true, viewMode: "grid" });
    assert.deepEqual(plain(ctx.resolve({ viewMode: "columns", sortDescending: false }, { sortDescending: true }, defaults)), { sortKey: "name", sortDescending: false, viewMode: "columns" });
});

test("the cap drops the least recently used folders first", () => {
    const ctx = context();
    const table = {};
    for (let i = 0; i < 6; i++)
        table[`/f${i}`] = { sortKey: "name", used: (i * 7) % 6 };
    const kept = ctx.prune(table, 4);
    assert.deepEqual([...Object.keys(kept).sort()], ["/f2", "/f3", "/f4", "/f5"]);
    assert.equal(ctx.prune(table, 6), table);
});

test("renaming a folder carries its entry and the entries below it", () => {
    const ctx = context();
    const table = {
        "/a/old": { viewMode: "grid" },
        "/a/old/sub": { sortKey: "size" },
        "/a/older": { sortKey: "type" },
        "uuid:u:/old": { viewMode: "grid" },
        "uuid:u:/old/x": { viewMode: "list" }
    };
    assert.deepEqual(plain(ctx.renameKeys(table, "/a/old", "/b/new")), {
        "/b/new": { viewMode: "grid" },
        "/b/new/sub": { sortKey: "size" },
        "/a/older": { sortKey: "type" },
        "uuid:u:/old": { viewMode: "grid" },
        "uuid:u:/old/x": { viewMode: "list" }
    });
    assert.equal(ctx.hasKeysUnder(table, "/a/old"), true);
    assert.equal(ctx.hasKeysUnder(table, "/a/ol"), false);
    assert.equal(ctx.hasKeysUnder(table, "/zzz"), false);
    assert.deepEqual([...Object.keys(ctx.renameKeys(table, "uuid:u:/old", "/home/me/old")).sort()], ["/a/old", "/a/old/sub", "/a/older", "/home/me/old", "/home/me/old/x"]);
});

test("making a folder's values the default drops the entries that now match", () => {
    const ctx = context();
    const table = {
        "/same": { sortKey: "size", sortDescending: true, viewMode: "grid", used: 1 },
        "/sortOnly": { sortKey: "size", sortDescending: true, viewMode: "list", used: 2 },
        "/other": { sortKey: "size", sortDescending: false, used: 3 }
    };
    assert.deepEqual(plain(ctx.stripMatching(table, { sortKey: "size", sortDescending: true, viewMode: "grid" })), {
        "/sortOnly": { viewMode: "list", used: 2 },
        "/other": { sortKey: "size", sortDescending: false, used: 3 }
    });
});
