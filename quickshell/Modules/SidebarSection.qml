pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

Column {
    id: section

    property string title: ""
    property string emptyText: ""
    property bool showEmpty: false

    default property alias rows: body.data

    spacing: Theme.spacingXXS

    StyledText {
        text: section.title
        color: Theme.primary
        font.pixelSize: Theme.fontSizeSmall
        font.weight: Font.DemiBold
        leftPadding: Theme.spacingM
        bottomPadding: Theme.spacingXXS
    }

    Column {
        id: body

        width: section.width
        spacing: Theme.spacingXXS
    }

    StyledText {
        width: section.width
        visible: section.showEmpty && section.emptyText !== ""
        text: section.emptyText
        color: Theme.surfaceVariantText
        font.pixelSize: Theme.fontSizeSmall
        leftPadding: Theme.spacingM
        rightPadding: Theme.spacingM
        wrapMode: Text.Wrap
    }
}
