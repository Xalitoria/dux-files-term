pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Controls
import qs.Common
import qs.Services
import qs.DankCommon.Widgets

Popup {
    id: root

    width: Theme.dialogMaxWidth * 0.6
    padding: Theme.spacingL
    modal: false
    focus: false
    closePolicy: Popup.CloseOnEscape | Popup.CloseOnPressOutsideParent

    background: Rectangle {
        radius: Theme.cornerRadiusL
        color: Theme.surfaceContainer
        border.width: Theme.outlineWidth
        border.color: Theme.outlineVariant
    }

    contentItem: Loader {
        active: root.visible
        sourceComponent: Column {
            spacing: Theme.spacingM

            StyledText {
                text: I18n.tr("File operations", "jobs popover header")
                color: Theme.primary
                font.pixelSize: Theme.fontSizeSmall
            }

            Repeater {
                model: FileOpsService.jobs

                JobRow {
                    required property var modelData
                    width: root.availableWidth
                    job: modelData
                }
            }

            StyledText {
                visible: FileOpsService.jobs.length === 0
                text: I18n.tr("Nothing is running", "jobs popover empty state")
                color: Theme.surfaceVariantText
                font.pixelSize: Theme.fontSizeSmall
            }
        }
    }
}
