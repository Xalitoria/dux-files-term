import assert from "node:assert/strict";
import test from "node:test";
import { source } from "./qml-script.mjs";

const qml = source("../Common/Shortcuts.qml");
const doc = source("../../docs/shortcuts.md");

const table = qml.slice(qml.indexOf("readonly property var entries:"), qml.indexOf("readonly property var groups:"));
const entryPattern = /\{\s*"id":\s*"([^"]+)",\s*"keys":\s*\[([^\]]*)\],\s*"group":\s*"([^"]+)",\s*"scope":\s*"([^"]+)",\s*"label":\s*I18n\.tr\("([^"]*)"/g;

const entries = [...table.matchAll(entryPattern)].map(match => ({
    id: match[1],
    keys: match[2].split(",").map(key => key.trim().replaceAll('"', "")),
    group: match[3],
    scope: match[4],
    label: match[5]
}));

const documented = [...doc.matchAll(/^\| (.+?) \| (.+?) \| (\w+) \|$/gm)]
    .filter(row => row[1] !== "Keys")
    .map(row => ({
        keys: row[1].split(" / ").map(key => key.replaceAll("`", "")),
        label: row[2],
        scope: row[3]
    }));

test("every shortcut in the table was parsed", () => {
    assert.equal(entries.length, table.split('"id": "').length - 1);
    assert.ok(entries.length > 30);
});

test("the table and docs/shortcuts.md list the same shortcuts", () => {
    assert.deepEqual(documented.map(row => row.keys), entries.map(entry => entry.keys));
    assert.deepEqual(documented.map(row => row.label), entries.map(entry => entry.label));
    assert.deepEqual(documented.map(row => row.scope), entries.map(entry => entry.scope));
});

test("no key sequence is bound twice", () => {
    const seen = new Map();
    for (const entry of entries) {
        for (const key of entry.keys) {
            assert.ok(!seen.has(key), `${key} is bound by both ${seen.get(key)} and ${entry.id}`);
            seen.set(key, entry.id);
        }
    }
});

test("every id is unique and every scope is known", () => {
    const ids = entries.map(entry => entry.id);
    assert.equal(new Set(ids).size, ids.length);
    for (const entry of entries)
        assert.ok(["window", "selection", "path", "view"].includes(entry.scope), entry.scope);
});

test("every group in the table has a heading in the docs", () => {
    for (const group of new Set(entries.map(entry => entry.group)))
        assert.ok(qml.includes(`"id": "${group}"`), group);
});
