pragma ComponentBehavior: Bound

import QtQuick

QtObject {
    id: root

    property var transport
    property string topic
    property var handler: null

    signal connected

    readonly property var _handler: data => root.handler?.(data)
    property var _subscribed: null

    readonly property Connections _link: Connections {
        target: root.transport

        function onConnectedChanged() {
            if (root.transport.connected)
                root.connected();
        }
    }

    Component.onCompleted: _attach()
    onTransportChanged: _attach()

    function _attach() {
        if (!transport || topic === "" || _subscribed === transport)
            return;
        _subscribed?.unsubscribeTopic(topic, _handler);
        _subscribed = transport;
        transport.subscribeTopic(topic, _handler);
        if (transport.connected)
            connected();
    }
}
