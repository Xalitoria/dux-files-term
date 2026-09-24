pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import QtCore

Singleton {
    id: root

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
        return Quickshell.iconPath(iconName, true) || bundledIconPath(iconName);
    }

    function bundledIconPath(iconName: string): string {
        switch (iconName) {
        case "folder":
        case "folder-documents":
        case "folder-download":
        case "folder-music":
        case "folder-pictures":
        case "folder-publicshare":
        case "folder-remote":
        case "folder-templates":
        case "folder-videos":
            return Qt.resolvedUrl(`../assets/icons/${iconName}.svg`).toString();
        case "user-home":
        case "user-desktop":
        case "inode-directory":
            return Qt.resolvedUrl("../assets/icons/folder.svg").toString();
        default:
            return "";
        }
    }

    function copyPathToClipboard(path: string): void {
        Quickshell.clipboardText = path;
    }

    Component.onCompleted: mkdir(imagecache)
}
