pragma ComponentBehavior: Bound

import QtQuick
import qs.Services

SnapshotModel {
    id: root

    path: TrashService.filesPath
    sortKey: "deleted"
    sortDescending: true
    descendingKeys: ["deleted"]

    readonly property Connections _trashLink: Connections {
        target: TrashService

        function onChanged() {
            root.refresh();
        }
    }

    function load(callback) {
        TrashService.entries(callback);
    }

    function compare(key, a, b) {
        switch (key) {
        case "origin":
            return a.originalPath.localeCompare(b.originalPath);
        case "deleted":
            return a.deletedMs - b.deletedMs;
        case "size":
            return a.size - b.size;
        default:
            return 0;
        }
    }

    function rowChanged(current, next) {
        return current.deletedMs !== next.deletedMs;
    }
}
