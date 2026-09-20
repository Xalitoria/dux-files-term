pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

StyledButton {
    id: root

    property string iconName: ""
    property bool submenu: false

    signal entered

    width: parent?.width ?? 0
    implicitHeight: Theme.menuItemHeight
    radius: Theme.cornerRadiusS
    color: "transparent"
    onHoveredChanged: {
        if (hovered)
            root.entered();
    }

    StateLayer {
        control: root
        disabled: !root.enabled
        hovered: root.hovered
        stateColor: Theme.surfaceText
    }

    Row {
        anchors.fill: parent
        anchors.leftMargin: Theme.spacingM
        anchors.rightMargin: Theme.spacingM
        spacing: Theme.spacingM

        DankIcon {
            anchors.verticalCenter: parent.verticalCenter
            visible: root.iconName !== ""
            name: root.iconName
            size: Theme.fontSizeLarge
            color: root.enabled ? Theme.surfaceVariantText : Theme.onSurface_38
        }

        StyledText {
            anchors.verticalCenter: parent.verticalCenter
            width: parent.width - x - (root.submenu ? Theme.fontSizeLarge + Theme.spacingM : 0)
            text: root.text
            color: root.enabled ? Theme.surfaceText : Theme.onSurface_38
            font.pixelSize: Theme.fontSizeMedium
            elide: Text.ElideRight
        }
    }

    DankIcon {
        anchors.right: parent.right
        anchors.rightMargin: Theme.spacingM
        anchors.verticalCenter: parent.verticalCenter
        visible: root.submenu
        name: "chevron_right"
        size: Theme.fontSizeLarge
        color: Theme.surfaceVariantText
    }
}
