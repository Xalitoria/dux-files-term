pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Services

Singleton {
    id: root

    property var transport: DankFilesService

    property var devices: []
    property bool available: false
    property string reason: ""
    property var usage: ({})

    readonly property var systemDevice: devices.find(device => device.system) ?? null
    readonly property bool hasDevices: devices.length > 0

    signal requestFailed(string message, string code)

    property var _handler: data => root._onEvent(data)
    property var _subscribed: null

    Component.onCompleted: _attach()
    onTransportChanged: _attach()

    readonly property Connections _daemonLink: Connections {
        target: root.transport
        function onConnectedChanged() {
            if (root.transport.connected)
                root.refresh();
        }
    }

    function _attach() {
        if (_subscribed === transport)
            return;
        if (_subscribed)
            _subscribed.unsubscribeTopic("devices", _handler);
        _subscribed = transport;
        transport.subscribeTopic("devices", _handler);
        if (transport.connected)
            refresh();
    }

    function refresh() {
        transport.sendRequest("devices.list", null, response => {
            if (response.error)
                return;
            root._onEvent(response.result || {});
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
        transport.sendRequest("devices.usage", {
            "paths": paths,
            "force": force === true
        }, response => {
            if (response.error)
                return;
            const next = ({});
            for (const entry of (response.result || {}).usage || [])
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
        transport.sendRequest(method, params, response => {
            if (response.error) {
                root.requestFailed(response.error, response.code || "");
                if (callback)
                    callback({
                        "error": response.error,
                        "code": response.code || ""
                    });
                return;
            }
            if (callback)
                callback(response.result || {});
        });
    }
}
