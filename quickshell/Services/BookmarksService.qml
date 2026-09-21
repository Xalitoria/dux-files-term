pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Common

Singleton {
    id: root

    readonly property var paths: SettingsData.bookmarks

    function has(path) {
        return paths.includes(path);
    }

    function add(path) {
        if (path === "" || has(path))
            return;
        SettingsData.bookmarks = paths.concat([path]);
    }

    function remove(path) {
        SettingsData.bookmarks = paths.filter(entry => entry !== path);
    }

    function toggle(path) {
        if (path === "")
            return;
        if (has(path)) {
            remove(path);
            return;
        }
        add(path);
    }
}
