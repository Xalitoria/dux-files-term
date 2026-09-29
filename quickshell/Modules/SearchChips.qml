pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.DankCommon.Widgets

// The filter row above search results: type chips toggle, the date and size
// chips pick one each, hidden is a toggle. Every change restarts the search.
Flow {
    id: chips

    property var types: []
    property string modified: ""
    property string size: ""
    property bool includeHidden: SettingsData.searchHidden

    readonly property var filterState: ({
            "types": types,
            "modified": modified,
            "size": size,
            "includeHidden": includeHidden
        })

    signal changed

    function reset() {
        types = [];
        modified = "";
        size = "";
        changed();
    }

    // A Flow only knows its single-row width once it has one; summing the
    // chips keeps each group on one line and lets the outer Flow wrap groups.
    function rowWidth(group) {
        let total = 0;
        const items = group.children;
        for (let i = 0; i < items.length; i++) {
            const width = items[i].width;
            if (!(width > 0))
                continue;
            total += width + (total > 0 ? group.spacing : 0);
        }
        return total;
    }

    spacing: Theme.spacingS
    LayoutMirroring.enabled: I18n.isRtl
    LayoutMirroring.childrenInherit: true

    DankFilterChips {
        id: typeChips

        width: chips.rowWidth(typeChips)
        model: SearchFilters.types
        multiSelect: true
        selectedValues: chips.types
        showCounts: false
        onSelectionToggled: (index, selected) => {
            const value = SearchFilters.types[index].value;
            chips.types = selected ? chips.types.concat([value]) : chips.types.filter(entry => entry !== value);
            chips.changed();
        }
    }

    Rectangle {
        width: Theme.dividerWidth
        height: Theme.buttonHeightXS
        color: Theme.outlineVariant
    }

    DankFilterChips {
        id: modifiedChips

        width: chips.rowWidth(modifiedChips)
        model: SearchFilters.modified
        currentIndex: Math.max(0, SearchFilters.modified.findIndex(entry => entry.value === chips.modified))
        showCounts: false
        onSelectionChanged: index => {
            chips.modified = SearchFilters.modified[index].value;
            chips.changed();
        }
    }

    Rectangle {
        width: Theme.dividerWidth
        height: Theme.buttonHeightXS
        color: Theme.outlineVariant
    }

    DankFilterChips {
        id: sizeChips

        width: chips.rowWidth(sizeChips)
        model: SearchFilters.sizes
        currentIndex: Math.max(0, SearchFilters.sizes.findIndex(entry => entry.value === chips.size))
        showCounts: false
        onSelectionChanged: index => {
            chips.size = SearchFilters.sizes[index].value;
            chips.changed();
        }
    }

    Rectangle {
        width: Theme.dividerWidth
        height: Theme.buttonHeightXS
        color: Theme.outlineVariant
    }

    DankFilterChips {
        id: hiddenChips

        width: chips.rowWidth(hiddenChips)
        model: [
            {
                "value": "hidden",
                "label": I18n.tr("Show hidden files", "menu entry listing dot files")
            }
        ]
        multiSelect: true
        selectedValues: chips.includeHidden ? ["hidden"] : []
        showCounts: false
        onSelectionToggled: (index, selected) => {
            chips.includeHidden = selected;
            SettingsData.searchHidden = selected;
            chips.changed();
        }
    }
}
