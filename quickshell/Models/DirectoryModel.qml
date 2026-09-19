pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services

QtObject {
    id: root

    property var service: DirectoryService
    property string path: ""
    property bool includeHidden: false
    property string sortKey: "name"
    property bool sortDesc: false
    property bool dirsFirst: true
    property int pageSize: 500
    property string thumbnailSize: "normal"

    property bool loading: false
    property string error: ""
    property string errorCode: ""
    property int total: 0
    property string watchId: ""
    property string topic: ""
    property bool watching: false
    property bool pollOnFocus: false
    property string cursor: ""

    readonly property ListModel entries: ListModel {}
    readonly property int count: entries.count
    readonly property bool hasMore: cursor !== ""

    property int _seq: 0
    property int _firstVisible: -1
    property int _lastVisible: -1
    property int _generation: 0
    property var _handler: data => root._onEvent(data)

    signal gone
    signal listed

    readonly property Connections _daemonLink: Connections {
        target: root.service
        function onConnectedChanged() {
            if (root.service.connected)
                root.reload();
        }
    }

    onPathChanged: reload()
    Component.onCompleted: reload()
    Component.onDestruction: _release()

    function reload() {
        _release();
        if (path === "") {
            entries.clear();
            total = 0;
            return;
        }
        if (!service.connected)
            return;

        const generation = ++_generation;
        loading = true;
        service.watch(path, _options(), result => {
            if (generation !== _generation)
                return;
            loading = false;
            if (!_accept(result))
                return;

            watchId = result.watchId || "";
            topic = result.topic || "";
            watching = result.watching === true;
            pollOnFocus = result.pollOnFocus === true;
            _replace(result);
            if (topic !== "")
                service.subscribe(topic, _handler);
            listed();
        });
    }

    function loadMore() {
        if (cursor === "" || watchId === "" || loading)
            return;
        const generation = _generation;
        loading = true;
        service.page(watchId, _options(cursor), result => {
            if (generation !== _generation)
                return;
            loading = false;
            if (!_accept(result))
                return;
            cursor = result.cursor || "";
            total = result.total || 0;
            entries.append(result.entries || []);
        });
    }

    function setSort(key, desc) {
        sortKey = key;
        sortDesc = desc === true;
        _repage();
    }

    function setShowHidden(show) {
        includeHidden = show === true;
        _repage();
    }

    function refresh() {
        if (!pollOnFocus)
            return;
        _repage();
    }

    readonly property Timer _thumbnailRetry: Timer {
        interval: Anims.durShort
        onTriggered: root._requestVisibleThumbnails()
    }

    function requestThumbnails(first, last) {
        if (first < 0 || last < first)
            return;
        _firstVisible = first;
        _lastVisible = last;
        _requestVisibleThumbnails();
    }

    function _requestVisibleThumbnails() {
        const first = _firstVisible;
        const last = _lastVisible;
        if (watchId === "" || first < 0 || last < first)
            return;

        const paths = [];
        for (let i = first; i <= Math.min(last, entries.count - 1); i++) {
            const entry = entries.get(i);
            if (entry.thumbnailable && entry.thumbnail === "")
                paths.push(entry.path);
        }
        if (paths.length === 0)
            return;

        service.thumbnails(paths, thumbnailSize, watchId, result => {
            if (result.error)
                return;
            for (const item of result.results || []) {
                if (!item.thumbnail)
                    continue;
                _setThumbnail(item.path, item.thumbnail);
            }
        });
    }

    function indexOfName(name) {
        for (let i = 0; i < entries.count; i++) {
            if (entries.get(i).name === name)
                return i;
        }
        return -1;
    }

    function _options(cursorValue) {
        return {
            "includeHidden": includeHidden,
            "sortKey": sortKey,
            "sortDesc": sortDesc,
            "dirsFirst": dirsFirst,
            "limit": pageSize,
            "cursor": cursorValue || ""
        };
    }

    function _accept(result) {
        if (!result.error) {
            error = "";
            errorCode = "";
            return true;
        }
        error = result.error;
        errorCode = result.code || "";
        entries.clear();
        total = 0;
        return false;
    }

    function _repage() {
        if (watchId === "") {
            reload();
            return;
        }
        const generation = _generation;
        loading = true;
        service.page(watchId, _options(), result => {
            if (generation !== _generation)
                return;
            loading = false;
            if (!_accept(result))
                return;
            _replace(result);
            listed();
        });
    }

    function _replace(result) {
        _seq = result.seq || 0;
        entries.clear();
        entries.append(result.entries || []);
        total = result.total || 0;
        cursor = result.cursor || "";
        if (_firstVisible >= 0)
            _thumbnailRetry.restart();
    }

    function _onEvent(data) {
        if (data.watchId !== watchId)
            return;
        if (data.kind === "gone") {
            _release();
            gone();
            return;
        }
        if (data.seq !== _seq + 1) {
            _repage();
            _seq = data.seq;
            return;
        }
        _seq = data.seq;

        switch (data.kind) {
        case "batch":
            _applyBatch(data);
            break;
        case "resync":
            _repage();
            break;
        case "thumbnails":
            _applyChanged(data.thumbnails || []);
            break;
        }
    }

    function _applyBatch(data) {
        for (const name of data.removed || []) {
            const index = indexOfName(name);
            if (index < 0) {
                total = Math.max(0, total - 1);
                continue;
            }
            entries.remove(index);
            total = Math.max(0, total - 1);
        }

        _applyChanged(data.changed || []);

        const added = data.added || [];
        const at = data.addedAt || [];
        for (let i = 0; i < added.length; i++) {
            total++;
            const index = at[i] ?? entries.count;
            if (index > entries.count)
                continue;
            entries.insert(index, added[i]);
        }
    }

    function _applyChanged(changed) {
        let awaitsThumbnail = false;
        for (const entry of changed) {
            const index = indexOfName(entry.name);
            if (index < 0)
                continue;
            entries.set(index, entry);
            awaitsThumbnail = awaitsThumbnail || (entry.thumbnailable && entry.thumbnail === "");
        }
        if (awaitsThumbnail)
            _thumbnailRetry.restart();
    }

    function _setThumbnail(path, thumbnail) {
        for (let i = 0; i < entries.count; i++) {
            if (entries.get(i).path !== path)
                continue;
            entries.setProperty(i, "thumbnail", thumbnail);
            return;
        }
    }

    function _release() {
        if (topic !== "")
            service.unsubscribe(topic, _handler);
        if (watchId !== "")
            service.unwatch(watchId);
        topic = "";
        watchId = "";
        watching = false;
        pollOnFocus = false;
        cursor = "";
        _seq = 0;
    }
}
