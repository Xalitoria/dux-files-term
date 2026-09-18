pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import Quickshell.Io
import qs.DankCommon.Common
import qs.Services

Singleton {
    id: root

    readonly property var log: Log.scoped("DankFilesService")

    readonly property string socketPath: Quickshell.env("DFILES_SOCKET")

    property bool socketReady: false
    property bool connected: false
    property bool connecting: false
    property bool subscribed: false
    property int apiVersion: 0
    property string daemonVersion: ""
    property var capabilities: []
    property string lastError: ""

    property var pendingRequests: ({})
    property var topicHandlers: ({})
    property int requestCounter: 0

    signal connectionStateChanged
    signal windowActionRequested(string action)
    signal openPathRequested(string path)
    signal colorSchemeUpdate(var data)

    Component.onCompleted: {
        subscribeTopic("ui", _onUiEvent);
        subscribeTopic("colorScheme", data => root.colorSchemeUpdate(data));
        if (socketPath && socketPath.length > 0)
            socketProbe.running = true;
    }

    Process {
        id: socketProbe
        command: ["test", "-S", root.socketPath]
        running: false
        onExited: code => {
            root.socketReady = code === 0;
            if (root.socketReady)
                root.connectSocket();
        }
    }

    function connectSocket() {
        if (!socketReady || connected || connecting)
            return;
        connecting = true;
        requestSocket.connected = true;
    }

    DankSocket {
        id: requestSocket
        path: root.socketPath
        connected: false

        onConnectionStateChanged: {
            root.connecting = false;
            root.connected = connected;
            root.connectionStateChanged();
            if (connected) {
                subscribeSocket.connected = true;
                root.refreshVersion();
                return;
            }
            root._failPendingRequests("disconnected from dfiles socket");
        }

        parser: SplitParser {
            onRead: line => {
                if (!line || line.length === 0)
                    return;
                let response;
                try {
                    response = JSON.parse(line);
                } catch (e) {
                    root.log.warn("bad response", line.substring(0, 200));
                    return;
                }
                root._handleResponse(response);
            }
        }
    }

    DankSocket {
        id: subscribeSocket
        path: root.socketPath
        connected: false

        onConnectionStateChanged: {
            root.subscribed = connected;
            if (connected)
                root._sendSubscribe(Object.keys(root.topicHandlers));
        }

        parser: SplitParser {
            onRead: line => {
                if (!line || line.length === 0)
                    return;
                let event;
                try {
                    event = JSON.parse(line);
                } catch (e) {
                    return;
                }
                root._handleEvent(event);
            }
        }
    }

    function subscribeTopic(topic, handler) {
        const handlers = topicHandlers[topic] || [];
        handlers.push(handler);
        topicHandlers[topic] = handlers;
        if (subscribed && handlers.length === 1)
            _sendSubscribe([topic]);
    }

    function unsubscribeTopic(topic, handler) {
        const handlers = topicHandlers[topic];
        if (!handlers)
            return;
        const index = handlers.indexOf(handler);
        if (index < 0)
            return;
        handlers.splice(index, 1);
        if (handlers.length > 0)
            return;
        delete topicHandlers[topic];
        if (subscribed)
            _sendUnsubscribe([topic]);
    }

    function _sendSubscribe(topics) {
        if (topics.length === 0)
            return;
        subscribeSocket.send({
            "id": _nextId(),
            "method": "subscribe",
            "params": {
                "topics": topics
            }
        });
    }

    function _sendUnsubscribe(topics) {
        subscribeSocket.send({
            "id": _nextId(),
            "method": "unsubscribe",
            "params": {
                "topics": topics
            }
        });
    }

    function _nextId() {
        requestCounter++;
        return Date.now() + requestCounter;
    }

    function _handleResponse(response) {
        if (response.event) {
            _handleEvent(response);
            return;
        }

        const id = response.id;
        if (!id) {
            if (response.apiVersion !== undefined) {
                apiVersion = response.apiVersion;
                capabilities = response.capabilities || [];
            }
            return;
        }

        const cb = pendingRequests[id];
        if (!cb)
            return;
        delete pendingRequests[id];
        cb(response);
    }

    function _handleEvent(event) {
        const handlers = topicHandlers[event.event];
        if (!handlers)
            return;
        const data = event.data || {};
        for (const handler of handlers.slice())
            handler(data);
    }

    function _onUiEvent(data) {
        if (data.action === "open") {
            openPathRequested(data.path || "");
            return;
        }
        windowActionRequested(data.action || "");
    }

    function _failPendingRequests(reason) {
        const callbacks = pendingRequests;
        pendingRequests = ({});
        for (const id in callbacks) {
            callbacks[id]({
                "error": reason
            });
        }
    }

    function sendRequest(method, params, callback) {
        if (!connected) {
            if (callback)
                callback({
                    "error": "not connected to dfiles socket"
                });
            return;
        }

        const id = _nextId();
        const req = {
            "id": id,
            "method": method
        };
        if (params)
            req.params = params;
        if (callback)
            pendingRequests[id] = callback;
        requestSocket.send(req);
    }

    function refreshVersion() {
        sendRequest("version", null, response => {
            if (response.error)
                return;
            const info = response.result || {};
            daemonVersion = info.version || "";
            if (info.apiVersion)
                apiVersion = info.apiVersion;
        });
    }

    function openUri(url) {
        sendRequest("system.openUri", {
            "uri": url
        }, response => {
            if (response.error)
                Qt.openUrlExternally(url);
        });
    }

    function quit() {
        if (connected) {
            sendRequest("ui.quit", null, null);
            return;
        }
        Qt.quit();
    }
}
