pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DankCommon.Widgets
import qs.DankCommon.FileBrowser

OverlayDialog {
    id: root

    property string mime: ""
    property var paths: []
    property var apps: []
    property bool rememberChoice: false

    readonly property var current: appList.currentIndex >= 0 && appList.currentIndex < matches.length ? matches[appList.currentIndex] : null
    readonly property string selectedId: current ? current.id : ""

    readonly property var matches: {
        const needle = searchField.text.trim().toLowerCase();
        if (needle === "")
            return apps;
        return apps.filter(app => app.name.toLowerCase().includes(needle) || (app.genericName ?? "").toLowerCase().includes(needle) || app.id.toLowerCase().includes(needle));
    }

    function ask(mimeType, targets) {
        mime = mimeType;
        paths = targets;
        apps = [];
        rememberChoice = false;
        searchField.text = "";
        AppsService.allApps(response => {
            if (response.error) {
                ToastService.info(FileFormat.operationError(response.code, response.error));
                return;
            }
            apps = response.apps || [];
        });
        open();
        searchField.forceActiveFocus();
    }

    function submit() {
        if (selectedId === "")
            return;
        if (rememberChoice && mime !== "")
            AppsService.setDefault(mime, selectedId, null);
        AppsService.openWith(paths, selectedId, response => {
            if (response.error)
                ToastService.info(FileFormat.operationError(response.code, response.error));
        });
        close();
    }

    title: I18n.tr("Open with", "open with dialog header")
    acceptEnabled: selectedId !== ""
    onAccepted: submit()

    DankSearchField {
        id: searchField

        width: parent.width
        placeholderText: I18n.tr("Search applications", "open with dialog search placeholder")
        onAccepted: root.submit()
        Keys.onDownPressed: appList.forceActiveFocus()
    }

    DankListView {
        id: appList

        width: parent.width
        height: Theme.listItemHeight * 6
        clip: true
        spacing: Theme.groupedListGap
        model: root.matches
        focus: true
        currentIndex: 0

        delegate: AppRow {
            onClicked: appList.currentIndex = index
            onDoubleClicked: root.submit()
        }
    }

    DankToggle {
        width: parent.width
        visible: root.mime !== "" && SandboxService.available("setDefault")
        text: I18n.tr("Always use this application", "open with dialog option that sets the default")
        checked: root.rememberChoice
        onToggled: checked => root.rememberChoice = checked
    }

    actions: [
        TextButton {
            text: I18n.tr("Cancel", "dialog button to cancel")
            onClicked: root.close()
        },
        DankButton {
            text: I18n.tr("Open", "context menu entry opening a file")
            enabled: root.selectedId !== ""
            backgroundColor: Theme.primary
            textColor: Theme.primaryText
            onClicked: root.submit()
        }
    ]
}
