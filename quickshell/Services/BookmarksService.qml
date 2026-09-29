pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Common
import qs.Services

Singleton {
    id: root

    property var transport: DankFilesService

    property var entries: []
    property bool loaded: false
    property bool gtkImportAvailable: false
    property bool importOffer: false

    readonly property var paths: entries.map(entry => entry.path)

    signal requestFailed(string message)

    property var _handler: data => root._onEvent(data)
    property var _subscribed: null

    Component.onCompleted: _attach()
    onTransportChanged: _attach()

    readonly property Connections _daemonLink: Connections {
        target: root.transport

        function onConnectedChanged() {
            if (root.transport.connected)
                root.refresh();
        }
    }

    function _attach() {
        if (_subscribed === transport)
            return;
        if (_subscribed)
            _subscribed.unsubscribeTopic("bookmarks", _handler);
        _subscribed = transport;
        transport.subscribeTopic("bookmarks", _handler);
        if (transport.connected)
            refresh();
    }

    function has(path) {
        return paths.includes(path);
    }

    function labelFor(path) {
        return entries.find(entry => entry.path === path)?.label ?? "";
    }

    function refresh() {
        transport.sendRequest("bookmarks.list", null, response => {
            if (response.error)
                return;
            const result = response.result || {};
            root.entries = result.bookmarks || [];
            root.gtkImportAvailable = result.gtkImportAvailable === true;
            root.loaded = true;
            root.importOffer = root.gtkImportAvailable && !SettingsData.gtkBookmarksOffered;
        });
    }

    function isPlace(path) {
        return DirectoryService.userDirs.some(dir => dir.path === path);
    }

    function add(path, label) {
        if (path === "" || has(path) || isPlace(path))
            return;
        _call("bookmarks.add", {
            "path": path,
            "label": label || ""
        });
    }

    function remove(path) {
        _call("bookmarks.remove", {
            "path": path
        });
    }

    function toggle(path) {
        if (path === "")
            return;
        if (has(path)) {
            remove(path);
            return;
        }
        add(path, "");
    }

    function rename(path, label) {
        _call("bookmarks.rename", {
            "path": path,
            "label": label
        });
    }

    function reorder(orderedPaths) {
        _call("bookmarks.reorder", {
            "paths": orderedPaths
        });
    }

    function importGtk(callback) {
        declineImport();
        transport.sendRequest("bookmarks.importGtk", null, response => {
            if (response.error) {
                root.requestFailed(response.error);
                callback?.(0);
                return;
            }
            const result = response.result || {};
            root.entries = result.bookmarks || [];
            root.gtkImportAvailable = false;
            callback?.(result.added || 0);
        });
    }

    function declineImport() {
        importOffer = false;
        SettingsData.gtkBookmarksOffered = true;
    }

    function _call(method, params) {
        transport.sendRequest(method, params, response => {
            if (response.error) {
                root.requestFailed(response.error);
                return;
            }
            root.entries = (response.result || {}).bookmarks || [];
            root.gtkImportAvailable = false;
        });
    }

    function _onEvent(data) {
        if (data.kind !== "changed")
            return;
        entries = data.bookmarks || [];
    }
}
