import QtQuick
import qs.Common
import qs.DankCommon.Widgets

OverlayDialog {
    id: root

    property string message: ""
    property string subject: ""
    property string confirmText: I18n.tr("Delete", "confirm dialog default confirm button")
    property bool danger: true

    signal confirmed

    function confirm() {
        confirmed();
        close();
    }

    supportingText: message
    onAccepted: confirm()

    actions: [
        TextButton {
            text: I18n.tr("Cancel", "dialog button to cancel")
            onClicked: root.close()
        },
        DankButton {
            text: root.confirmText
            backgroundColor: root.danger ? Theme.errorContainer : Theme.primary
            textColor: root.danger ? Theme.onErrorContainer : Theme.primaryText
            onClicked: root.confirm()
        }
    ]
}
