pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Common

// One drag at a time, so the payload lives here instead of being re-parsed out
// of the mime data by every drop target inside the window. Drops from other
// applications carry no payload and fall back to their uri-list.
Singleton {
    id: root

    property var paths: []
    property string sourceDir: ""
    property bool active: false
    property string action: "move"

    function begin(selection, dir) {
        paths = selection.slice();
        sourceDir = dir;
        action = "move";
        active = true;
    }

    function end() {
        paths = [];
        sourceDir = "";
        active = false;
    }

    function mimeData() {
        return {
            "text/uri-list": paths.map(path => Paths.toFileUrl(encodeURI(path))).join("\r\n") + "\r\n",
            "x-special/gnome-copied-files": "copy\n" + paths.map(path => Paths.toFileUrl(encodeURI(path))).join("\n") + "\n",
            "text/plain;charset=utf-8": paths.join("\n")
        };
    }

    // Internal drops keep the exact paths; a foreign drop is whatever local
    // files its uri-list names.
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

    // Ctrl forces a copy, Shift forces a move, Alt asks; with no modifier the
    // daemon decides from the device the sources sit on.
    function actionFor(modifiers) {
        switch (true) {
        case (modifiers & Qt.ControlModifier) !== 0:
            return "copy";
        case (modifiers & Qt.ShiftModifier) !== 0:
            return "move";
        case (modifiers & Qt.AltModifier) !== 0:
            return "ask";
        default:
            return "";
        }
    }
}
