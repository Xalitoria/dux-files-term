pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell

Singleton {
    id: root

    property string message: ""
    property string actionLabel: ""
    property var action: null
    property bool active: false

    function show(msg, opts) {
        opts = opts || {};
        message = msg;
        actionLabel = opts.actionLabel || "";
        action = (typeof opts.action === "function") ? opts.action : null;
        active = true;
        timer.interval = opts.duration || (actionLabel !== "" ? 6000 : 2500);
        timer.restart();
    }

    function info(msg) {
        show(msg, {});
    }

    function runAction() {
        const pending = action;
        dismiss();
        if (pending)
            pending();
    }

    function dismiss() {
        active = false;
        actionLabel = "";
        action = null;
        timer.stop();
    }

    readonly property Timer _timeout: Timer {
        id: timer
        repeat: false
        onTriggered: root.dismiss()
    }
}
