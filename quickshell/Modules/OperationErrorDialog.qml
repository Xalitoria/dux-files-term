import QtQuick
import QtQuick.Layouts
import qs.Common
import qs.Services
import qs.DankCommon.Widgets

OverlayDialog {
    id: root

    readonly property var paused: FileOpsService.jobs.filter(job => job.state === "paused-error")
    readonly property var current: paused.length > 0 ? paused[0] : null
    readonly property var failure: current ? current.failure : null
    property bool applyToAll: false

    function answer(action) {
        if (!current || !failure)
            return;
        FileOpsService.respond(current.id, failure.id, action, applyToAll);
        applyToAll = false;
    }

    function offers(action) {
        return failure ? (failure.actions || []).includes(action) : false;
    }

    onFailureChanged: failure ? open() : close()
    title: failure ? FileFormat.operationError(failure.code, failure.message) : ""
    supportingText: failure ? failure.path : ""
    closeEnabled: false

    ColumnLayout {
        width: parent.width
        spacing: Theme.spacingM

        StyledText {
            Layout.fillWidth: true
            visible: root.failure !== null
            text: root.failure ? root.failure.message : ""
            color: Theme.surfaceVariantText
            font.pixelSize: Theme.fontSizeSmall
            wrapMode: Text.WordWrap
        }

        DankToggle {
            Layout.fillWidth: true
            text: I18n.tr("Do this for the rest", "file conflict dialog option to reuse the answer")
            checked: root.applyToAll
            onToggled: checked => root.applyToAll = checked
        }
    }

    actions: [
        DankButton {
            text: I18n.tr("Cancel", "dialog button to cancel")
            backgroundColor: "transparent"
            textColor: Theme.primary
            onClicked: root.answer("cancel")
        },
        DankButton {
            visible: root.offers("skip")
            text: I18n.tr("Skip", "file conflict dialog button to leave the item alone")
            backgroundColor: Theme.secondaryContainer
            textColor: Theme.onSecondaryContainer
            onClicked: root.answer("skip")
        },
        DankButton {
            visible: root.offers("retry")
            text: I18n.tr("Try again", "file operation error dialog button to retry")
            backgroundColor: Theme.primary
            textColor: Theme.primaryText
            onClicked: root.answer("retry")
        }
    ]
}
