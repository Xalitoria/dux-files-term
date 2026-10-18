pragma ComponentBehavior: Bound

import QtQuick
import qs.Services
import qs.DCommon.FileBrowser

SnapshotModel {
    id: root

    path: RecentService.path
    sortKey: "used"
    sortDescending: true
    descendingKeys: ["used", "mtime"]

    readonly property Connections _recentLink: Connections {
        target: RecentService

        function onChanged(entries) {
            root.apply({
                "entries": entries
            });
        }
    }

    Component.onCompleted: RecentService.listeners++
    Component.onDestruction: RecentService.listeners--

    function load(callback) {
        RecentService.fetch(callback);
    }

    function compare(key, a, b) {
        switch (key) {
        case "used":
            return a.usedMs - b.usedMs;
        case "mtime":
            return a.mtimeMs - b.mtimeMs;
        case "location":
            return FilePaths.parentOf(a.path).localeCompare(FilePaths.parentOf(b.path));
        case "size":
            return a.size - b.size;
        default:
            return 0;
        }
    }

    function rowChanged(current, next) {
        return current.usedMs !== next.usedMs || current.mtimeMs !== next.mtimeMs || current.size !== next.size;
    }
}
