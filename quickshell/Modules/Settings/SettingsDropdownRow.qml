import QtQuick
import qs.Common
import qs.Modules

SettingsRow {
    id: root

    property string text: ""
    property string description: ""
    property string currentValue: ""
    property alias options: dropdown.options
    property alias enableFuzzySearch: dropdown.enableFuzzySearch
    property alias maxPopupHeight: dropdown.maxPopupHeight
    property alias dropdownWidth: dropdown.dropdownWidth

    signal valueChanged(string value)

    title: text
    subtitle: description
    onCurrentValueChanged: dropdown.currentValue = currentValue

    FitDropdown {
        id: dropdown
        enabled: root.enabled
        Accessible.name: root.text
        Accessible.description: root.description + (root.description ? " · " : "") + currentValue
        alignPopupRight: true
        width: Math.min(Math.max(dropdownWidth, fittedWidth), root.width - SettingsMetrics.rowPaddingH * 2)
        Component.onCompleted: currentValue = root.currentValue
        onValueChanged: value => root.valueChanged(value)
    }
}
