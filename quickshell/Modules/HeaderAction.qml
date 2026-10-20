import QtQuick
import qs.Common
import qs.DCommon.Widgets

Item {
    id: headerAction

    property string label: ""
    property string iconName: ""
    property bool compact: false
    property color containerColor: Theme.secondaryContainer
    property color contentColor: Theme.onSecondaryContainer

    signal triggered

    implicitWidth: compact ? iconButton.width : textButton.implicitWidth
    implicitHeight: Theme.buttonHeightXS

    DButton {
        id: textButton

        visible: !headerAction.compact
        text: headerAction.label
        iconName: headerAction.iconName
        buttonHeight: Theme.buttonHeightXS
        backgroundColor: headerAction.containerColor
        textColor: headerAction.contentColor
        onClicked: headerAction.triggered()
    }

    DActionButton {
        id: iconButton

        visible: headerAction.compact
        buttonSize: Theme.buttonHeightXS
        iconName: headerAction.iconName
        iconSize: Theme.iconSizeMedium
        iconColor: headerAction.contentColor
        backgroundColor: headerAction.containerColor
        tooltipText: headerAction.label
        onClicked: headerAction.triggered()
    }
}
