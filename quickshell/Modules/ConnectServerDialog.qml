pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DCommon.Widgets
import qs.DCommon.FileBrowser

OverlayDialog {
    id: root

    property string errorText: ""
    readonly property var logins: [
        {
            "label": I18n.tr("Yourself", "connect dialog: log in as the user in the address"),
            "kind": "",
            "asks": false
        },
        {
            "label": I18n.tr("root, with sudo", "connect dialog: become root on the server with sudo"),
            "kind": "sudo",
            "asks": false
        },
        {
            "label": I18n.tr("Another user, with sudo", "connect dialog: become another user on the server with sudo"),
            "kind": "sudo",
            "asks": true
        },
        {
            "label": I18n.tr("Another user, with machinectl", "connect dialog: become another user on the server with machinectl"),
            "kind": "machinectl",
            "asks": true
        }
    ]
    readonly property var login: logins.find(item => item.label === loginDropdown.currentValue) ?? logins[0]
    readonly property bool canElevate: /^(sftp|ssh):\/\//.test(addressField.text.trim())
    readonly property bool elevating: canElevate && login.kind !== ""
    property string server: ""
    property var shares: []

    signal connected(string path)

    function ask() {
        reset();
        addressField.text = schemeDropdown.currentValue === "" ? "" : schemeDropdown.currentValue + "://";
        loginDropdown.currentValue = logins[0].label;
        userField.text = "";
        sudoPasswordField.text = "";
        open();
        addressField.forceActiveFocus();
    }

    function reset() {
        errorText = "";
        server = "";
        shares = [];
    }

    function submit(uri) {
        const target = (uri || addressField.text).trim();
        if (target === "")
            return;
        errorText = "";
        NetworkService.connectTo(target, result => root.present(target, result), "", elevating ? {
            "elevate": login.kind,
            "elevateUser": login.asks ? userField.text.trim() : "",
            "elevatePassword": sudoPasswordField.text
        } : null);
    }

    function present(target, result) {
        if (result.shares) {
            showShares(result);
            return;
        }
        const path = localPath(result);
        if (path === "") {
            showFailure(target, result);
            return;
        }
        sudoPasswordField.text = "";
        if (result.notice)
            ToastService.info(I18n.tr("This server has no separate sftp-server, so its files open as you. The terminal still becomes the other user.", "toast after connecting as another user to a server that can only share files as the login user"));
        connected(path);
        close();
    }

    function useRecent(server) {
        if (!server.elevate) {
            submit(server.uri);
            return;
        }
        const [kind, user] = server.elevate.split("+");
        addressField.text = server.uri;
        loginDropdown.currentValue = (logins.find(item => item.kind === kind && item.asks === !!user) ?? logins[0]).label;
        userField.text = user ?? "";
        sudoPasswordField.forceActiveFocus();
    }

    function localPath(result) {
        if (result.error)
            return "";
        return result.location || result.mount?.path || "";
    }

    function showShares(result) {
        errorText = "";
        server = result.server;
        shares = result.shares;
        addressField.text = result.server;
        if (!visible)
            open();
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

        FitDropdown {
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

        FitDropdown {
            id: loginDropdown

            width: parent.width
            visible: root.canElevate
            text: I18n.tr("Log in as", "label for choosing which user to become on the server")
            description: I18n.tr("Become another user after logging in", "connect dialog hint for the log in as picker")
            options: root.logins.map(item => item.label)
            currentValue: root.logins[0].label
        }

        DTextField {
            id: userField

            width: parent.width
            visible: root.elevating && root.login.asks
            outlined: true
            labelText: I18n.tr("User", "label for the user to become on the server")
            placeholderText: "nix"
            onAccepted: root.submit("")
        }

        DTextField {
            id: sudoPasswordField

            width: parent.width
            visible: root.elevating
            outlined: true
            echoMode: TextInput.Password
            showPasswordToggle: true
            labelText: I18n.tr("sudo password", "label for the sudo password input on the connect dialog")
            supportingText: I18n.tr("Leave empty if sudo doesn't ask. It is only used for this connection and isn't saved.", "connect dialog hint under the sudo password input")
            onAccepted: root.submit("")
        }

        StyledText {
            visible: root.server !== ""
            text: I18n.tr("Shares on %1", "header above the shares of a browsed server, %1 is the server").arg(root.server)
            color: Theme.primary
            font.pixelSize: Theme.fontSizeSmall
            font.weight: Theme.fontWeightMedium
        }

        StyledText {
            visible: root.server !== "" && root.shares.length === 0
            text: I18n.tr("This server offers no shares", "connect dialog note when a browsed server lists nothing")
            color: Theme.surfaceVariantText
            font.pixelSize: Theme.fontSizeSmall
        }

        Repeater {
            model: root.shares

            FilePlaceRow {
                required property var modelData

                width: parent.width
                label: modelData.name
                subtitle: modelData.comment ?? ""
                iconName: "folder-remote"
                busy: NetworkService.connecting
                onClicked: root.submit(modelData.uri)
            }
        }

        StyledText {
            visible: root.server === "" && NetworkService.servers.length > 0
            text: I18n.tr("Recent", "header above previously used servers")
            color: Theme.primary
            font.pixelSize: Theme.fontSizeSmall
            font.weight: Theme.fontWeightMedium
        }

        Repeater {
            model: root.server === "" ? NetworkService.servers : []

            FilePlaceRow {
                required property var modelData

                width: parent.width
                label: modelData.elevate ? modelData.uri + " (" + modelData.elevate.replace("+", " ") + ")" : modelData.uri
                iconName: "folder-remote"
                actionIcon: "close"
                busy: NetworkService.connecting
                onClicked: root.useRecent(modelData)
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
