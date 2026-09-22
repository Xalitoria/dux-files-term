pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DankCommon.Widgets

// The band above the home listing: every XDG user directory as a tile. The
// counts are one bounded files.count for at most eight paths, refreshed only
// when the directory list itself changes.
Item {
    id: band

    property var dirs: DirectoryService.userDirs.filter(dir => dir.key !== "home")
    property var counts: ({})

    signal opened(string path)

    implicitHeight: dirs.length === 0 ? 0 : title.height + Theme.spacingS + tiles.implicitHeight
    visible: dirs.length > 0

    onDirsChanged: refresh()
    Component.onCompleted: refresh()

    function refresh() {
        if (dirs.length === 0)
            return;
        DirectoryService.count(dirs.map(dir => dir.path), false, result => {
            if (result.error)
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
        anchors.left: parent.left
        text: I18n.tr("Quick access", "header above the shortcuts on the home folder")
        color: Theme.primary
        font.pixelSize: Theme.fontSizeSmall
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

            Rectangle {
                id: tile

                required property var modelData

                readonly property bool bookmarked: BookmarksService.has(modelData.path)

                width: FilesMetrics.quickAccessTileSize
                height: FilesMetrics.quickAccessTileSize
                radius: FilesMetrics.gridTileRadius
                color: "transparent"

                StateLayer {
                    id: hover

                    stateColor: Theme.surfaceText
                    cornerRadius: tile.radius
                    onClicked: band.opened(tile.modelData.path)
                }

                Column {
                    anchors.centerIn: parent
                    width: parent.width - FilesMetrics.gridTilePadding * 2
                    spacing: Theme.spacingXXS

                    FileIcon {
                        anchors.horizontalCenter: parent.horizontalCenter
                        iconName: tile.modelData.iconName
                        size: Theme.iconSizeLarge + Theme.spacingM
                        color: Theme.primary
                    }

                    StyledText {
                        width: parent.width
                        text: tile.modelData.name
                        color: Theme.surfaceText
                        font.pixelSize: Theme.fontSizeSmall
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
                    anchors.margins: Theme.spacingXXS
                    visible: tile.bookmarked || hover.containsMouse
                    buttonSize: Theme.iconSizeMedium + Theme.spacingXS
                    iconName: "favorite"
                    iconFilled: tile.bookmarked
                    iconSize: Theme.fontSizeMedium
                    iconColor: tile.bookmarked ? Theme.primary : Theme.surfaceVariantText
                    tooltipText: tile.bookmarked ? I18n.tr("Remove bookmark", "tooltip on the bookmark button") : I18n.tr("Bookmark this folder", "tooltip on the bookmark button")
                    onClicked: BookmarksService.toggle(tile.modelData.path)
                }
            }
        }
    }
}
