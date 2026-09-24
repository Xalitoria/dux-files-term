import QtQuick
import QtQuick.Layouts
import qs.Common
import qs.Services
import qs.DankCommon.Widgets
import qs.DankCommon.FileBrowser

Item {
    id: row

    required property var job

    readonly property bool finished: job.state === "done" || job.state === "failed"
    readonly property real fraction: {
        const total = job.progress.bytesTotal > 0 ? job.progress.bytesTotal : job.progress.filesTotal;
        const done = job.progress.bytesTotal > 0 ? job.progress.bytesDone : job.progress.filesDone;
        if (total <= 0)
            return 0;
        return Math.min(1, done / total);
    }

    function label() {
        switch (job.kind) {
        case "copy":
            return I18n.tr("Copying", "file operation in progress");
        case "move":
            return I18n.tr("Moving", "file operation in progress");
        case "trash":
            return I18n.tr("Moving to trash", "file operation in progress");
        case "restore":
            return I18n.tr("Restoring", "file operation in progress");
        case "delete":
            return I18n.tr("Deleting", "file operation in progress");
        case "emptyTrash":
            return I18n.tr("Emptying the trash", "file operation in progress");
        default:
            return I18n.tr("Working", "file operation in progress");
        }
    }

    function detail() {
        if (job.state === "failed")
            return FileFormat.operationError(job.code, job.error);
        if (job.cancelled)
            return I18n.tr("Cancelled", "file operation was cancelled");
        if (job.state === "queued")
            return I18n.tr("Waiting", "file operation is queued");
        if (job.progress.estimating)
            return I18n.tr("Estimating…", "file operation is still measuring the work");
        if (finished)
            return I18n.tr("Finished", "file operation completed");
        return job.progress.currentPath.split("/").pop();
    }

    implicitHeight: content.implicitHeight

    ColumnLayout {
        id: content

        anchors.left: parent.left
        anchors.right: parent.right
        spacing: Theme.spacingXS

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingS

            StyledText {
                text: row.label()
                color: Theme.surfaceText
                font.pixelSize: Theme.fontSizeMedium
            }

            Item {
                Layout.fillWidth: true
            }

            StyledText {
                text: row.job.progress.filesTotal > 0 ? `${row.job.progress.filesDone}/${row.job.progress.filesTotal}` : ""
                color: Theme.surfaceVariantText
                font.pixelSize: Theme.fontSizeSmall
            }

            DankActionButton {
                visible: !row.finished
                iconName: "close"
                buttonSize: Theme.buttonHeightXS
                iconSize: Theme.iconSizeSmall
                tooltipText: I18n.tr("Cancel", "dialog button to cancel")
                onClicked: FileOpsService.cancel(row.job.id)
            }
        }

        Rectangle {
            Layout.fillWidth: true
            visible: !row.finished
            Layout.preferredHeight: Theme.spacingXS
            radius: Theme.cornerRadiusFull
            color: Theme.surfaceContainerHigh

            Rectangle {
                width: parent.width * row.fraction
                height: parent.height
                radius: parent.radius
                color: Theme.primary

                Behavior on width {
                    NumberAnimation {
                        duration: Theme.shortDuration
                        easing.type: Theme.standardEasing
                    }
                }
            }
        }

        StyledText {
            Layout.fillWidth: true
            text: row.detail()
            color: row.job.state === "failed" ? Theme.error : Theme.surfaceVariantText
            font.pixelSize: Theme.fontSizeSmall
            elide: Text.ElideMiddle
        }
    }
}
