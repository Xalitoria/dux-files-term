pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DCommon.Widgets

StyledButton {
    id: tab

    focusPolicy: Qt.TabFocus

    property string label: ""
    property bool current: false
    property bool closable: true

    signal closeRequested
    signal dropHovered

    readonly property color contentColor: current ? Theme.primary : Theme.onSurfaceVariant

    implicitWidth: Math.max(FilesMetrics.tabMinWidth, Math.min(FilesMetrics.tabMaxWidth, title.implicitWidth + closeButton.width + Theme.spacingL))
    implicitHeight: FilesMetrics.tabHeight
    radius: Theme.cornerRadiusM
    color: dropZone.containsDrag ? Theme.primaryContainer : "transparent"
    Accessible.role: Accessible.PageTab
    Accessible.name: label
    Accessible.selected: current

    StateLayer {
        control: tab
        stateColor: tab.contentColor
    }

    DropArea {
        id: dropZone

        anchors.fill: parent
        onEntered: hoverSwitch.restart()
        onExited: hoverSwitch.stop()
        onDropped: hoverSwitch.stop()
    }

    Timer {
        id: hoverSwitch

        interval: FilesMetrics.tabHoverDelay
        onTriggered: tab.dropHovered()
    }

    TapHandler {
        acceptedButtons: Qt.MiddleButton
        onTapped: {
            if (tab.closable)
                tab.closeRequested();
        }
    }

    FocusRing {
        visible: tab.visualFocus
    }

    StyledText {
        id: title

        anchors.left: parent.left
        anchors.leftMargin: Theme.spacingM
        anchors.right: closeButton.left
        anchors.rightMargin: Theme.spacingXXS
        anchors.verticalCenter: parent.verticalCenter
        text: tab.label
        color: tab.contentColor
        font.pixelSize: Theme.fontSizeSmall
        font.weight: tab.current ? Theme.fontWeightMedium : Theme.fontWeight
        elide: Text.ElideMiddle
    }

    DActionButton {
        id: closeButton

        anchors.right: parent.right
        anchors.rightMargin: Theme.spacingXXS
        anchors.verticalCenter: parent.verticalCenter
        visible: tab.closable
        width: visible ? Theme.buttonHeightXXS : 0
        buttonSize: Theme.buttonHeightXXS
        iconName: "close"
        iconSize: Theme.iconSizeSmall
        iconColor: tab.contentColor
        tooltipText: I18n.tr("Close tab", "tooltip on the button closing a tab")
        onClicked: tab.closeRequested()
    }

    Rectangle {
        anchors.bottom: parent.bottom
        anchors.horizontalCenter: parent.horizontalCenter
        width: tab.current ? Math.max(Theme.tabIndicatorMinWidth, tab.width - Theme.spacingL) : 0
        height: Theme.tabIndicatorHeight
        radius: Theme.cornerRadiusFull
        color: Theme.primary

        Behavior on width {
            enabled: Theme.animationsEnabled
            NumberAnimation {
                duration: Theme.expressiveDurations.expressiveFastSpatial
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Theme.expressiveCurves.expressiveFastSpatial
            }
        }
    }
}
