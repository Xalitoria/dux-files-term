pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Services

Singleton {
    id: root

    readonly property bool connected: DankFilesService.connected

    function watch(path, options, callback) {
        _send("files.watch", Object.assign({
            "path": path
        }, _options(options)), callback);
    }

    function unwatch(watchId) {
        if (!watchId)
            return;
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

    function count(paths, includeHidden, callback) {
        _send("files.count", {
            "paths": paths,
            "includeHidden": includeHidden === true
        }, callback);
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

    function subscribe(topic, handler) {
        DankFilesService.subscribeTopic(topic, handler);
    }

    function unsubscribe(topic, handler) {
        DankFilesService.unsubscribeTopic(topic, handler);
    }

    function _options(options) {
        const opts = options || {};
        const params = {};
        if (opts.includeHidden !== undefined)
            params.includeHidden = opts.includeHidden;
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
