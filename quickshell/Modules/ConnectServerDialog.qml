pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DankCommon.Widgets
import qs.DankCommon.FileBrowser

OverlayDialog {
    id: root

    readonly property var schemes: ["sftp", "smb", "ftp", "dav", "davs", "nfs"]
    property string errorText: ""

    signal connected(string path)

    function ask() {
        errorText = "";
        addressField.text = schemeDropdown.currentValue + "://";
        open();
        addressField.forceActiveFocus();
    }

    function submit(uri) {
        const target = (uri || addressField.text).trim();
        if (target === "")
            return;
        errorText = "";
        NetworkService.connectTo(target, result => {
            if (result.error) {
                root.errorText = result.error;
                return;
            }
            root.connected(result.mount.path || "");
            root.close();
        });
    }

    title: I18n.tr("Connect to Server", "connect to server dialog header")
    onAccepted: submit("")

    Column {
        width: parent.width
        spacing: Theme.spacingM

        DankDropdown {
            id: schemeDropdown

            width: parent.width
            text: I18n.tr("Protocol", "label for the network protocol picker")
            options: root.schemes
            currentValue: root.schemes[0]
            onValueChanged: value => addressField.text = value + "://"
        }

        DankTextField {
            id: addressField

            width: parent.width
            outlined: true
            isError: root.errorText !== ""
            supportingText: root.errorText
            labelText: I18n.tr("Address", "label for a server address input")
            placeholderText: "sftp://host/path"
            onAccepted: root.submit("")
            Keys.onReturnPressed: event => event.accepted = true
            Keys.onEnterPressed: event => event.accepted = true
        }

        StyledText {
            visible: NetworkService.servers.length > 0
            text: I18n.tr("Recent", "header above previously used servers")
            color: Theme.primary
            font.pixelSize: Theme.fontSizeSmall
            font.weight: Theme.fontWeightMedium
        }

        Repeater {
            model: NetworkService.servers

            FilePlaceRow {
                required property var modelData

                width: parent.width
                label: modelData.uri
                iconName: "folder-remote"
                actionIcon: "close"
                busy: NetworkService.connecting
                onClicked: root.submit(modelData.uri)
                onActionTriggered: NetworkService.forget(modelData.uri)
            }
        }
    }

    actions: [
        TextButton {
            text: I18n.tr("Cancel", "dialog button to cancel")
            onClicked: root.close()
        },
        DankButton {
            text: I18n.tr("Connect", "connect to server dialog action")
            backgroundColor: Theme.primary
            textColor: Theme.primaryText
            enabled: !NetworkService.connecting
            onClicked: root.submit("")
        }
    ]
}
