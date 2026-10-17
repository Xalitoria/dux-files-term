pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DCommon.Widgets
import qs.DCommon.FileBrowser

OverlayDialog {
    id: root

    property string errorText: ""

    signal connected(string path)

    function ask() {
        errorText = "";
        addressField.text = schemeDropdown.currentValue === "" ? "" : schemeDropdown.currentValue + "://";
        open();
        addressField.forceActiveFocus();
    }

    function submit(uri) {
        const target = (uri || addressField.text).trim();
        if (target === "")
            return;
        errorText = "";
        NetworkService.connectTo(target, result => {
            const path = root.localPath(result);
            if (path === "") {
                root.showFailure(target, result);
                return;
            }
            root.connected(path);
            root.close();
        });
    }

    function localPath(result) {
        if (result.error)
            return "";
        return result.location || result.mount?.path || "";
    }

    function showFailure(target, result) {
        errorText = result.error || I18n.tr("Connected, but the share has no local folder. The gvfs FUSE service may not be running.", "connect dialog error when a mounted share has no fuse path");
        if (visible)
            return;
        addressField.text = target;
        open();
    }

    title: I18n.tr("Connect to Server", "connect to server dialog header")
    onAccepted: submit("")

    Column {
        width: parent.width
        spacing: Theme.spacingM

        DDropdown {
            id: schemeDropdown

            width: parent.width
            visible: NetworkService.schemes.length > 0
            text: I18n.tr("Protocol", "label for the network protocol picker")
            options: NetworkService.schemes
            currentValue: NetworkService.schemes[0] ?? ""
            onValueChanged: value => addressField.text = value + "://"
        }

        DTextField {
            id: addressField

            width: parent.width
            outlined: true
            isError: root.errorText !== ""
            supportingText: root.errorText
            labelText: I18n.tr("Address", "label for a server address input")
            placeholderText: "sftp://host/path"
            onAccepted: root.submit("")
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
        DButton {
            text: I18n.tr("Connect", "connect to server dialog action")
            backgroundColor: Theme.primary
            textColor: Theme.primaryText
            enabled: !NetworkService.connecting
            onClicked: root.submit("")
        }
    ]
}
