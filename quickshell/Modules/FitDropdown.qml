import QtQuick
import qs.Common
import qs.DCommon.Widgets

DDropdown {
    id: root

    property real widest: 0
    readonly property real iconRoom: optionIcons.length > 0 || Object.keys(optionIconMap).length > 0 ? Theme.iconSizeMedium + Theme.spacingS : 0
    readonly property int fittedWidth: Math.ceil(widest + iconRoom + Theme.spacingM * 4 + Theme.iconSize)

    function measure() {
        let longest = 0;
        for (let i = 0; i < probes.count; i++)
            longest = Math.max(longest, probes.itemAt(i)?.implicitWidth ?? 0);
        widest = longest;
    }

    popupWidth: Math.max(dropdownWidth, fittedWidth)
    onOptionsChanged: Qt.callLater(measure)

    Repeater {
        id: probes

        model: root.options

        StyledText {
            required property string modelData

            visible: false
            text: modelData
            font.pixelSize: Theme.fontSizeMedium
            onImplicitWidthChanged: Qt.callLater(root.measure)
        }
    }
}
