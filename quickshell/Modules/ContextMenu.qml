pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Controls
import qs.Common
import qs.DankCommon.Widgets

Popup {
    id: root

    property Item anchorItem: parent
    default property alias rows: column.children

    function openAt(pointX, pointY) {
        const limits = anchorItem ?? parent;
        x = Math.max(0, Math.min(pointX, (limits?.width ?? width) - width));
        y = Math.max(0, Math.min(pointY, (limits?.height ?? height) - height));
        open();
    }

    width: Theme.iconSize * 9
    padding: Theme.spacingXS
    modal: false
    focus: true
    closePolicy: Popup.CloseOnEscape | Popup.CloseOnPressOutsideParent

    background: Rectangle {
        radius: Theme.cornerRadiusL
        color: Theme.surfaceContainer
        border.width: Theme.outlineWidth
        border.color: Theme.outlineVariant
    }

    contentItem: Column {
        id: column

        spacing: 0
    }
}
