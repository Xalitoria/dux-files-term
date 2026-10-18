pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Services

Singleton {
    id: root

    // org.freedesktop.appearance color-scheme: 0 = no preference, 1 = prefer dark, 2 = prefer light
    property int systemColorScheme: 0
    property bool available: false

    readonly property bool systemPrefersLight: available && systemColorScheme !== 1

    function _apply(data) {
        if (!data)
            return;
        available = data.available === true;
        systemColorScheme = data.colorScheme || 0;
    }

    function refresh() {
        DuxFilesService.call("system.colorScheme.get", null, result => {
            if (result.error)
                return;
            root._apply(result);
        });
    }

    Component.onCompleted: {
        if (DuxFilesService.connected)
            refresh();
    }

    Connections {
        target: DuxFilesService

        function onColorSchemeUpdate(data) {
            root._apply(data);
        }

        function onConnectedChanged() {
            if (DuxFilesService.connected)
                root.refresh();
        }
    }
}
