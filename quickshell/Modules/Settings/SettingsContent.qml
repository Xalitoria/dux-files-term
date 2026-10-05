import QtQuick
import qs.Common
import qs.Services
import qs.DCommon.Widgets
import qs.DCommon.FileBrowser

Item {
    id: root

    property int currentIndex: 0
    property var hostWindow: null
    property var terminals: []
    property var fileManagerStatus: ({})

    readonly property var colorSourceOptions: [
        {
            label: I18n.tr("Auto", "color source button group option following DMS"),
            value: "auto"
        },
        {
            label: I18n.tr("Preset", "color source button group option for bundled palettes"),
            value: "preset"
        },
        {
            label: I18n.tr("Custom", "color source button group option for a custom theme file"),
            value: "custom"
        }
    ]

    readonly property var viewOptions: [
        {
            label: I18n.tr("List view", "view mode showing one row per item"),
            value: "list"
        },
        {
            label: I18n.tr("Grid view", "view mode showing items as tiles"),
            value: "grid"
        },
        {
            label: I18n.tr("Column view", "view mode showing folders side by side"),
            value: "columns"
        }
    ]

    readonly property var gridSizeOptions: FileBrowserMetrics.gridIconSizes.map((size, index) => ({
                label: I18n.tr("%1 px", "icon size option").arg(size),
                value: index
            }))

    readonly property var listSizeOptions: FileBrowserMetrics.listIconSizes.map((size, index) => ({
                label: I18n.tr("%1 px", "icon size option").arg(size),
                value: index
            }))

    readonly property var sortOptions: [
        {
            label: I18n.tr("Name", "file browser sort criterion option"),
            value: "name"
        },
        {
            label: I18n.tr("Size", "file browser sort criterion option"),
            value: "size"
        },
        {
            label: I18n.tr("Modified", "file browser sort criterion option"),
            value: "mtime"
        },
        {
            label: I18n.tr("Type", "file browser sort criterion option"),
            value: "type"
        }
    ]

    readonly property var dragOptions: [
        {
            label: I18n.tr("Move on the same drive, copy otherwise", "drag default action option"),
            value: "auto"
        },
        {
            label: I18n.tr("Copy", "context menu entry copying the selection"),
            value: "copy"
        },
        {
            label: I18n.tr("Move", "drag label for moving files"),
            value: "move"
        },
        {
            label: I18n.tr("Ask every time", "drag default action option"),
            value: "ask"
        }
    ]

    readonly property var terminalOptions: [
        {
            label: I18n.tr("Automatic", "terminal dropdown option following $TERMINAL and the installed emulators"),
            value: ""
        }
    ].concat(terminals.map(name => ({
                label: name,
                value: name
            })))

    function localeDisplayName(code) {
        const name = I18n.presentLocales[code].nativeLanguageName;
        return name.charAt(0).toUpperCase() + name.slice(1);
    }

    readonly property var languageOptions: [
        {
            label: I18n.tr("System default", "language dropdown option that follows the system language"),
            value: ""
        }
    ].concat(Object.keys(I18n.presentLocales).sort().map(code => ({
                label: root.localeDisplayName(code),
                value: code
            })))

    function optionLabels(options) {
        return options.map(o => o.label);
    }

    function labelForValue(options, value) {
        for (let i = 0; i < options.length; i++) {
            if (options[i].value === value)
                return options[i].label;
        }
        return options[0].label;
    }

    function valueForLabel(options, label) {
        for (let i = 0; i < options.length; i++) {
            if (options[i].label === label)
                return options[i].value;
        }
        return options[0].value;
    }

    function indexForValue(options, value) {
        const index = options.findIndex(option => option.value === value);
        return index < 0 ? 0 : index;
    }

    function loadTerminals() {
        DankFilesService.sendRequest("files.apps.terminals", null, response => {
            if (response.error)
                return;
            root.terminals = (response.result || {}).terminals || [];
        });
    }

    function loadFileManagerStatus() {
        DankFilesService.sendRequest("system.fileManager.get", null, response => {
            if (response.error)
                return;
            root.fileManagerStatus = response.result || {};
        });
    }

    function setFileManager(enabled) {
        SettingsData.registerFileManager = enabled;
        DankFilesService.sendRequest("system.fileManager.set", {
            "enabled": enabled
        }, response => {
            if (response.error) {
                root.loadFileManagerStatus();
                return;
            }
            root.fileManagerStatus = response.result || {};
        });
    }

    function fileManagerDescription() {
        const status = fileManagerStatus;
        if (status.available === false)
            return I18n.tr("This instance cannot reach the session bus.", "file manager registration setting description when the daemon has no session bus connection");
        if (status.owned === true)
            return I18n.tr("Other apps open folders and reveal files here.", "file manager registration setting description while registered");
        if (status.queued === true && (status.owner ?? "") !== "")
            return I18n.tr("%1 has the role now; Files takes it when %1 quits.", "file manager registration setting description while another file manager holds the name").arg(status.owner);
        if (status.queued === true)
            return I18n.tr("Another file manager has the role now; Files takes it when that one quits.", "file manager registration setting description while an unknown file manager holds the name");
        if ((status.reason ?? "") !== "")
            return I18n.tr("The session bus refused the registration.", "file manager registration setting description when requesting the name failed");
        return I18n.tr("Other apps open folders with whichever file manager they find.", "file manager registration setting description while not registered");
    }

    Component.onCompleted: {
        loadTerminals();
        loadFileManagerStatus();
    }

    Connections {
        target: root.hostWindow
        function onVisibleChanged() {
            if (root.hostWindow.visible)
                root.loadFileManagerStatus();
        }
    }

    Loader {
        anchors.fill: parent
        sourceComponent: {
            switch (root.currentIndex) {
            case 0:
                return generalPage;
            case 1:
                return appearancePage;
            case 2:
                return behaviourPage;
            case 3:
                return devicesPage;
            case 4:
                return aboutPage;
            default:
                return generalPage;
            }
        }
    }

    component PageHeader: Column {
        property string title: ""
        property string subtitle: ""

        width: parent.width
        spacing: Theme.spacingXS
        bottomPadding: Theme.spacingS

        StyledText {
            text: parent.title
            font.pixelSize: Theme.fontSizeXLarge
            font.weight: Theme.fontWeightMedium
            color: Theme.surfaceText
            width: parent.width
            horizontalAlignment: Text.AlignLeft
        }

        StyledText {
            visible: text !== ""
            text: parent.subtitle
            font.pixelSize: Theme.fontSizeMedium
            color: Theme.surfaceVariantText
            width: parent.width
            wrapMode: Text.WordWrap
            horizontalAlignment: Text.AlignLeft
        }
    }

    component OptionDropdownRow: SettingsDropdownRow {
        property var optionList: []
        property var current: null

        signal picked(var value)

        dropdownWidth: SettingsMetrics.dropdownWidth
        options: root.optionLabels(optionList)
        currentValue: root.labelForValue(optionList, current)
        onValueChanged: label => picked(root.valueForLabel(optionList, label))
    }

    component OptionGroupRow: SettingsButtonGroupRow {
        property var optionList: []
        property var current: null

        signal picked(var value)

        model: root.optionLabels(optionList)
        currentIndex: root.indexForValue(optionList, current)
        onSelectionChanged: (index, selected) => {
            if (selected)
                picked(optionList[index].value);
        }
    }

    Component {
        id: generalPage
        SettingsPage {
            PageHeader {
                title: I18n.tr("General", "settings sidebar tab label")
                subtitle: I18n.tr("How new windows start and how folders open.", "general settings section subtitle")
            }

            SettingsSectionLabel {
                text: I18n.tr("Window", "general settings section label")
            }

            SettingsGroup {
                SettingsButtonGroupRow {
                    text: I18n.tr("Close behavior", "close behavior setting label")
                    description: I18n.tr("What the window's close button does.", "close behavior setting description")
                    model: [I18n.tr("Keep running", "close behavior button group option keeping the daemon warm"), I18n.tr("Quit", "close behavior button group option to quit the app")]
                    currentIndex: SettingsData.closeBehavior === "quit" ? 1 : 0
                    onSelectionChanged: (index, selected) => {
                        if (!selected)
                            return;
                        SettingsData.closeBehavior = index === 1 ? "quit" : "hide";
                    }
                }

                SettingsToggleRow {
                    text: I18n.tr("Register as the session file manager", "setting label for owning org.freedesktop.FileManager1")
                    description: root.fileManagerDescription()
                    enabled: root.fileManagerStatus.available !== false
                    checked: SettingsData.registerFileManager
                    onToggled: checked => root.setFileManager(checked)
                }

                SettingsToggleRow {
                    text: I18n.tr("Open folders in a new tab", "setting label for external open requests")
                    description: I18n.tr("Locations opened from other applications get their own tab instead of replacing the current one.", "setting description for external open requests")
                    checked: SettingsData.openLocationIn === "tab"
                    onToggled: checked => SettingsData.openLocationIn = checked ? "tab" : "current"
                }

                OptionDropdownRow {
                    text: I18n.tr("Language", "interface language setting label")
                    description: I18n.tr("Language of the interface.", "interface language setting description")
                    optionList: root.languageOptions
                    current: SettingsData.language
                    onPicked: value => SettingsData.language = value
                }
            }

            SettingsSectionLabel {
                text: I18n.tr("View", "keyboard shortcut group")
            }

            SettingsGroup {
                OptionGroupRow {
                    text: I18n.tr("Default view", "setting label for the view a new window starts in")
                    optionList: root.viewOptions
                    current: SettingsData.defaultView
                    onPicked: value => SettingsData.defaultView = value
                }

                OptionDropdownRow {
                    text: I18n.tr("Sort by", "setting label for the default sort key")
                    optionList: root.sortOptions
                    current: SettingsData.defaultSort
                    onPicked: value => SettingsData.defaultSort = value
                }

                SettingsToggleRow {
                    text: I18n.tr("Sort descending", "sort direction option")
                    checked: SettingsData.defaultSortDescending
                    onToggled: checked => SettingsData.defaultSortDescending = checked
                }

                SettingsToggleRow {
                    text: I18n.tr("Show hidden files", "menu entry listing dot files")
                    description: I18n.tr("Ctrl+H still toggles them for the current window.", "show hidden files setting description")
                    checked: SettingsData.showHidden
                    onToggled: checked => SettingsData.showHidden = checked
                }

                SettingsToggleRow {
                    text: I18n.tr("Folders before files", "folders first toggle label")
                    checked: SettingsData.foldersFirst
                    onToggled: checked => SettingsData.foldersFirst = checked
                }

                OptionDropdownRow {
                    text: I18n.tr("Grid icon size", "setting label for the grid view icon size")
                    description: I18n.tr("Ctrl+Wheel or Ctrl and plus or minus resize the current window.", "icon size setting description")
                    optionList: root.gridSizeOptions
                    current: SessionData.gridZoom
                    onPicked: value => SessionData.gridZoom = value
                }

                OptionDropdownRow {
                    text: I18n.tr("List icon size", "setting label for the list view icon size")
                    optionList: root.listSizeOptions
                    current: SessionData.listZoom
                    onPicked: value => SessionData.listZoom = value
                }

                SettingsToggleRow {
                    text: I18n.tr("Count items in folders", "folder counts toggle label")
                    description: I18n.tr("The size column shows how many entries a folder holds, one request per folder in view.", "folder counts toggle description")
                    checked: SettingsData.folderCounts
                    onToggled: checked => SettingsData.folderCounts = checked
                }
            }

            SettingsSectionLabel {
                text: I18n.tr("Files", "keyboard shortcut group")
            }

            SettingsGroup {
                SettingsButtonGroupRow {
                    text: I18n.tr("Open items with", "setting label choosing single or double click")
                    model: [I18n.tr("Double click", "click to open option"), I18n.tr("Single click", "click to open option")]
                    currentIndex: SettingsData.clickToOpen === "single" ? 1 : 0
                    onSelectionChanged: (index, selected) => {
                        if (!selected)
                            return;
                        SettingsData.clickToOpen = index === 1 ? "single" : "double";
                    }
                }

                SettingsToggleRow {
                    text: I18n.tr("Confirm before moving to the trash", "setting label")
                    checked: SettingsData.confirmTrash
                    onToggled: checked => SettingsData.confirmTrash = checked
                }
            }

            SettingsSectionLabel {
                text: I18n.tr("Keyboard", "general settings section label")
            }

            SettingsGroup {
                SettingsToggleRow {
                    text: I18n.tr("Vim keys", "toggle label for vim style navigation")
                    description: I18n.tr("h j k l move, / searches and letters run commands instead of jumping to a name.", "vim keys toggle description")
                    checked: SettingsData.vimKeys
                    onToggled: checked => SettingsData.vimKeys = checked
                }

                SettingsRow {
                    title: I18n.tr("Keyboard shortcuts", "overlay listing every keyboard shortcut")
                    subtitle: I18n.tr("Press ? or F1 in the main window for the full list.", "keyboard shortcuts settings row description")
                    iconName: "keyboard"
                }
            }
        }
    }

    Component {
        id: appearancePage
        SettingsPage {
            id: appearanceFlickable

            PageHeader {
                title: I18n.tr("Appearance", "settings sidebar tab label")
                subtitle: I18n.tr("Pick a color source, palette, or your own theme file.", "appearance settings section subtitle")
            }

            SettingsSectionLabel {
                text: I18n.tr("Theme", "appearance settings section label")
            }

            SettingsGroup {
                SettingsButtonGroupRow {
                    text: I18n.tr("Theme", "theme mode setting label")
                    description: {
                        switch (SettingsData.themeMode) {
                        case "light":
                            return I18n.tr("Always use the light theme.", "theme mode setting description");
                        case "dark":
                            return I18n.tr("Always use the dark theme.", "theme mode setting description");
                        default:
                            return PortalService.available ? I18n.tr("Following the system color scheme (currently %1).", "theme mode setting description").arg(Theme.isLightMode ? I18n.tr("light", "current scheme name in theme mode description") : I18n.tr("dark", "current scheme name in theme mode description")) : I18n.tr("System preference unavailable, using dark.", "theme mode setting description");
                        }
                    }
                    model: [I18n.tr("Auto", "theme mode button group option"), I18n.tr("Light", "theme mode button group option"), I18n.tr("Dark", "theme mode button group option")]
                    currentIndex: {
                        switch (SettingsData.themeMode) {
                        case "light":
                            return 1;
                        case "dark":
                            return 2;
                        default:
                            return 0;
                        }
                    }
                    onSelectionChanged: (index, selected) => {
                        if (!selected)
                            return;
                        switch (index) {
                        case 1:
                            SettingsData.themeMode = "light";
                            break;
                        case 2:
                            SettingsData.themeMode = "dark";
                            break;
                        default:
                            SettingsData.themeMode = "auto";
                            break;
                        }
                    }
                }

                OptionGroupRow {
                    text: I18n.tr("Color source", "color source setting label")
                    description: {
                        switch (SettingsData.colorSource) {
                        case "preset":
                            return I18n.tr("Use a bundled color palette.", "color source description for preset");
                        case "custom":
                            return I18n.tr("Load colors from your own theme file.", "color source description for custom");
                        default:
                            return I18n.tr("Follow DankMaterialShell colors, falling back to a preset.", "color source description for auto");
                        }
                    }
                    optionList: root.colorSourceOptions
                    current: SettingsData.colorSource
                    onPicked: value => SettingsData.colorSource = value
                }

                SettingsRow {
                    visible: SettingsData.colorSource === "auto"
                    iconName: Theme.dmsColorsAvailable ? "check_circle" : "info"
                    iconColor: Theme.dmsColorsAvailable ? Theme.success : Theme.surfaceVariantText
                    title: Theme.dmsColorsAvailable ? I18n.tr("Using DankMaterialShell dynamic colors.", "auto color source status when DMS is active") : I18n.tr("DankMaterialShell not detected, using the %1 preset.", "auto color source status when DMS is missing").arg(Theme.presetLabel(SettingsData.presetTheme))
                }

                SettingsRow {
                    visible: SettingsData.colorSource === "preset"
                    title: I18n.tr("Palette: %1", "selected preset palette name").arg(Theme.presetLabel(SettingsData.presetTheme))

                    body: Flow {
                        width: parent.width
                        spacing: Theme.spacingS

                        Repeater {
                            model: Theme.presetNames()

                            Rectangle {
                                id: swatch
                                required property string modelData
                                readonly property bool active: SettingsData.presetTheme === modelData

                                width: Theme.avatarSize
                                height: Theme.avatarSize
                                radius: Theme.fullRadius(width, height)
                                color: Theme.presetColors(modelData).primary
                                border.color: active ? Theme.primary : Theme.outlineVariant
                                border.width: active ? Theme.outlineWidthFocused + Theme.outlineWidth : Theme.outlineWidth
                                Accessible.role: Accessible.RadioButton
                                Accessible.name: Theme.presetLabel(modelData)
                                Accessible.checked: active

                                StateLayer {
                                    stateColor: Theme.surfaceText
                                    cornerRadius: parent.radius
                                    onClicked: SettingsData.presetTheme = swatch.modelData
                                }
                            }
                        }
                    }
                }

                SettingsRow {
                    visible: SettingsData.colorSource === "custom"
                    clickable: true
                    iconName: "folder_open"
                    title: SettingsData.customThemeFile ? SettingsData.customThemeFile.split('/').pop() : I18n.tr("No theme file selected", "custom theme empty state title")
                    subtitle: {
                        if (!SettingsData.customThemeFile)
                            return I18n.tr("Choose a JSON color theme file.", "custom theme hint when none selected");
                        if (Theme.customThemeLoaded)
                            return SettingsData.customThemeFile;
                        return I18n.tr("Could not read that file, expected a JSON color theme.", "custom theme error when file is invalid");
                    }
                    subtitleColor: (SettingsData.customThemeFile && !Theme.customThemeLoaded) ? Theme.error : Theme.surfaceVariantText
                    showChevron: true
                    onClicked: {
                        themePickerLoader.active = true;
                        themePickerLoader.item.open();
                    }
                }
            }

            SettingsSectionLabel {
                text: I18n.tr("Shape", "appearance settings section label")
            }

            SettingsGroup {
                SettingsSliderRow {
                    text: I18n.tr("Radius strength", "global component corner rounding")
                    description: I18n.tr("50 uses Material shapes. Lower values reduce rounding; higher values increase it.", "radius strength slider description")
                    minimumLabel: I18n.tr("Square", "radius strength slider label at zero")
                    value: SettingsData.radiusStrength
                    minimum: 0
                    maximum: 100
                    unit: ""
                    onSliderValueChanged: newValue => SettingsData.radiusStrength = newValue
                }
            }

            SettingsSectionLabel {
                text: I18n.tr("Typography", "appearance settings section label")
            }

            SettingsGroup {
                SettingsSliderRow {
                    text: I18n.tr("Weight", "font weight slider label")
                    minimum: Font.Thin
                    maximum: Font.Black
                    step: 100
                    showStops: true
                    unit: ""
                    value: SettingsData.fontWeight
                    onSliderValueChanged: newValue => SettingsData.fontWeight = newValue
                }

                SettingsSliderRow {
                    text: I18n.tr("Scale", "font scale slider label")
                    minimum: 75
                    maximum: 150
                    unit: "%"
                    value: Math.round(SettingsData.fontScale * 100)
                    onSliderValueChanged: newValue => SettingsData.fontScale = newValue / 100
                }
            }

            SettingsSectionLabel {
                text: I18n.tr("Motion", "appearance settings section label")
            }

            SettingsGroup {
                SettingsSliderRow {
                    text: I18n.tr("Duration", "animation duration slider label")
                    minimumLabel: I18n.tr("Off", "animation duration slider label at zero")
                    minimum: 0
                    maximum: 1000
                    unit: "ms"
                    value: SettingsData.animationDuration
                    onSliderValueChanged: newValue => SettingsData.animationDuration = newValue
                }

                SettingsButtonGroupRow {
                    text: I18n.tr("Spring", "spring bounce setting label")
                    model: [I18n.tr("Smooth", "spring bounce option"), I18n.tr("Balanced", "spring bounce option"), I18n.tr("Playful", "spring bounce option")]
                    currentIndex: SettingsData.springBounce
                    onSelectionChanged: (index, selected) => {
                        if (selected)
                            SettingsData.springBounce = index;
                    }
                }

                SettingsToggleRow {
                    text: I18n.tr("Ripple effects", "ripple effects toggle label")
                    checked: SettingsData.enableRippleEffects
                    onToggled: checked => SettingsData.enableRippleEffects = checked
                }

                SettingsToggleRow {
                    text: I18n.tr("Reduce motion", "reduce motion toggle label")
                    checked: SettingsData.reduceMotion
                    onToggled: checked => SettingsData.reduceMotion = checked
                }
            }

            SettingsSectionLabel {
                text: I18n.tr("Focus ring", "appearance settings section label")
            }

            SettingsGroup {
                SettingsToggleRow {
                    text: I18n.tr("Focus ring", "focus ring toggle label")
                    description: I18n.tr("Outline the control that has keyboard focus.", "focus ring toggle description")
                    checked: SettingsData.focusRingEnabled
                    onToggled: checked => SettingsData.focusRingEnabled = checked
                }

                SettingsSliderRow {
                    text: I18n.tr("Thickness", "focus ring thickness slider label")
                    enabled: SettingsData.focusRingEnabled
                    value: Math.round(SettingsData.focusRingWidth * 10)
                    minimum: 10
                    maximum: 40
                    step: 5
                    decimals: 1
                    unit: "px"
                    onSliderValueChanged: newValue => SettingsData.focusRingWidth = newValue / 10
                }
            }

            Loader {
                id: themePickerLoader
                active: false
                sourceComponent: FileBrowserModal {
                    parentModal: root.hostWindow
                    bucket: "theme"
                    filters: ["*.json"]
                    onAccepted: paths => SettingsData.customThemeFile = paths[0]
                }
            }
        }
    }

    Component {
        id: behaviourPage
        SettingsPage {
            PageHeader {
                title: I18n.tr("Behaviour", "settings sidebar tab label")
                subtitle: I18n.tr("Terminal, drag and drop, thumbnails and search.", "behaviour settings section subtitle")
            }

            SettingsSectionLabel {
                text: I18n.tr("Open in terminal", "context menu entry opening a terminal")
            }

            SettingsGroup {
                OptionDropdownRow {
                    text: I18n.tr("Terminal", "setting label for the terminal emulator")
                    description: SandboxService.sandboxed ? I18n.tr("The sandbox cannot see the host's terminals; name the one to launch.", "terminal setting description inside flatpak") : I18n.tr("Automatic follows $TERMINAL, then the first installed emulator.", "terminal setting description")
                    optionList: root.terminalOptions
                    current: SettingsData.terminal
                    onPicked: value => SettingsData.terminal = value
                }
            }

            SettingsSectionLabel {
                text: I18n.tr("Drag and drop", "behaviour settings section label")
            }

            SettingsGroup {
                OptionDropdownRow {
                    text: I18n.tr("Dropping without a modifier", "drag default action setting label")
                    description: I18n.tr("Ctrl copies, Shift moves and Alt asks regardless.", "drag default action setting description")
                    optionList: root.dragOptions
                    current: SettingsData.dragAction
                    onPicked: value => SettingsData.dragAction = value
                }
            }

            SettingsSectionLabel {
                text: I18n.tr("Thumbnails", "capability row on the about page")
            }

            SettingsGroup {
                SettingsToggleRow {
                    text: I18n.tr("Show thumbnails", "thumbnails toggle label")
                    description: SandboxService.degraded("thumbnails") ? FilesFormat.featureUnavailable(SandboxService.reason("thumbnails")) : I18n.tr("Previews for images, videos and PDFs, cached under the shared thumbnail folder.", "thumbnails toggle description")
                    checked: SettingsData.thumbnails
                    onToggled: checked => SettingsData.thumbnails = checked
                }

                SettingsSliderRow {
                    text: I18n.tr("Skip images larger than", "thumbnail size cap slider label")
                    minimumLabel: I18n.tr("No limit", "thumbnail size cap slider label at zero")
                    enabled: SettingsData.thumbnails
                    minimum: 0
                    maximum: 1024
                    step: 16
                    unit: " MB"
                    value: SettingsData.thumbnailSizeLimitMb
                    onSliderValueChanged: newValue => SettingsData.thumbnailSizeLimitMb = newValue
                }

                SettingsToggleRow {
                    text: I18n.tr("Thumbnails on network locations", "network thumbnails toggle label")
                    description: I18n.tr("Reads every file over the connection to draw its preview.", "network thumbnails toggle description")
                    enabled: SettingsData.thumbnails
                    checked: SettingsData.networkThumbnails
                    onToggled: checked => SettingsData.networkThumbnails = checked
                }
            }

            SettingsSectionLabel {
                text: I18n.tr("Search in Files", "placeholder in the search field")
            }

            SettingsGroup {
                SettingsToggleRow {
                    text: I18n.tr("Include hidden files", "search hidden files toggle label")
                    description: I18n.tr("Recursive searches also descend into dot folders.", "search hidden files toggle description")
                    checked: SettingsData.searchHidden
                    onToggled: checked => SettingsData.searchHidden = checked
                }
            }
        }
    }

    Component {
        id: devicesPage
        SettingsPage {
            PageHeader {
                title: I18n.tr("Devices", "sidebar section listing drives and volumes")
                subtitle: I18n.tr("What the sidebar lists and what a click does.", "devices settings section subtitle")
            }

            SettingsGroup {
                SettingsToggleRow {
                    text: I18n.tr("Show system partitions", "devices setting label")
                    description: I18n.tr("Internal partitions udisks marks as part of the system, such as a separate /home.", "devices setting description")
                    checked: SettingsData.showSystemPartitions
                    onToggled: checked => SettingsData.showSystemPartitions = checked
                }

                SettingsToggleRow {
                    text: I18n.tr("Open unmounted drives on click", "devices setting label")
                    description: I18n.tr("Mount and open the drive together. Off mounts it and leaves you where you are.", "devices setting description")
                    checked: SettingsData.mountOnClick
                    onToggled: checked => SettingsData.mountOnClick = checked
                }
            }

            SettingsRow {
                visible: !SandboxService.available("devices")
                iconName: "info"
                iconColor: Theme.surfaceVariantText
                title: FilesFormat.featureUnavailable(SandboxService.reason("devices"))
            }
        }
    }

    Component {
        id: aboutPage
        SettingsAboutPage {}
    }
}
