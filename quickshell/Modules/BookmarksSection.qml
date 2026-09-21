pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services

SidebarSection {
    id: section

    property string currentPath: ""

    signal navigateRequested(string path)

    readonly property var entries: {
        const out = [];
        for (const dir of DirectoryService.userDirs) {
            out.push({
                "path": dir.path,
                "label": dir.key === "home" ? I18n.tr("Home", "sidebar entry for the user's home folder") : dir.name,
                "iconName": dir.iconName,
                "removable": false
            });
        }
        const known = out.map(entry => entry.path);
        for (const path of BookmarksService.paths) {
            if (known.includes(path))
                continue;
            out.push({
                "path": path,
                "label": path.substring(path.lastIndexOf("/") + 1) || path,
                "iconName": "folder",
                "removable": true
            });
        }
        return out;
    }

    title: I18n.tr("Bookmarks", "sidebar section listing saved folders")

    Repeater {
        model: section.entries

        SidebarRow {
            required property var modelData

            label: modelData.label
            iconName: modelData.iconName
            selected: modelData.path === section.currentPath
            actionIcon: modelData.removable ? "bookmark_remove" : ""
            actionTooltip: I18n.tr("Remove bookmark", "tooltip on the bookmark button")
            onClicked: section.navigateRequested(modelData.path)
            onActionTriggered: BookmarksService.remove(modelData.path)
        }
    }
}
