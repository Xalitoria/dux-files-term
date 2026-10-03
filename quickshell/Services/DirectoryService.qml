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
            "trash": SandboxService.available("trash"),
            "search": DankFilesService.capabilities.includes("search")
        })

    property var userDirs: []
    property var _watchTopics: ({})
    property var _searchTopics: ({})
    readonly property var _eventHandler: data => root.watchEvent(data)
    readonly property var _searchHandler: data => root._onSearchEvent(data)

    signal watchEvent(var data)
    signal searchEvent(var data)

    readonly property string templatesDir: userDirs.find(dir => dir.key === "templates")?.path ?? ""

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

    function preview(path, maxBytes, callback) {
        _send("files.preview", {
            "path": path,
            "maxBytes": maxBytes
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

    function search(rootPath, query, options, callback) {
        const params = Object.assign({
            "root": rootPath,
            "query": query
        }, options || {});
        _send("search.start", params, result => {
            if (result.topic) {
                const topics = Object.assign({}, root._searchTopics);
                topics[result.searchId] = result.topic;
                root._searchTopics = topics;
                DankFilesService.subscribeTopic(result.topic, root._searchHandler);
            }
            callback?.(result);
        });
    }

    function cancelSearch(searchId) {
        if (!searchId)
            return;
        _forgetSearch(searchId);
        _send("search.cancel", {
            "searchId": searchId
        }, null);
    }

    function _forgetSearch(searchId) {
        const topic = _searchTopics[searchId];
        if (!topic)
            return;
        const topics = Object.assign({}, _searchTopics);
        delete topics[searchId];
        _searchTopics = topics;
        DankFilesService.unsubscribeTopic(topic, _searchHandler);
    }

    function _onSearchEvent(data) {
        searchEvent(data);
        if (data.kind === "done")
            _forgetSearch(data.searchId);
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
            for (const searchId in root._searchTopics)
                DankFilesService.unsubscribeTopic(root._searchTopics[searchId], root._searchHandler);
            root._searchTopics = ({});
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
