pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Common
import qs.Services
import qs.DCommon.FileBrowser

Singleton {
    id: root

    readonly property string path: "recent:///"
    readonly property bool shown: SettingsData.recentFiles
    property string stamp: ""
    property int count: -1
    property int listeners: 0
    property var _waiting: []

    signal changed(var entries)
    signal requestFailed(string message, string code)

    function refresh() {
        checkDelay.restart();
    }

    function invalidate() {
        stamp = "";
        count = -1;
        refresh();
    }

    function fetch(callback) {
        _waiting.push(callback);
        if (_waiting.length > 1)
            return;
        DankFilesService.call("recent.entries", {}, result => {
            if (!result.error) {
                root.stamp = result.stamp ?? "";
                root.count = (result.entries || []).length;
            }
            const waiting = root._waiting;
            root._waiting = [];
            for (const pending of waiting)
                pending(result);
        });
    }

    function remove(paths) {
        if (paths.length === 0)
            return;
        DankFilesService.call("recent.remove", {
            "paths": paths
        }, result => root._edited(result));
    }

    function clear() {
        DankFilesService.call("recent.clear", {}, result => root._edited(result));
    }

    function affects(job) {
        switch (job.kind) {
        case "move":
        case "rename":
        case "newFolderWith":
        case "trash":
        case "delete":
        case "restore":
        case "emptyTrash":
            return true;
        default:
            return false;
        }
    }

    function _edited(result) {
        if (result.error) {
            requestFailed(result.error, result.code ?? "");
            return;
        }
        invalidate();
    }

    function _check() {
        if (!DankFilesService.connected || listeners === 0)
            return;
        DankFilesService.call("recent.entries", {
            "stamp": stamp
        }, result => {
            if (result.error || result.unchanged)
                return;
            root.stamp = result.stamp ?? "";
            root.count = (result.entries || []).length;
            root.changed(result.entries || []);
        });
    }

    Connections {
        target: DankFilesService

        function onConnectedChanged() {
            root.invalidate();
        }
    }

    Connections {
        target: FileOpsService

        function onJobFinished(job) {
            if (job.state === "done" && root.affects(job))
                root.invalidate();
        }
    }

    Timer {
        id: checkDelay

        interval: FileBrowserMetrics.settleInterval
        onTriggered: root._check()
    }
}
