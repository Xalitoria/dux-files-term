import assert from "node:assert/strict";
import test from "node:test";
import vm from "node:vm";
import { lift, source } from "./qml-script.mjs";

const pathBar = source("../Modules/PathBar.qml");

function segments(path, home) {
    const ctx = vm.createContext({});
    lift(pathBar, "segmentsFor", ctx);
    return vm.runInContext("segmentsFor", ctx)(path, home);
}

const labels = (path, home) => [...segments(path, home)].map(s => s.iconName !== "" ? s.iconName : s.label);
const targets = (path, home) => [...segments(path, home)].map(s => s.path);

test("a path under home starts at a home pill", () => {
    assert.deepEqual(labels("/home/me/docs", "/home/me"), ["home", "docs"]);
    assert.deepEqual(targets("/home/me/docs", "/home/me"), ["/home/me", "/home/me/docs"]);
    assert.deepEqual(labels("/home/me", "/home/me"), ["home"]);
});

test("a path outside home starts at the root", () => {
    assert.deepEqual(labels("/etc/ssh", "/home/me"), ["hard_drive", "etc", "ssh"]);
    assert.deepEqual(targets("/etc/ssh", "/home/me"), ["/", "/etc", "/etc/ssh"]);
    assert.deepEqual(targets("/", "/home/me"), ["/"]);
});

test("a directory whose name starts with the home path is not home", () => {
    assert.deepEqual(labels("/home/melon", "/home/me"), ["hard_drive", "home", "melon"]);
});

test("an empty path has no pills", () => {
    assert.deepEqual(labels("", "/home/me"), []);
});
