pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import Quickshell.Io

Singleton {
    id: root

    property alias fileBrowserSettings: adapter.fileBrowserSettings
    property alias wallpaperLastPath: adapter.wallpaperLastPath
    property alias profileLastPath: adapter.profileLastPath
    property alias sidebarWidth: adapter.sidebarWidth
    property alias previewWidth: adapter.previewWidth

    function saveCache() {
        if (!cacheFile.loaded)
            return;
        cacheFile.writeAdapter();
    }

    Component.onCompleted: Paths.mkdir(Paths.cache)

    FileView {
        id: cacheFile

        property bool loaded: false

        path: Paths.strip(Paths.cache) + "/cache.json"
        blockLoading: true
        blockWrites: true
        atomicWrites: true
        printErrors: false
        onLoaded: loaded = true
        onLoadFailed: loaded = true
        onAdapterUpdated: root.saveCache()

        JsonAdapter {
            id: adapter

            property var fileBrowserSettings: ({})
            property string wallpaperLastPath: ""
            property string profileLastPath: ""
            property int sidebarWidth: FilesMetrics.sidebarWidth
            property int previewWidth: 0
        }
    }
}
