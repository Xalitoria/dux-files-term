import assert from "node:assert/strict";
import test from "node:test";
import { source } from "./qml-script.mjs";

// The sandbox fixture fakes the transport, so the real signal name is pinned
// here: renaming it in DankFilesService would otherwise stop capability
// refreshes after a reconnect without any test noticing.
test("SandboxService listens to the transport's connection signal", () => {
    assert.match(source("../Services/DankFilesService.qml"), /^    signal connectionStateChanged$/m);
    assert.match(source("../Services/SandboxService.qml"), /function onConnectionStateChanged\(\)/);
});
