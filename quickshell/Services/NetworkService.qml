pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Services

Singleton {
    id: root

    property var transport: DankFilesService

    property var mounts: []
    property var portables: []
    property var servers: []
    property var schemes: []
    property bool available: false
    property string reason: ""
    property bool connecting: false

    readonly property var networkMounts: mounts.filter(mount => mount.kind === "network")
    readonly property var deviceMounts: mounts.filter(mount => mount.kind === "device")

    signal askRequested(var ask)
    signal requestFailed(string message, string code)
    signal connected(var mount)

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
            _subscribed.unsubscribeTopic("network", _handler);
        _subscribed = transport;
        transport.subscribeTopic("network", _handler);
        if (transport.connected)
            refresh();
    }

    function refresh() {
        transport.sendRequest("network.mounts", null, response => {
            if (response.error)
                return;
            root._applyMounts(response.result || {});
        });
        refreshRecent();
    }

    function refreshRecent() {
        transport.sendRequest("network.recent", null, response => {
            if (response.error)
                return;
            root.servers = (response.result || {}).servers || [];
        });
    }

    // connectTo blocks in the daemon until gvfs is done, which includes every
    // credential round trip the ask dialog answers.
    function connectTo(uri, callback, password) {
        connecting = true;
        const params = {
            "uri": uri
        };
        if (password)
            params.password = password;
        _send("network.connect", params, result => {
            root.connecting = false;
            if (!result.error && result.mount) {
                if (!root.mounts.some(mount => mount.id === result.mount.id))
                    root.mounts = root.mounts.concat([result.mount]);
                root.connected(result.mount);
                root.refreshRecent();
            }
            if (callback)
                callback(result);
        });
    }

    function answer(id, params) {
        const request = Object.assign({
            "id": id
        }, params || {});
        _send("network.answer", request, null);
    }

    function abort(id) {
        answer(id, {
            "abort": true
        });
    }

    function disconnect(id, callback) {
        _send("network.disconnect", {
            "id": id
        }, callback);
    }

    function forget(uri) {
        transport.sendRequest("network.forget", {
            "uri": uri
        }, response => {
            if (response.error)
                return;
            root.servers = (response.result || {}).servers || [];
        });
    }

    function mountPortable(monitor, id, callback) {
        _send("network.mountPortable", {
            "monitor": monitor,
            "id": id
        }, callback);
    }

    function _onEvent(data) {
        switch (data.kind) {
        case "ask":
            askRequested(data);
            return;
        case "mounts":
            _applyMounts(data);
            return;
        }
    }

    function _applyMounts(data) {
        available = data.available === true;
        reason = data.reason || "";
        schemes = data.schemes || [];
        mounts = data.mounts || [];
        if (data.portables !== undefined)
            portables = data.portables || [];
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
