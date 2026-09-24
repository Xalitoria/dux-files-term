pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Services

Singleton {
    id: root

    readonly property bool connected: DankFilesService.connected
    readonly property var capabilities: ({
            "mkdir": true,
            "rename": true,
            "trash": SandboxService.available("trash")
        })

    property var userDirs: []
    property var _watchTopics: ({})
    readonly property var _eventHandler: data => root.watchEvent(data)

    signal watchEvent(var data)

    function refreshUserDirs() {
        _send("files.userDirs", {}, response => {
            if (response.error)
                return;
            root.userDirs = response.dirs || [];
        });
    }

    function watch(path, options, callback) {
        _send("files.watch", Object.assign({
            "path": path
        }, _options(options)), result => {
            if (result.topic) {
                const topics = Object.assign({}, root._watchTopics);
                topics[result.watchId] = result.topic;
                root._watchTopics = topics;
                DankFilesService.subscribeTopic(result.topic, root._eventHandler);
            }
            callback?.(result);
        });
    }

    function unwatch(watchId) {
        if (!watchId)
            return;
        const topic = _watchTopics[watchId];
        if (topic) {
            const topics = Object.assign({}, _watchTopics);
            delete topics[watchId];
            _watchTopics = topics;
            DankFilesService.unsubscribeTopic(topic, _eventHandler);
        }
        _send("files.unwatch", {
            "watchId": watchId
        }, null);
    }

    function list(path, options, callback) {
        _send("files.list", Object.assign({
            "path": path
        }, _options(options)), callback);
    }

    function page(watchId, options, callback) {
        _send("files.list", Object.assign({
            "watchId": watchId
        }, _options(options)), callback);
    }

    function stat(path, callback) {
        _send("files.stat", {
            "path": path
        }, callback);
    }

    function count(paths, includeHidden, callback) {
        _send("files.count", {
            "paths": paths,
            "includeHidden": includeHidden === true
        }, callback);
    }

    function scanSize(paths, callback) {
        _send("files.size", {
            "paths": paths
        }, callback);
    }

    function cancelScan(scanId) {
        if (!scanId)
            return;
        _send("files.sizeCancel", {
            "scanId": scanId
        }, null);
    }

    function thumbnails(paths, size, watchId, callback) {
        const params = {
            "paths": paths,
            "size": size || "normal"
        };
        if (watchId)
            params.watchId = watchId;
        _send("files.thumbnail", params, callback);
    }

    function icons(paths, callback) {
        _send("files.icon", {
            "paths": paths
        }, callback);
    }

    function mkdir(path, callback) {
        _send("files.mkdir", {
            "path": path
        }, callback);
    }

    function rename(path, name, callback) {
        _send("files.rename", {
            "path": path,
            "name": name
        }, callback);
    }

    function trash(paths, callback) {
        _send("files.trash", {
            "paths": paths
        }, callback);
    }

    function _options(options) {
        const opts = options || {};
        const params = {};
        if (opts.includeHidden !== undefined)
            params.includeHidden = opts.includeHidden;
        if (opts.filters !== undefined)
            params.filters = opts.filters;
        if (opts.sortKey)
            params.sort = opts.sortKey;
        if (opts.sortDesc !== undefined)
            params.desc = opts.sortDesc;
        if (opts.dirsFirst !== undefined)
            params.dirsFirst = opts.dirsFirst;
        if (opts.limit)
            params.limit = opts.limit;
        if (opts.cursor)
            params.cursor = opts.cursor;
        return params;
    }

    Component.onCompleted: refreshUserDirs()

    Connections {
        target: DankFilesService

        function onConnectionStateChanged() {
            for (const watchId in root._watchTopics)
                DankFilesService.unsubscribeTopic(root._watchTopics[watchId], root._eventHandler);
            root._watchTopics = ({});
            if (DankFilesService.connected)
                root.refreshUserDirs();
        }
    }

    function _send(method, params, callback) {
        DankFilesService.sendRequest(method, params, response => {
            if (!callback)
                return;
            if (response.error) {
                callback({
                    "error": response.error,
                    "code": response.code || ""
                });
                return;
            }
            callback(response.result || {});
        });
    }
}
