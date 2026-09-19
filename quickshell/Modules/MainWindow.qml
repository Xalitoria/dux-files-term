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
    readonly property bool dialogOpen: conflictDialog.visible || errorDialog.visible || confirmDialog.visible || renameDialog.visible || newFolderDialog.visible
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
            clip: true
            spacing: Theme.groupedListGap
            model: directory.entries
            cacheBuffer: Math.max(0, height)
            currentIndex: -1

            delegate: FileRow {
                onClicked: {
                    fileList.currentIndex = index;
                    if (isDir)
                        window.currentPath = path;
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
