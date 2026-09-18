pragma ComponentBehavior: Bound

import QtQuick
import qs.Services

QtObject {
    id: root

    property string path: ""
    property bool includeHidden: false
    property bool loading: false
    property string error: ""
    property string errorCode: ""
    property int _generation: 0

    readonly property ListModel entries: ListModel {}
    readonly property int count: entries.count

    readonly property Connections _daemonLink: Connections {
        target: DankFilesService
        function onConnectionStateChanged() {
            if (DankFilesService.connected)
                root.reload();
        }
    }

    onPathChanged: reload()
    onIncludeHiddenChanged: reload()
    Component.onCompleted: reload()

    function reload() {
        if (path === "") {
            entries.clear();
            return;
        }
        if (!DankFilesService.connected)
            return;
        const generation = ++_generation;
        loading = true;
        DirectoryService.list(path, includeHidden, result => {
            if (generation !== _generation)
                return;
            loading = false;
            entries.clear();
            if (result.error) {
                error = result.error;
                errorCode = result.code;
                return;
            }
            error = "";
            errorCode = "";
            entries.append(result.entries);
        });
    }
}
