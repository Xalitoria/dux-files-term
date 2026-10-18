import QtQuick
import qs.Common
import qs.Services
import qs.DCommon.Widgets

SettingsRow {
    property string url: ""

    clickable: url !== ""
    onClicked: DuxFilesService.openUri(url)

    DIcon {
        anchors.verticalCenter: parent.verticalCenter
        name: "open_in_new"
        size: Theme.iconSize
        color: Theme.surfaceVariantText
    }
}
