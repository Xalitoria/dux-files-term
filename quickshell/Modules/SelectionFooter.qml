pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

Rectangle {
    id: footer

    property int count: 0
    property real bytes: 0

    signal cleared

    readonly property string summary: {
        const items = count === 1 ? I18n.tr("1 selected", "a single selected item") : I18n.tr("%1 selected", "count of selected items").arg(count);
        if (bytes < 0)
            return items;
        return `${items} · ${FileFormat.size(bytes)}`;
    }

    implicitWidth: label.implicitWidth + clearButton.width + Theme.spacingM * 2 + Theme.spacingXS
    height: FilesMetrics.selectionFooterHeight
    radius: Theme.cornerRadiusFull
    color: Theme.secondaryContainer

    StyledText {
        id: label

        anchors.left: parent.left
        anchors.leftMargin: Theme.spacingM
        anchors.right: clearButton.left
        anchors.rightMargin: Theme.spacingXS
        anchors.verticalCenter: parent.verticalCenter
        text: footer.summary
        color: Theme.onSecondaryContainer
        font.pixelSize: Theme.fontSizeSmall
        elide: Text.ElideRight
    }

    DankActionButton {
        id: clearButton

        anchors.right: parent.right
        anchors.rightMargin: Theme.spacingXXS
        anchors.verticalCenter: parent.verticalCenter
        buttonSize: parent.height - Theme.spacingXXS * 2
        iconName: "close"
        iconSize: Theme.fontSizeMedium
        iconColor: Theme.onSecondaryContainer
        tooltipText: I18n.tr("Clear the selection", "keyboard shortcut")
        onClicked: footer.cleared()
    }
}
