pragma ComponentBehavior: Bound

import QtQuick
import qs.Common

Item {
    id: splitter

    property int orientation: Qt.Horizontal
    readonly property bool stacked: orientation === Qt.Vertical

    signal moved(real delta)
    signal reset

    implicitWidth: FilesMetrics.splitterWidth
    implicitHeight: FilesMetrics.splitterWidth

    Rectangle {
        anchors.centerIn: parent
        width: splitter.stacked ? FilesMetrics.splitterHandleHeight : Theme.spacingXXS
        height: splitter.stacked ? Theme.spacingXXS : FilesMetrics.splitterHandleHeight
        radius: Theme.cornerRadiusFull
        color: handle.containsMouse || handle.pressed ? Theme.primary : Theme.outlineVariant
        opacity: handle.containsMouse || handle.pressed ? 1 : Theme.pendingOpacity

        Behavior on color {
            enabled: Theme.animationsEnabled
            ColorAnimation {
                duration: Theme.shortDuration
                easing.type: Theme.standardEasing
            }
        }
    }

    MouseArea {
        id: handle

        property real pressPosition: 0

        anchors.fill: parent
        hoverEnabled: true
        cursorShape: splitter.stacked ? Qt.SplitVCursor : Qt.SplitHCursor
        acceptedButtons: Qt.LeftButton
        onPressed: mouse => pressPosition = splitter.stacked ? mouse.y : mouse.x
        onPositionChanged: mouse => {
            if (!pressed)
                return;
            if (splitter.stacked) {
                splitter.moved(mouse.y - pressPosition);
                return;
            }
            splitter.moved(I18n.isRtl ? pressPosition - mouse.x : mouse.x - pressPosition);
        }
        onDoubleClicked: splitter.reset()
    }
}
