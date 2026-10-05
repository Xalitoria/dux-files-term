import assert from "node:assert/strict";
import test from "node:test";
import vm from "node:vm";
import { lift, source } from "./qml-script.mjs";

const fileFormat = source("../DCommon/FileBrowser/FileFormat.qml");
const filesFormat = source("../Common/FilesFormat.qml");

function context() {
    const ctx = vm.createContext({ I18n: { tr: term => term } });
    for (const name of ["size", "operationError"])
        lift(fileFormat, name, ctx);
    for (const name of ["elevationUnavailable", "featureUnavailable", "isRunnable"])
        lift(filesFormat, name, ctx);
    return ctx;
}

test("size scales by 1000 and hides negative sizes", () => {
    const ctx = context();
    assert.equal(ctx.size(-1), "");
    assert.equal(ctx.size(0), "0 B");
    assert.equal(ctx.size(999), "999 B");
    assert.equal(ctx.size(1000), "1.0 kB");
    assert.equal(ctx.size(1536000), "1.5 MB");
    assert.equal(ctx.size(2.5e12), "2.5 TB");
    assert.equal(ctx.size(9e15), "9000.0 TB");
});

test("operationError keeps the daemon message only for unmapped codes", () => {
    const ctx = context();
    assert.equal(ctx.operationError("NOTSUPPORTED", "raw"), "raw");
    assert.equal(ctx.operationError("", "raw"), "raw");
    assert.equal(ctx.operationError("", ""), "This item could not be handled");
    assert.equal(ctx.operationError("EACCES", "raw"), "You do not have permission to do this");
    assert.equal(ctx.operationError("SANDBOXED", "raw"), "This item could not be handled");
});

test("every sandbox reason code has its own explanation", () => {
    const ctx = context();
    const codes = ["SANDBOXED", "NOHOSTFS", "NOUDISKS", "NOGVFS", "NOHOSTSPAWN", "PORTALCHOOSER"];
    const texts = codes.map(code => ctx.featureUnavailable(code));
    assert.equal(new Set(texts).size, codes.length);
    assert.equal(ctx.featureUnavailable("SOMETHING_NEW"), "Not available");
    assert.notEqual(ctx.elevationUnavailable("SANDBOXED"), ctx.featureUnavailable("SANDBOXED"));
});

test("isRunnable needs the execute bit and a program-like type", () => {
    const ctx = context();
    assert.equal(ctx.isRunnable("application/x-executable", false), false);
    assert.equal(ctx.isRunnable("application/x-executable", true), true);
    assert.equal(ctx.isRunnable("application/x-shellscript", true), true);
    assert.equal(ctx.isRunnable("text/x-python", true), true);
    assert.equal(ctx.isRunnable("", true), true);
    assert.equal(ctx.isRunnable("image/png", true), false);
});
