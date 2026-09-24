import QtQuick
import QtQuick.Layouts
import qs.Common
import qs.Services
import qs.DankCommon.Widgets

OverlayDialog {
    id: root

    readonly property var paused: FileOpsService.jobs.filter(job => job.state === "paused-conflict")
    readonly property var current: paused.length > 0 ? paused[0] : null
    readonly property var conflict: current ? current.conflict : null
    property bool applyToAll: false

    function answer(action) {
        if (!current || !conflict)
            return;
        FileOpsService.respond(current.id, conflict.id, action, applyToAll);
        applyToAll = false;
    }

    function offers(action) {
        return conflict ? (conflict.actions || []).includes(action) : false;
    }

    onConflictChanged: conflict ? open() : close()
    acceptEnabled: offers("keepBoth")
    title: I18n.tr("This item already exists", "file conflict dialog header")
    supportingText: conflict ? I18n.tr("%1 already exists in this folder.", "file conflict dialog explanation").arg(conflict.target.name) : ""
    closeEnabled: false
    onAccepted: {
        if (offers("keepBoth"))
            answer("keepBoth");
    }

    ColumnLayout {
        width: parent.width
        spacing: Theme.spacingM

        ConflictSide {
            Layout.fillWidth: true
            label: I18n.tr("Already here", "file conflict dialog label for the existing item")
            entry: root.conflict ? root.conflict.target : null
        }

        ConflictSide {
            Layout.fillWidth: true
            label: I18n.tr("Being added", "file conflict dialog label for the incoming item")
            entry: root.conflict ? root.conflict.source : null
        }

        DankToggle {
            Layout.fillWidth: true
            text: I18n.tr("Do this for the rest", "file conflict dialog option to reuse the answer")
            checked: root.applyToAll
            onToggled: checked => root.applyToAll = checked
        }
    }

    actions: [
        TextButton {
            text: I18n.tr("Cancel", "dialog button to cancel")
            onClicked: root.answer("cancel")
        },
        DankButton {
            visible: root.offers("merge")
            text: I18n.tr("Merge", "file conflict dialog button to merge folders")
            backgroundColor: Theme.secondaryContainer
            textColor: Theme.onSecondaryContainer
            onClicked: root.answer("merge")
        },
        DankButton {
            visible: root.offers("skip")
            text: I18n.tr("Skip", "file conflict dialog button to leave the item alone")
            backgroundColor: Theme.secondaryContainer
            textColor: Theme.onSecondaryContainer
            onClicked: root.answer("skip")
        },
        DankButton {
            visible: root.offers("replace")
            text: I18n.tr("Replace", "file conflict dialog button to overwrite the existing item")
            backgroundColor: Theme.errorContainer
            textColor: Theme.onErrorContainer
            onClicked: root.answer("replace")
        },
        DankButton {
            visible: root.offers("keepBoth")
            text: I18n.tr("Keep both", "file conflict dialog button to keep both items")
            backgroundColor: Theme.primary
            textColor: Theme.primaryText
            onClicked: root.answer("keepBoth")
        }
    ]
}
