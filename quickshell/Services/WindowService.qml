pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell

Singleton {
    id: root

    readonly property var log: Log.scoped("WindowService")
    readonly property ListModel windows: ListModel {}
    readonly property int count: windows.count
    property int nextId: 1
    property int lastActiveId: 0
    property int primaryId: 0
    property var pendingLocations: ({})

    signal openInWindow(int windowId, var locations)
    signal windowAdded(int windowId)

    function indexOf(windowId) {
        for (let i = 0; i < windows.count; i++) {
            if (windows.get(i).windowId === windowId)
                return i;
        }
        return -1;
    }

    function ids() {
        const out = [];
        for (let i = 0; i < windows.count; i++)
            out.push(windows.get(i).windowId);
        return out;
    }

    function target() {
        if (indexOf(lastActiveId) >= 0)
            return lastActiveId;
        return windows.count > 0 ? windows.get(0).windowId : 0;
    }

    function create(locations) {
        const windowId = nextId++;
        const pending = pendingLocations;
        pending[windowId] = locations ?? [];
        pendingLocations = pending;
        if (windows.count === 0)
            primaryId = windowId;
        windows.append({
            "windowId": windowId
        });
        lastActiveId = windowId;
        log.debug(`window ${windowId} opened, ${windows.count} open`);
        windowAdded(windowId);
        return windowId;
    }

    function takePendingLocations(windowId) {
        const pending = pendingLocations;
        const locations = pending[windowId] ?? [];
        delete pending[windowId];
        pendingLocations = pending;
        return locations;
    }

    function open(locations, newWindow) {
        const existing = newWindow ? 0 : target();
        if (existing === 0)
            return create(locations);
        if (locations.length > 0)
            openInWindow(existing, locations);
        return existing;
    }

    function activate(windowId) {
        if (indexOf(windowId) < 0)
            return;
        lastActiveId = windowId;
    }

    function close(windowId) {
        const index = indexOf(windowId);
        if (index < 0)
            return;
        windows.remove(index);
        primaryId = windows.count > 0 ? windows.get(0).windowId : 0;
        log.debug(`window ${windowId} closed, ${windows.count} open`);
        if (lastActiveId === windowId)
            lastActiveId = windows.count > 0 ? windows.get(windows.count - 1).windowId : 0;
    }

    function closeAll() {
        log.debug(`closing all ${windows.count} windows`);
        windows.clear();
        primaryId = 0;
        lastActiveId = 0;
        pendingLocations = ({});
    }
}
