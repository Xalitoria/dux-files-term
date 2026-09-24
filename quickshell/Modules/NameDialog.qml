import QtQuick
import qs.Common
import qs.DankCommon.Widgets

OverlayDialog {
    id: root

    property string acceptText: I18n.tr("Save", "dialog button to confirm a name")

    signal submitted(string name)

    function ask(name) {
        nameField.text = name || "";
        open();
        nameField.forceActiveFocus();
    }

    function submit() {
        const trimmed = nameField.text.trim();
        if (trimmed === "")
            return;
        submitted(trimmed);
        close();
    }

    onAccepted: submit()

    DankTextField {
        id: nameField

        width: parent.width
        outlined: true
        labelText: I18n.tr("Name", "label for a file name input")
        onAccepted: root.submit()
        Keys.onReturnPressed: event => event.accepted = true
        Keys.onEnterPressed: event => event.accepted = true
    }

    actions: [
        TextButton {
            text: I18n.tr("Cancel", "dialog button to cancel")
            onClicked: root.close()
        },
        DankButton {
            text: root.acceptText
            backgroundColor: Theme.primary
            textColor: Theme.primaryText
            onClicked: root.submit()
        }
    ]
}
