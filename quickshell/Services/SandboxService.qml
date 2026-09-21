pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Services

Singleton {
    id: root

    property QtObject transport: DankFilesService

    property bool loaded: false
    property bool sandboxed: false
    property string appId: ""
    property bool hostFilesystem: true
    property string hostOsRoot: ""
    property var portals: ({})
    property var features: ({})
    property var thumbnailers: []

    // Unknown features count as available so nothing flickers out at startup.
    function available(name) {
        const feature = features[name];
        return feature ? feature.available === true : true;
    }

    function degraded(name) {
        const feature = features[name];
        return feature ? feature.degraded === true : false;
    }

    function reason(name) {
        const feature = features[name];
        return feature ? (feature.reason || "") : "";
    }

    function refresh() {
        transport.sendRequest("system.capabilities", null, response => {
            if (response.error)
                return;
            root._apply(response.result || {});
        });
    }

    function _apply(data) {
        sandboxed = data.sandboxed === true;
        appId = data.appId || "";
        hostFilesystem = data.hostFilesystem === true;
        hostOsRoot = data.hostOsRoot || "";
        portals = data.portals || {};
        features = data.features || {};
        thumbnailers = data.thumbnailers || [];
        loaded = true;
    }

    Component.onCompleted: refresh()

    Connections {
        target: root.transport

        function onConnectionStateChanged() {
            root.refresh();
        }
    }
}
