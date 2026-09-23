import assert from "node:assert/strict";
import test from "node:test";
import { source } from "./qml-script.mjs";

// The keyboard map is one table: it binds the keys and fills the F1 overlay.
// These checks are what stops it growing a duplicate binding or a dead entry.
const qml = source("../Common/Shortcuts.qml");

const table = qml.slice(qml.indexOf("readonly property var entries:"), qml.indexOf("readonly property var groups:"));
const entryPattern = /\{\s*"id":\s*"([^"]+)",\s*"keys":\s*\[([^\]]*)\],\s*"group":\s*"([^"]+)",\s*"scope":\s*"([^"]+)",\s*"label":\s*I18n\.tr\("([^"]*)",\s*"([^"]*)"\)/g;

const entries = [...table.matchAll(entryPattern)].map(match => ({
    id: match[1],
    keys: match[2].split(",").map(key => key.trim().replaceAll('"', "")),
    group: match[3],
    scope: match[4],
    label: match[5],
    context: match[6]
}));

const groups = [...qml.slice(qml.indexOf("readonly property var groups:")).matchAll(/"id":\s*"([^"]+)"/g)].map(match => match[1]);

test("every entry in the table parses", () => {
    assert.equal(entries.length, table.split('"id": "').length - 1);
    assert.ok(entries.length > 30);
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

test("every id is unique", () => {
    const ids = entries.map(entry => entry.id);
    assert.equal(new Set(ids).size, ids.length);
});

test("every scope is one MainWindow binds", () => {
    for (const entry of entries)
        assert.ok(["window", "selection", "path", "view"].includes(entry.scope), `${entry.id}: ${entry.scope}`);
});

test("every entry lands in a group the overlay renders", () => {
    assert.ok(groups.length > 0);
    for (const entry of entries)
        assert.ok(groups.includes(entry.group), `${entry.id}: ${entry.group}`);
});

test("every entry has keys and a translated label with a context", () => {
    for (const entry of entries) {
        assert.ok(entry.keys.length > 0, entry.id);
        assert.ok(entry.keys.every(key => key.length > 0), entry.id);
        assert.ok(entry.label.length > 0, entry.id);
        assert.ok(entry.context.length > 0, `${entry.id} has no translator context`);
    }
});
