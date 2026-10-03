pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

OverlayDialog {
    id: overlay

    readonly property bool vimKeys: SettingsData.vimKeys

    title: I18n.tr("Keyboard shortcuts", "overlay listing every keyboard shortcut")
    maximumWidth: Theme.dialogMaxWidth

    Shortcut {
        enabled: overlay.visible
        sequences: ["F1", "?"]
        onActivated: overlay.close()
    }

    DankFlickable {
        width: parent.width
        height: Math.min(contentHeight, overlay.height / 2)
        contentHeight: groups.implicitHeight
        clip: true

        Column {
            id: groups

            width: parent.width
            spacing: Theme.spacingL

            Repeater {
                model: Shortcuts.groups

                Column {
                    id: group

                    required property var modelData

                    width: groups.width
                    spacing: Theme.spacingXS

                    StyledText {
                        text: group.modelData.label
                        color: Theme.primary
                        font.pixelSize: Theme.fontSizeSmall
                        font.weight: Theme.fontWeightMedium
                    }

                    Repeater {
                        model: Shortcuts.inGroup(group.modelData.id, overlay.vimKeys)

                        Item {
                            id: entry

                            required property var modelData

                            width: group.width
                            height: Math.max(keys.implicitHeight, label.implicitHeight) + Theme.spacingXXS

                            StyledText {
                                id: label

                                anchors.left: parent.left
                                anchors.right: keys.left
                                anchors.rightMargin: Theme.spacingM
                                anchors.verticalCenter: parent.verticalCenter
                                text: entry.modelData.label
                                color: Theme.surfaceText
                                font.pixelSize: Theme.fontSizeSmall
                                elide: Text.ElideRight
                            }

                            Row {
                                id: keys

                                anchors.right: parent.right
                                anchors.verticalCenter: parent.verticalCenter
                                spacing: Theme.spacingXXS

                                Repeater {
                                    model: Shortcuts.keysFor(entry.modelData, overlay.vimKeys)

                                    DankKeycap {
                                        required property string modelData

                                        text: modelData
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    DankToggle {
        width: parent.width
        text: I18n.tr("Vim keys", "toggle label for vim style navigation")
        description: I18n.tr("h j k l move, / searches and letters run commands instead of jumping to a name.", "vim keys toggle description")
        checked: SettingsData.vimKeys
        onToggled: checked => SettingsData.vimKeys = checked
    }

    actions: [
        TextButton {
            text: I18n.tr("Close", "dialog button dismissing a dialog")
            onClicked: overlay.close()
        }
    ]
}
