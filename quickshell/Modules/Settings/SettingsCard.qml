import QtQuick
import qs.Common

Column {
    id: root

    readonly property bool isSettingsCard: true

    property string title: ""
    default property alias content: contentGroup.data

    width: parent?.width ?? 0
    spacing: 0

    SettingsSectionLabel {
        width: parent.width
        text: root.title
        visible: root.title !== ""
    }

    SettingsGroup {
        id: contentGroup
        width: parent.width
    }
}
