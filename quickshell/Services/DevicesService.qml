pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Services

Singleton {
    id: root

    property var transport: DuxFilesService

    property var devices: []
    property bool available: false
    property string reason: ""
    property var usage: ({})

    readonly property var systemDevice: devices.find(device => device.system) ?? null

    signal requestFailed(string message, string code)

    readonly property DaemonTopic _devices: DaemonTopic {
        transport: root.transport
        topic: "devices"
        handler: data => root._onEvent(data)
        onConnected: root.refresh()
    }

    function refresh() {
        transport.call("devices.list", null, result => {
            if (result.error)
                return;
            root._onEvent(result);
        });
    }

    function refreshUsage(force) {
        const paths = [];
        for (const device of devices) {
            const mounts = device.mountPoints?.length ? device.mountPoints : device.mounted ? [device.mountPoint] : [];
            for (const path of mounts) {
                if (!paths.includes(path))
                    paths.push(path);
            }
        }
        if (!paths.includes("/"))
            paths.unshift("/");
        transport.call("devices.usage", {
            "paths": paths,
            "force": force === true
        }, result => {
            if (result.error)
                return;
            const next = ({});
            for (const entry of result.usage || [])
                next[entry.mountPoint] = entry;
            root.usage = next;
        });
    }

    function usageRatio(mountPoint) {
        const entry = usage[mountPoint];
        if (!entry || !entry.total)
            return -1;
        return entry.used / entry.total;
    }

    function mount(id, callback) {
        _send("devices.mount", {
            "id": id
        }, callback);
    }

    function unmount(id, force, callback) {
        _send("devices.unmount", {
            "id": id,
            "force": force === true
        }, callback);
    }

    function eject(id, callback) {
        _send("devices.eject", {
            "id": id
        }, callback);
    }

    function powerOff(id, callback) {
        _send("devices.powerOff", {
            "id": id
        }, callback);
    }

    function unlock(id, passphrase, callback) {
        _send("devices.unlock", {
            "id": id,
            "passphrase": passphrase,
            "mount": true
        }, callback);
    }

    function lock(id, callback) {
        _send("devices.lock", {
            "id": id
        }, callback);
    }

    function _onEvent(data) {
        available = data.available === true;
        reason = data.reason || "";
        devices = data.devices || [];
        refreshUsage(false);
    }

    function _send(method, params, callback) {
        transport.call(method, params, result => {
            if (result.error)
                root.requestFailed(result.error, result.code);
            callback?.(result);
        });
    }
}
