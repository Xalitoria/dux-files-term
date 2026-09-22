pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell

Singleton {
    id: root

    // name is always first and always fills; the rest are opt-in and resizable.
    readonly property var optional: [
        {
            "id": "size",
            "sortKey": "size",
            "width": FilesMetrics.sizeColumnWidth,
            "align": Text.AlignRight
        },
        {
            "id": "modified",
            "sortKey": "mtime",
            "width": FilesMetrics.modifiedColumnWidth,
            "align": Text.AlignRight
        },
        {
            "id": "type",
            "sortKey": "type",
            "width": FilesMetrics.typeColumnWidth,
            "align": Text.AlignLeft
        },
        {
            "id": "owner",
            "sortKey": "",
            "width": FilesMetrics.ownerColumnWidth,
            "align": Text.AlignLeft
        },
        {
            "id": "permissions",
            "sortKey": "",
            "width": FilesMetrics.permissionsColumnWidth,
            "align": Text.AlignLeft
        },
        {
            "id": "created",
            "sortKey": "",
            "width": FilesMetrics.modifiedColumnWidth,
            "align": Text.AlignRight
        }
    ]

    function label(id) {
        switch (id) {
        case "name":
            return I18n.tr("Name", "label for a file name input");
        case "size":
            return I18n.tr("Size", "file list column header");
        case "modified":
            return I18n.tr("Modified", "file list column header");
        case "type":
            return I18n.tr("Type", "file list column header");
        case "owner":
            return I18n.tr("Owner", "file list column header");
        case "permissions":
            return I18n.tr("Permissions", "file list column header");
        case "created":
            return I18n.tr("Created", "file list column header");
        default:
            return id;
        }
    }

    function value(id, entry) {
        switch (id) {
        case "size":
            if (!entry.isDir)
                return FileFormat.size(entry.size);
            return entry.childCount >= 0 ? FileFormat.count(entry.childCount) : "";
        case "modified":
            return FileFormat.modified(entry.mtimeMs);
        case "type":
            return FileFormat.typeLabel(entry);
        case "owner":
            return entry.owner;
        case "permissions":
            return entry.mode;
        case "created":
            return FileFormat.modified(entry.ctimeMs);
        default:
            return "";
        }
    }

    function specFor(id) {
        return optional.find(column => column.id === id) ?? null;
    }

    function sortKeyFor(id) {
        return specFor(id)?.sortKey ?? "";
    }
}
