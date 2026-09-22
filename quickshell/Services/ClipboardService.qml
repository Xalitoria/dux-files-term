pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Services

// The clipboard lives in the daemon: it is the only side that can publish the
// GNOME and KDE flavours other file managers paste, and the dfiles CLI reads
// the same selection.
Singleton {
    id: root

    property var transport: DankFilesService

    property var paths: []
    property bool cut: false
    property bool available: true

    readonly property bool hasFiles: paths.length > 0

    signal requestFailed(string message)

    function copy(selection) {
        _publish(selection, false);
    }

    function cutPaths(selection) {
        _publish(selection, true);
    }

    function read(callback) {
        transport.sendRequest("clipboard.paths", null, response => {
            if (response.error) {
                root.requestFailed(response.error);
                callback({
                    "paths": [],
                    "cut": false
                });
                return;
            }
            const result = response.result || {};
            callback({
                "paths": result.paths || [],
                "cut": result.cut === true,
                "source": result.source || ""
            });
        });
    }

    function clear() {
        paths = [];
        cut = false;
        transport.sendRequest("clipboard.clear", null, null);
    }

    function _publish(selection, asCut) {
        if (selection.length === 0)
            return;
        transport.sendRequest("clipboard.copy", {
            "paths": selection,
            "cut": asCut
        }, response => {
            if (response.error) {
                root.available = false;
                root.requestFailed(response.error);
                return;
            }
            root.available = true;
            root.paths = selection.slice();
            root.cut = asCut;
        });
    }
}
