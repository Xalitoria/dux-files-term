pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

Item {
    id: header

    property var columnIds: []
    property var widths: ({})
    property string sortKey: "name"
    property bool sortDescending: false
    property real nameIndent: 0

    signal sortRequested(string key)
    signal columnResized(string id, real width)
    signal columnsMenuRequested(real pointX, real pointY)

    function columnWidth(id) {
        return widths[id] ?? FileColumns.specFor(id)?.width ?? FilesMetrics.columnMinWidth;
    }

    implicitHeight: FilesMetrics.columnHeaderHeight
    clip: true

    // Opaque, or the list scrolling underneath shows through the labels.
    Rectangle {
        anchors.fill: parent
        color: Theme.surfaceContainerLow
    }

    Rectangle {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        height: Theme.dividerWidth
        color: Theme.outlineVariant
    }

    component HeaderCell: Item {
        id: cell

        property string columnId: ""
        property string sortsBy: ""
        property int align: Text.AlignLeft

        readonly property bool active: header.sortKey === sortsBy && sortsBy !== ""

        height: header.height

        StateLayer {
            enabled: cell.sortsBy !== ""
            stateColor: Theme.surfaceText
            cornerRadius: Theme.cornerRadiusS
            onClicked: header.sortRequested(cell.sortsBy)
        }

        Row {
            anchors.verticalCenter: parent.verticalCenter
            anchors.left: cell.align === Text.AlignRight ? undefined : parent.left
            anchors.right: cell.align === Text.AlignRight ? parent.right : undefined
            spacing: Theme.spacingXXS

            StyledText {
                anchors.verticalCenter: parent.verticalCenter
                width: Math.min(implicitWidth, Math.max(0, cell.width - (cell.active ? Theme.fontSizeSmall + Theme.spacingXXS : 0)))
                text: FileColumns.label(cell.columnId)
                color: cell.active ? Theme.primary : Theme.surfaceVariantText
                font.pixelSize: Theme.fontSizeSmall
                font.weight: Theme.fontWeightMedium
                elide: Text.ElideRight
            }

            DankIcon {
                anchors.verticalCenter: parent.verticalCenter
                visible: cell.active
                name: header.sortDescending ? "arrow_downward" : "arrow_upward"
                size: Theme.fontSizeSmall
                color: Theme.primary
            }
        }
    }

    HeaderCell {
        anchors.left: parent.left
        anchors.leftMargin: header.nameIndent
        anchors.right: optional.left
        anchors.rightMargin: FilesMetrics.columnGap
        columnId: "name"
        sortsBy: "name"
    }

    Row {
        id: optional

        anchors.right: parent.right
        anchors.rightMargin: Theme.spacingM
        spacing: FilesMetrics.columnGap

        Repeater {
            model: header.columnIds

            HeaderCell {
                required property string modelData

                width: header.columnWidth(modelData)
                columnId: modelData
                sortsBy: FileColumns.sortKeyFor(modelData)
                align: FileColumns.specFor(modelData)?.align ?? Text.AlignLeft

                MouseArea {
                    width: Theme.spacingS
                    height: parent.height
                    cursorShape: Qt.SizeHorCursor
                    acceptedButtons: Qt.LeftButton

                    property real pressX: 0
                    property real pressWidth: 0

                    onPressed: mouse => {
                        pressX = mapToItem(header, mouse.x, 0).x;
                        pressWidth = parent.width;
                    }
                    onPositionChanged: mouse => {
                        if (!pressed)
                            return;
                        const delta = pressX - mapToItem(header, mouse.x, 0).x;
                        header.columnResized(parent.modelData, Math.max(FilesMetrics.columnMinWidth, pressWidth + delta));
                    }
                }
            }
        }
    }

    MouseArea {
        anchors.fill: parent
        acceptedButtons: Qt.RightButton
        onClicked: mouse => header.columnsMenuRequested(mouse.x, mouse.y)
    }
}
