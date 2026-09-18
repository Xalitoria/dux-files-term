pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import Quickshell.Io
import QtCore

Singleton {
    id: root

    property var _trashCallback: null

    readonly property url home: StandardPaths.standardLocations(StandardPaths.HomeLocation)[0]
    readonly property url xdgCache: StandardPaths.standardLocations(StandardPaths.GenericCacheLocation)[0]
    readonly property url cache: `${xdgCache}/dfiles`
    readonly property url imagecache: `${cache}/imagecache`

    function stringify(path: url): string {
        return path.toString().replace(/%20/g, " ");
    }

    function strip(path: url): string {
        return stringify(path).replace("file://", "");
    }

    function expandTilde(path: string): string {
        if (!path.startsWith("~"))
            return path;
        return strip(root.home) + path.substring(1);
    }

    function shortenHome(path: string): string {
        return path.replace(strip(root.home), "~");
    }

    function toFileUrl(path: string): string {
        return path.startsWith("file://") ? path : "file://" + path;
    }

    function mkdir(path: url): void {
        Quickshell.execDetached(["mkdir", "-p", strip(path)]);
    }

    function resolveIconPath(iconName: string): string {
        if (!iconName)
            return "";
        return Quickshell.iconPath(iconName, true) || "";
    }

    function trashPath(path: string, callback): void {
        if (!path)
            return;
        _trashCallback = callback ?? null;
        trashProcess.targetPath = path;
        trashProcess.running = true;
    }

    function copyPathToClipboard(path: string): void {
        Quickshell.clipboardText = path;
    }

    Process {
        id: trashProcess

        property string targetPath: ""

        command: ["gio", "trash", targetPath]
        onExited: exitCode => {
            const cb = root._trashCallback;
            root._trashCallback = null;
            if (cb)
                cb(exitCode === 0);
        }
    }

    Component.onCompleted: mkdir(imagecache)
}
