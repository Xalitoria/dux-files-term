import assert from "node:assert/strict";
import test from "node:test";
import vm from "node:vm";
import { lift, source } from "./qml-script.mjs";

const view = source("../Modules/TerminalView.qml");

const Qt = {
    ShiftModifier: 0x02000000,
    ControlModifier: 0x04000000,
    AltModifier: 0x08000000,
    MetaModifier: 0x10000000,
    Key_A: 65, Key_Z: 90, Key_0: 48, Key_9: 57,
    Key_Plus: 43, Key_Equal: 61, Key_Minus: 45, Key_Underscore: 95,
    Key_Backspace: 0x01000003, Key_Insert: 0x01000006, Key_PageUp: 0x01000016,
    Key_F1: 0x01000030, Key_F12: 0x0100003b,
};

const terminal = (behavior = {}, selected = false) => {
    const ctx = vm.createContext({ Qt, SessionData: { terminalZoom: 0 }, FilesMetrics: { terminalMinZoom: -6, terminalMaxZoom: 12 } });
    vm.runInContext(`var behavior = ${JSON.stringify(behavior)}; var selStart = ${selected ? "{x: 0, y: 0}" : "null"}; var copies = 0; var sent = [];
        function copySelection() { copies++; }
        function sendInput(params) { sent.push(params); }
        var Quickshell = { clipboardText: "clip" };`, ctx);
    for (const name of ["keyName", "chordKey", "chordOf", "actionFor", "runAction", "setZoom"])
        lift(view, name, ctx);
    return ctx;
};

const press = (ctx, key, modifiers = 0) => {
    ctx.event = { key: Qt[key], modifiers: ["Shift", "Control", "Alt", "Meta"].reduce((mask, name) => (modifiers.includes?.(name) ? mask | Qt[`${name}Modifier`] : mask), 0) };
    return vm.runInContext("chordOf(event)", ctx);
};

test("key events read as the same chords the daemon sends", () => {
    const ctx = terminal();
    assert.equal(press(ctx, "Key_A", ["Control", "Shift"]), "ctrl+shift+a");
    assert.equal(press(ctx, "Key_Equal", ["Control"]), "ctrl+plus");
    assert.equal(press(ctx, "Key_Underscore", ["Control", "Shift"]), "ctrl+shift+minus");
    assert.equal(press(ctx, "Key_Insert", ["Shift"]), "shift+insert");
    assert.equal(press(ctx, "Key_PageUp", ["Alt", "Meta"]), "alt+super+pageup");
    assert.equal(press(ctx, "Key_F1"), "f1");
});

test("the defaults stay bound and a terminal's own chords join them", () => {
    const ctx = terminal({ keys: { copy: ["ctrl+c"], zoomIn: ["ctrl+plus"] } });
    const action = chord => vm.runInContext(`actionFor(${JSON.stringify(chord)})`, ctx);
    assert.equal(action("ctrl+shift+c"), "copy");
    assert.equal(action("ctrl+c"), "copy");
    assert.equal(action("ctrl+plus"), "zoomIn");
    assert.equal(action("ctrl+shift+plus"), "zoomIn");
    assert.equal(action("ctrl+shift+backspace"), "zoomReset");
    assert.equal(action("ctrl+d"), "");
});

test("a copy chord without a selection reaches the shell, except the default one", () => {
    const idle = terminal();
    assert.equal(vm.runInContext('runAction("copy", "ctrl+c")', idle), false, "ctrl+c still interrupts");
    assert.equal(vm.runInContext('runAction("copy", "ctrl+shift+c")', idle), true);

    const selected = terminal({}, true);
    assert.equal(vm.runInContext('runAction("copy", "ctrl+c")', selected), true);
    assert.equal(selected.copies, 1);
});

test("paste sends the clipboard and zoom steps stay within the limits", () => {
    const ctx = terminal();
    vm.runInContext('runAction("paste", "ctrl+shift+v")', ctx);
    assert.deepEqual(JSON.parse(JSON.stringify(ctx.sent)), [{ text: "clip", paste: true }]);
    for (let i = 0; i < 30; i++)
        vm.runInContext('runAction("zoomIn", "")', ctx);
    assert.equal(ctx.SessionData.terminalZoom, 12);
    vm.runInContext('runAction("zoomReset", "")', ctx);
    assert.equal(ctx.SessionData.terminalZoom, 0);
});
