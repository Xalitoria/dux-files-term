pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DankCommon.Widgets
import qs.DankCommon.FileBrowser

Item {
    id: band

    property var dirs: DirectoryService.userDirs.filter(dir => dir.key !== "home")
    property var counts: ({})

    signal opened(string path)

    implicitHeight: dirs.length === 0 ? 0 : Theme.spacingM + title.height + Theme.spacingS + tiles.implicitHeight + Theme.spacingM
    visible: dirs.length > 0

    onDirsChanged: refresh()
    Component.onCompleted: refresh()

    function refresh() {
        if (dirs.length === 0)
            return;
        DirectoryService.count(dirs.map(dir => dir.path), false, result => {
            if (result.error || !band)
                return;
            band.counts = result.counts || ({});
        });
    }

    function countLabel(path) {
        const count = counts[path];
        if (!count || count.code)
            return "";
        return count.capped ? `${FileFormat.count(count.count)}+` : FileFormat.count(count.count);
    }

    StyledText {
        id: title

        anchors.top: parent.top
        anchors.topMargin: Theme.spacingM
        anchors.left: parent.left
        text: I18n.tr("Quick access", "header above the shortcuts on the home folder")
        color: Theme.primary
        font.pixelSize: Theme.fontSizeMedium
        font.weight: Theme.fontWeightMedium
    }

    Flow {
        id: tiles

        anchors.top: title.bottom
        anchors.topMargin: Theme.spacingS
        anchors.left: parent.left
        anchors.right: parent.right
        spacing: FilesMetrics.gridGap

        Repeater {
            model: band.dirs

            StyledButton {
                id: tile

                required property var modelData

                readonly property bool bookmarked: BookmarksService.has(modelData.path)

                width: FilesMetrics.quickAccessTileSize
                height: FilesMetrics.quickAccessTileSize
                focusPolicy: Qt.NoFocus
                radius: FilesMetrics.gridTileRadius
                color: "transparent"
                Accessible.name: modelData.name
                onClicked: band.opened(modelData.path)

                StateLayer {
                    control: tile
                    stateColor: Theme.surfaceText
                    cornerRadius: tile.radius
                }

                Column {
                    anchors.centerIn: parent
                    width: parent.width - FilesMetrics.gridTilePadding * 2
                    spacing: Theme.spacingXXS

                    FileIcon {
                        anchors.horizontalCenter: parent.horizontalCenter
                        iconName: tile.modelData.iconName
                        size: FilesMetrics.quickAccessIconSize
                        color: Theme.primary
                    }

                    StyledText {
                        width: parent.width
                        text: tile.modelData.name
                        color: Theme.surfaceText
                        font.pixelSize: Theme.fontSizeMedium
                        horizontalAlignment: Text.AlignHCenter
                        elide: Text.ElideRight
                    }

                    StyledText {
                        width: parent.width
                        visible: text !== ""
                        text: band.countLabel(tile.modelData.path)
                        color: Theme.surfaceVariantText
                        font.pixelSize: Theme.fontSizeSmall
                        horizontalAlignment: Text.AlignHCenter
                        elide: Text.ElideRight
                    }
                }

                DankActionButton {
                    anchors.top: parent.top
                    anchors.right: parent.right
                    anchors.margins: Theme.spacingXS
                    visible: tile.bookmarked || tile.hovered || hovered
                    buttonSize: Theme.buttonHeightXS
                    iconName: "favorite"
                    iconFilled: tile.bookmarked
                    iconSize: Theme.iconSizeSmall
                    iconColor: tile.bookmarked ? Theme.primary : Theme.surfaceVariantText
                    tooltipText: tile.bookmarked ? I18n.tr("Remove bookmark", "tooltip on the bookmark button") : I18n.tr("Bookmark this folder", "tooltip on the bookmark button")
                    onClicked: BookmarksService.toggle(tile.modelData.path)
                }
            }
        }
    }
}
