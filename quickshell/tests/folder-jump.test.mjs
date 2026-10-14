import assert from "node:assert/strict";
import test from "node:test";
import vm from "node:vm";
import { source } from "./qml-script.mjs";

const library = (path, names) => {
    const ctx = vm.createContext({});
    vm.runInContext(source(path).replace(/^\.pragma.*$/gm, ""), ctx);
    return vm.runInContext(`({${names}})`, ctx);
};

const ctx = vm.createContext({
    Fzf: library("../DCommon/Common/fzf.js", "parseTerms, TermType"),
    PathCompletion: library("../DCommon/FileBrowser/PathCompletion.js", "compact, split")
});
vm.runInContext(source("../Common/FolderJump.js").replace(/^\.(pragma|import).*$/gm, ""), ctx);
const jump = vm.runInContext("({frecency, visit, forget, seed, parse, storedPath, prepare, top, match})", ctx);

const plain = value => JSON.parse(JSON.stringify(value));
const home = "/home/me";
const now = Date.UTC(2026, 9, 14, 12);
const minute = 60000;
const hour = 60 * minute;
const day = 24 * hour;

const entry = (path, visits, ago) => ({ path, visits, last: now - ago });
const paths = results => plain(results).map(result => result.path);
const find = (entries, query, limit = 10) => paths(jump.match(jump.prepare(entries, home), query, now, limit));

test("recent visits weigh more", () => {
    assert.equal(jump.frecency(entry("/a", 10, 30 * minute), now), 40);
    assert.equal(jump.frecency(entry("/a", 10, 2 * hour), now), 20);
    assert.equal(jump.frecency(entry("/a", 10, 2 * day), now), 5);
    assert.equal(jump.frecency(entry("/a", 10, 8 * day), now), 2.5);
});

test("a visit counts once more and moves the last visit", () => {
    const entries = [entry("/a", 3, day), entry("/b", 1, day)];
    assert.deepEqual(plain(jump.visit(entries, "/a", now)), [{ path: "/a", visits: 4, last: now }, entry("/b", 1, day)]);
    assert.deepEqual(plain(jump.visit(entries, "/c", now)).at(-1), { path: "/c", visits: 1, last: now });
});

test("the store keeps the thousand most frecent folders", () => {
    const entries = Array.from({ length: 1000 }, (_, index) => entry(`/f${index}`, index === 0 ? 1 : 2, day));
    const next = jump.visit(entries, "/new", now);
    assert.equal(next.length, 1000);
    assert.ok(next.some(item => item.path === "/new"));
    assert.ok(!next.some(item => item.path === "/f0"));
});

test("old visits age once the total grows too large, keeping the folder just visited", () => {
    const next = plain(jump.visit([entry("/busy", 10000, day), entry("/rare", 1, day)], "/new", now));
    assert.deepEqual(next.map(item => item.path), ["/busy", "/new"]);
    assert.ok(next[0].visits < 9000);
    assert.deepEqual(next[1], { path: "/new", visits: 1, last: now });
});

test("forgetting drops a folder", () => {
    assert.deepEqual(paths(jump.forget([entry("/a", 1, 0), entry("/b", 1, 0)], ["/a"])), ["/b"]);
});

test("seeding keeps the order of the old recent list", () => {
    const seeded = jump.seed(["/home/me/c", "/home/me/a", "relative", "/home/me/b"], now);
    assert.deepEqual(paths(jump.top(jump.prepare(seeded, home), now, 10)), ["/home/me/c", "/home/me/a", "/home/me/b"]);
});

test("stored state survives junk", () => {
    assert.equal(jump.parse("not json"), null);
    assert.equal(jump.parse("{}"), null);
    assert.deepEqual(plain(jump.parse('{"folders":[{"path":"/a","visits":2,"last":5},{"path":"","visits":1,"last":1},{"path":"/b","visits":0,"last":1}]}')), [{ path: "/a", visits: 2, last: 5 }]);
});

test("folders inside a network mount are stored as its uri", () => {
    const mounts = [{ path: "/run/user/1000/gvfs/sftp:host=nas,user=me", uri: "sftp://me@nas/" }];
    assert.equal(jump.storedPath("/run/user/1000/gvfs/sftp:host=nas,user=me/srv/www", mounts), "sftp://me@nas/srv/www");
    assert.equal(jump.storedPath("/run/user/1000/gvfs/sftp:host=nas,user=me", mounts), "sftp://me@nas/");
    assert.equal(jump.storedPath("/run/user/1000/gvfs/sftp:host=nas,user=meg", mounts), "/run/user/1000/gvfs/sftp:host=nas,user=meg");
    assert.equal(jump.storedPath("/home/me", mounts), "/home/me");
});

const visited = [
    entry("/home/me/Downloads", 3, 2 * day),
    entry("/home/me/Documents", 20, hour * 2),
    entry("/home/me/repos", 2, 2 * day),
    entry("/home/me/repos/dfiles", 40, minute),
    entry("/home/me/repos/dfiles/core/internal", 5, hour),
    entry("/home/me/repos/dms/src", 2, 3 * day),
    entry("/home/me/repos/dfiles/src", 9, hour),
    entry("/home/me/old/srcs", 30, minute),
    entry("/etc/ssh", 1, 9 * day)
];

test("a fuzzy word finds the folder", () => {
    assert.equal(find(visited, "ssh")[0], "/etc/ssh");
    assert.equal(find(visited, "dow")[0], "/home/me/Downloads");
});

test("frecency picks between equally good fuzzy matches", () => {
    assert.deepEqual(find(visited, "dl").slice(0, 2), ["/home/me/repos/dfiles", "/home/me/Downloads"]);
    const downloadsDaily = visited.map(item => item.path === "/home/me/Downloads" ? entry(item.path, 60, minute) : item);
    assert.equal(find(downloadsDaily, "dl")[0], "/home/me/Downloads");
});

test("a hit in the folder name beats a hit in the path", () => {
    assert.deepEqual(find(visited, "repos").slice(0, 1), ["/home/me/repos"]);
    assert.equal(find(visited, "internal")[0], "/home/me/repos/dfiles/core/internal");
});

test("an exact name beats a longer one however often that was visited", () => {
    assert.equal(find(visited, "src")[2], "/home/me/old/srcs");
});

test("frecency breaks ties between equal matches", () => {
    assert.deepEqual(find(visited, "src").slice(0, 2), ["/home/me/repos/dfiles/src", "/home/me/repos/dms/src"]);
});

test("several words must all match", () => {
    assert.deepEqual(find(visited, "dms src"), ["/home/me/repos/dms/src"]);
});

test("fzf operators", () => {
    assert.deepEqual(find(visited, "'ownl"), ["/home/me/Downloads"]);
    assert.deepEqual(find(visited, "^/etc"), ["/etc/ssh"]);
    assert.deepEqual(find(visited, "^dow"), ["/home/me/Downloads"]);
    assert.deepEqual(find(visited, "^dfiles$"), ["/home/me/repos/dfiles"]);
    assert.deepEqual(find(visited, "ments$"), ["/home/me/Documents"]);
    assert.ok(!find(visited, "src !dms").includes("/home/me/repos/dms/src"));
    assert.ok(find(visited, "src !dms").includes("/home/me/repos/dfiles/src"));
});

test("matches report the positions to mark", () => {
    const [hit] = plain(jump.match(jump.prepare([entry("/home/me/Downloads", 1, 0)], home), "dl", now, 1));
    assert.equal(hit.display, "~/Downloads");
    assert.deepEqual(hit.positions.sort((a, b) => a - b), [2, 6]);
});

test("an empty query lists the most frecent folders", () => {
    assert.deepEqual(paths(jump.match(jump.prepare(visited, home), "  ", now, 3)), ["/home/me/repos/dfiles", "/home/me/old/srcs", "/home/me/Documents"]);
});

test("the open folder is left out", () => {
    const prepared = jump.prepare(visited, home);
    assert.equal(paths(jump.top(prepared, now, 1, "/home/me/repos/dfiles"))[0], "/home/me/old/srcs");
    assert.ok(!paths(jump.match(prepared, "dfiles", now, 10, "/home/me/repos/dfiles")).includes("/home/me/repos/dfiles"));
});
