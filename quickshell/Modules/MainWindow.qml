pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Window
import Quickshell
import qs.Common
import qs.Models
import qs.Services
import qs.DankCommon.Widgets

FloatingWindow {
    id: window

    signal hideRequested

    property string currentPath: Paths.strip(Paths.home)
    readonly property string opScope: "main"
    property string pendingTrust: ""
    readonly property bool dialogOpen: conflictDialog.visible || errorDialog.visible || confirmDialog.visible || renameDialog.visible || newFolderDialog.visible || openWithDialog.visible || executableDialog.visible || trustDialog.visible
    readonly property var currentEntry: fileList.currentIndex >= 0 && fileList.currentIndex < directory.count ? directory.entries.get(fileList.currentIndex) : null

    title: I18n.tr("Dank Files", "main window title")
    implicitWidth: 960
    implicitHeight: 640
    minimumSize: Qt.size(480, 320)
    color: Theme.surface
    visible: true

    onClosed: hideRequested()

    function errorText(code) {
        switch (code) {
        case "ENOENT":
            return I18n.tr("This folder no longer exists", "directory listing error");
        case "EACCES":
            return I18n.tr("You do not have permission to view this folder", "directory listing error");
        default:
            return I18n.tr("This folder could not be opened", "directory listing error");
        }
    }

    function navigateUp() {
        const parent = currentPath.substring(0, currentPath.lastIndexOf("/"));
        currentPath = parent === "" ? "/" : parent;
    }

    function selection() {
        return currentEntry ? [currentEntry.path] : [];
    }

    function reportError(response) {
        if (response.error)
            ToastService.info(FileFormat.operationError(response.code, response.error));
    }

    function activate(entry) {
        if (!entry)
            return;
        if (entry.isDir) {
            currentPath = entry.path;
            return;
        }
        if (entry.untrusted) {
            trustDialog.subject = entry.name;
            trustDialog.pending = entry.path;
            trustDialog.open();
            return;
        }
        if (FileFormat.isRunnable(entry.mime, entry.isExecutable)) {
            executableDialog.ask(entry.path, entry.name);
            return;
        }
        AppsService.open([entry.path], response => window.reportError(response));
    }

    function openWithCurrent() {
        if (!currentEntry)
            return;
        openWithDialog.ask(currentEntry.mime, [currentEntry.path]);
    }

    function openContainingFolder() {
        if (!currentEntry)
            return;
        const parent = currentEntry.path.substring(0, currentEntry.path.lastIndexOf("/"));
        currentPath = parent === "" ? "/" : parent;
    }

    function openTerminalHere() {
        AppsService.openTerminal(currentPath, response => window.reportError(response));
    }

    function showMenuFor(index, pointX, pointY) {
        fileList.currentIndex = index;
        openWithMenu.close();
        fileMenu.openAt(pointX, pointY);
    }

    function loadOpenWithMenu() {
        openWithMenu.apps = [];
        if (!currentEntry)
            return;
        AppsService.appsForMime(currentEntry.mime, currentEntry.path, response => {
            if (response.error)
                return;
            openWithMenu.apps = (response.apps || []).slice(0, 5);
        });
    }

    function trashSelection() {
        const paths = selection();
        if (paths.length === 0)
            return;
        FileOpsService.trash(paths, opScope);
    }

    function deleteSelection() {
        const paths = selection();
        if (paths.length === 0)
            return;
        confirmDialog.pending = paths;
        confirmDialog.subject = currentEntry.name;
        confirmDialog.open();
    }

    function renameSelection() {
        if (!currentEntry)
            return;
        renameDialog.target = currentEntry.path;
        renameDialog.ask(currentEntry.name);
    }

    function toastFor(job) {
        switch (job.kind) {
        case "trash":
            return I18n.tr("Moved to trash", "toast after trashing files");
        case "move":
            return I18n.tr("Moved", "toast after moving files");
        case "copy":
            return I18n.tr("Copied", "toast after copying files");
        case "rename":
            return I18n.tr("Renamed", "toast after renaming a file");
        default:
            return "";
        }
    }

    DirectoryModel {
        id: directory

        path: window.currentPath
        onGone: window.navigateUp()
    }

    Connections {
        target: FileOpsService

        function onRequestFailed(message) {
            ToastService.info(message);
        }

        function onJobFinished(job) {
            if (job.kind === "setPermissions" && window.pendingTrust !== "") {
                const target = window.pendingTrust;
                window.pendingTrust = "";
                if (job.state !== "failed")
                    AppsService.run(target, response => window.reportError(response));
            }
            if (job.state === "failed") {
                ToastService.info(FileFormat.operationError(job.code, job.error));
                return;
            }
            if (job.cancelled || job.scope !== window.opScope || !job.undoable)
                return;
            const message = window.toastFor(job);
            if (message === "")
                return;
            ToastService.show(message, {
                "actionLabel": I18n.tr("Undo", "toast action reversing the last operation"),
                "action": () => FileOpsService.undo(window.opScope)
            });
        }
    }

    Item {
        id: surface

        anchors.fill: parent
        readonly property bool isFloatingWindowSurface: true
        readonly property bool windowActive: Window.active

        onWindowActiveChanged: {
            if (windowActive)
                directory.refresh();
        }

        Shortcut {
            enabled: !window.dialogOpen
            sequences: ["Return", "Enter"]
            onActivated: window.activate(window.currentEntry)
        }

        Shortcut {
            enabled: !window.dialogOpen
            sequences: ["Ctrl+Return", "Ctrl+Enter"]
            onActivated: window.openContainingFolder()
        }

        Shortcut {
            enabled: !window.dialogOpen && window.currentEntry !== null
            sequences: ["Shift+F10", "Menu"]
            onActivated: {
                const row = fileList.currentItem;
                const point = row ? row.mapToItem(surface, Theme.spacingL, row.height) : Qt.point(Theme.spacingL, Theme.spacingL);
                window.showMenuFor(fileList.currentIndex, point.x, point.y);
            }
        }

        Shortcut {
            enabled: !window.dialogOpen
            sequences: ["Ctrl+Shift+O"]
            onActivated: window.openWithCurrent()
        }

        Shortcut {
            enabled: !window.dialogOpen
            sequences: ["Ctrl+Alt+T"]
            onActivated: window.openTerminalHere()
        }

        Shortcut {
            enabled: !window.dialogOpen
            sequences: ["Ctrl+H"]
            onActivated: directory.setShowHidden(!directory.includeHidden)
        }

        Shortcut {
            enabled: !window.dialogOpen
            sequences: ["Ctrl+C"]
            onActivated: FileOpsService.copyToClipboard(window.selection())
        }

        Shortcut {
            enabled: !window.dialogOpen
            sequences: ["Ctrl+X"]
            onActivated: FileOpsService.cut(window.selection())
        }

        Shortcut {
            enabled: !window.dialogOpen
            sequences: ["Ctrl+V"]
            onActivated: FileOpsService.paste(window.currentPath, window.opScope)
        }

        Shortcut {
            enabled: !window.dialogOpen
            sequences: ["F2"]
            onActivated: window.renameSelection()
        }

        Shortcut {
            enabled: !window.dialogOpen
            sequences: ["Delete"]
            onActivated: window.trashSelection()
        }

        Shortcut {
            enabled: !window.dialogOpen
            sequences: ["Shift+Delete"]
            onActivated: window.deleteSelection()
        }

        Shortcut {
            enabled: !window.dialogOpen
            sequences: ["Ctrl+Z"]
            onActivated: FileOpsService.undo(window.opScope)
        }

        Shortcut {
            enabled: !window.dialogOpen
            sequences: ["Ctrl+Shift+N"]
            onActivated: newFolderDialog.ask(I18n.tr("Untitled folder", "default name for a new folder"))
        }

        StyledText {
            id: pathLabel

            anchors.left: parent.left
            anchors.right: jobsButton.left
            anchors.leftMargin: Theme.spacingL
            anchors.rightMargin: Theme.spacingL
            anchors.verticalCenter: jobsButton.verticalCenter
            text: Paths.shortenHome(window.currentPath)
            color: Theme.surfaceText
            font.pixelSize: Theme.fontSizeLarge
            elide: Text.ElideMiddle
        }

        DankIconButton {
            id: jobsButton

            anchors.top: parent.top
            anchors.right: parent.right
            anchors.margins: Theme.spacingL
            visible: FileOpsService.jobs.length > 0
            iconName: FileOpsService.activeCount > 0 ? "sync" : "check_circle"
            size: "s"
            onClicked: jobsPopover.visible ? jobsPopover.close() : jobsPopover.open()

            JobsPopover {
                id: jobsPopover

                y: jobsButton.height + Theme.spacingS
                x: jobsButton.width - width
            }
        }

        DankListView {
            id: fileList

            anchors.top: jobsButton.bottom
            anchors.bottom: parent.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.margins: Theme.spacingL
            focus: true
            clip: true
            spacing: Theme.groupedListGap
            model: directory.entries
            cacheBuffer: Math.max(0, height)
            currentIndex: -1

            delegate: FileRow {
                onClicked: fileList.currentIndex = index

                onDoubleClicked: {
                    fileList.currentIndex = index;
                    window.activate(directory.entries.get(index));
                }

                onContextMenuRequested: (mouseX, mouseY) => {
                    const point = mapToItem(surface, mouseX, mouseY);
                    window.showMenuFor(index, point.x, point.y);
                }
            }

            onContentYChanged: scrollSettle.restart()
            onCountChanged: scrollSettle.restart()

            Timer {
                id: scrollSettle

                interval: Theme.mediumDuration
                onTriggered: {
                    const first = fileList.indexAt(fileList.width / 2, fileList.contentY);
                    const last = fileList.indexAt(fileList.width / 2, fileList.contentY + fileList.height - 1);
                    directory.requestThumbnails(Math.max(first, 0), last < 0 ? fileList.count - 1 : last);
                    if (last < 0 || last >= fileList.count - 1)
                        directory.loadMore();
                }
            }
        }

        StyledText {
            anchors.centerIn: parent
            visible: !directory.loading && directory.error === "" && directory.count === 0
            text: I18n.tr("This folder is empty", "empty directory placeholder")
            color: Theme.surfaceVariantText
            font.pixelSize: Theme.fontSizeLarge
        }

        StyledText {
            anchors.centerIn: parent
            visible: directory.error !== ""
            text: window.errorText(directory.errorCode)
            color: Theme.surfaceVariantText
            font.pixelSize: Theme.fontSizeLarge
        }
    }

    ContextMenu {
        id: fileMenu

        anchorItem: surface

        ContextMenuItem {
            text: I18n.tr("Open", "context menu entry opening a file")
            iconName: "open_in_new"
            onClicked: {
                fileMenu.close();
                window.activate(window.currentEntry);
            }
        }

        ContextMenuItem {
            id: openWithItem

            text: I18n.tr("Open with", "open with dialog header")
            iconName: "apps"
            submenu: true
            enabled: window.currentEntry !== null && !window.currentEntry.isDir
            onEntered: {
                if (!enabled)
                    return;
                window.loadOpenWithMenu();
                openWithMenu.openAt(fileMenu.x + fileMenu.width - Theme.spacingM, fileMenu.y + y);
            }
            onClicked: entered()
        }

        ContextMenuItem {
            text: I18n.tr("Open in terminal", "context menu entry opening a terminal")
            iconName: "terminal"
            onClicked: {
                fileMenu.close();
                window.openTerminalHere();
            }
        }
    }

    ContextMenu {
        id: openWithMenu

        property var apps: []

        anchorItem: surface

        Repeater {
            model: openWithMenu.apps

            ContextMenuItem {
                required property var modelData

                text: modelData.name
                iconName: modelData.isDefault ? "check" : ""
                onClicked: {
                    openWithMenu.close();
                    fileMenu.close();
                    AppsService.openWith(window.selection(), modelData.id, response => window.reportError(response));
                }
            }
        }

        ContextMenuItem {
            text: I18n.tr("Other application…", "open with menu entry opening the full application picker")
            iconName: "more_horiz"
            onClicked: {
                openWithMenu.close();
                fileMenu.close();
                window.openWithCurrent();
            }
        }
    }

    OpenWithDialog {
        id: openWithDialog
    }

    ExecutableDialog {
        id: executableDialog

        onRunRequested: AppsService.run(path, response => window.reportError(response))
        onEditRequested: openWithDialog.ask("text/plain", [path])
    }

    ConfirmDialog {
        id: trustDialog

        property string pending: ""

        title: I18n.tr("Run this desktop file?", "untrusted desktop file dialog header")
        message: I18n.tr("%1 has not been marked as trusted. Running it allows it to start any program.", "untrusted desktop file dialog explanation").arg(subject)
        confirmText: I18n.tr("Trust and run", "untrusted desktop file dialog action")
        danger: false
        onConfirmed: {
            window.pendingTrust = pending;
            FileOpsService.setPermissions([pending], "755", false);
            pending = "";
        }
    }

    ConflictDialog {
        id: conflictDialog
    }

    OperationErrorDialog {
        id: errorDialog
    }

    NameDialog {
        id: renameDialog

        property string target: ""

        title: I18n.tr("Rename", "rename dialog header")
        acceptText: I18n.tr("Rename", "rename dialog header")
        onSubmitted: name => FileOpsService.rename(target, name, window.opScope)
    }

    NameDialog {
        id: newFolderDialog

        title: I18n.tr("New folder", "new folder dialog header")
        acceptText: I18n.tr("Create", "dialog button creating an item")
        onSubmitted: name => FileOpsService.newFolder(window.currentPath, name, window.opScope)
    }

    ConfirmDialog {
        id: confirmDialog

        property var pending: []

        title: I18n.tr("Delete permanently?", "permanent delete confirmation header")
        message: I18n.tr("%1 will be deleted permanently. This cannot be undone.", "permanent delete confirmation explanation").arg(subject)
        onConfirmed: {
            FileOpsService.deletePermanently(pending);
            pending = [];
        }
    }

    Toast {
        anchors.fill: parent
    }
}
