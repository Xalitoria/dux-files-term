pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Layouts
import qs.Common
import qs.DankCommon.Widgets

DankListItem {
    id: row

    required property int index
    required property string name
    required property string path
    required property bool isDir
    required property bool isSymlink
    required property real size
    required property real mtimeMs

    width: listView?.width ?? 0
    firstInGroup: index === 0
    lastInGroup: index === (listView?.count ?? 0) - 1

    function formatSize(bytes) {
        const units = [I18n.tr("B", "file size unit"), I18n.tr("kB", "file size unit"), I18n.tr("MB", "file size unit"), I18n.tr("GB", "file size unit"), I18n.tr("TB", "file size unit")];
        let value = bytes;
        let unit = 0;
        while (value >= 1000 && unit < units.length - 1) {
            value /= 1000;
            unit++;
        }
        const digits = unit === 0 ? 0 : 1;
        return `${value.toFixed(digits)} ${units[unit]}`;
    }

    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: Theme.spacingL
        anchors.rightMargin: Theme.spacingL
        spacing: Theme.spacingM

        DankIcon {
            name: row.isDir ? "folder" : row.isSymlink ? "link" : "draft"
            size: Theme.iconSize
            color: row.isDir ? Theme.primary : row.contentColor
        }

        StyledText {
            Layout.fillWidth: true
            text: row.name
            color: row.contentColor
            font.pixelSize: Theme.fontSizeMedium
            elide: Text.ElideMiddle
        }

        StyledText {
            visible: !row.isDir
            text: row.formatSize(row.size)
            color: row.supportingContentColor
            font.pixelSize: Theme.fontSizeSmall
        }

        StyledText {
            Layout.preferredWidth: Theme.spacingXL * 6
            horizontalAlignment: Text.AlignRight
            text: Qt.locale().toString(new Date(row.mtimeMs), Locale.ShortFormat)
            color: row.supportingContentColor
            font.pixelSize: Theme.fontSizeSmall
        }
    }
}
