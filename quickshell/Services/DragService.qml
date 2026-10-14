pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Common
import qs.Services
import qs.DCommon.FileBrowser

Singleton {
    id: root

    property var transport: DankFilesService
    property var paths: []
    property string sourceDir: ""
    property bool active: false
    property int generation: 0
    property var plainDrops: ({})

    readonly property string hoverDest: active ? FileDragHover.dest : ""
    readonly property string action: {
        if (hoverDest === "" || paths.includes(hoverDest))
            return "";
        const resolved = resolve(overrideFor(FileDragHover.proposedAction), plainDrops[hoverDest] ?? "");
        if (resolved === "move" && allInside(paths, hoverDest))
            return "";
        return resolved;
    }

    onHoverDestChanged: {
        if (hoverDest === "" || plainDrops[hoverDest] !== undefined)
            return;
        const dest = hoverDest;
        const drag = generation;
        plainDrop(paths, dest, resolved => {
            if (drag !== root.generation)
                return;
            const known = Object.assign({}, root.plainDrops);
            known[dest] = resolved;
            root.plainDrops = known;
        });
    }

    function begin(selection, dir) {
        paths = selection.slice();
        sourceDir = dir;
        plainDrops = ({});
        generation++;
        active = true;
    }

    function end() {
        paths = [];
        sourceDir = "";
        plainDrops = ({});
        generation++;
        active = false;
    }

    function mimeData() {
        return {
            "text/uri-list": paths.map(path => Paths.toFileUrl(encodeURI(path))).join("\r\n") + "\r\n",
            "x-special/gnome-copied-files": "copy\n" + paths.map(path => Paths.toFileUrl(encodeURI(path))).join("\n") + "\n",
            "text/plain;charset=utf-8": paths.join("\n")
        };
    }

    function pathsFrom(drop) {
        if (active && paths.length > 0)
            return paths.slice();
        return localPaths(drop);
    }

    function localPaths(drop) {
        const urls = drop.hasUrls ? drop.urls : [];
        const out = [];
        for (const url of urls) {
            const text = url.toString();
            if (!text.startsWith("file://"))
                continue;
            out.push(decodeURIComponent(text.substring(7)));
        }
        return out;
    }

    function allInside(sources, dir) {
        for (const path of sources) {
            if (FilePaths.parentOf(path) !== dir)
                return false;
        }
        return true;
    }

    function overrideFor(proposedAction) {
        if (!active)
            return "";
        switch (proposedAction) {
        case Qt.CopyAction:
            return "copy";
        case Qt.LinkAction:
            return "ask";
        default:
            return "";
        }
    }

    function resolve(forced, plain) {
        if (forced !== "")
            return forced;
        if (SettingsData.dragAction !== "auto")
            return SettingsData.dragAction;
        return plain;
    }

    function plainDrop(sources, dest, callback) {
        const known = active ? plainDrops[dest] : undefined;
        if (known !== undefined) {
            callback(known);
            return;
        }
        transport.sendRequest("files.op.dropAction", {
            "paths": sources,
            "dest": dest
        }, response => callback((response.result || {}).action === "move" ? "move" : "copy"));
    }
}
