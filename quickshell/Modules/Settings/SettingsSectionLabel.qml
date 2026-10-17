import QtQuick
import qs.Common
import qs.DCommon.Widgets

Item {
    id: root

    readonly property bool isSettingsSectionLabel: true

    property string text: ""

    Accessible.role: Accessible.StaticText
    Accessible.name: text

    width: parent?.width ?? 0
    height: label.implicitHeight + SettingsMetrics.sectionLabelTopGap + SettingsMetrics.sectionLabelBottomGap

    StyledText {
        id: label
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        anchors.verticalCenterOffset: (SettingsMetrics.sectionLabelTopGap - SettingsMetrics.sectionLabelBottomGap) / 2
        text: root.text
        font.pixelSize: Theme.fontSizeMedium
        font.weight: Theme.fontWeightMedium
        color: Theme.primary
        wrapMode: Text.WordWrap
        horizontalAlignment: Text.AlignLeft
    }
}
