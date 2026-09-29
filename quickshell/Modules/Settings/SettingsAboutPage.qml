import QtQuick
import qs.Common
import qs.Services
import qs.DankCommon.Widgets

Item {
    id: aboutPage

    LayoutMirroring.enabled: I18n.isRtl
    LayoutMirroring.childrenInherit: true

    readonly property string githubUrl: "https://github.com/AvengeMedia/dankfiles"
    readonly property string docsUrl: "https://danklinux.com/docs"
    readonly property string licenseUrl: githubUrl + "/blob/master/LICENSE"

    readonly property string versionText: {
        const version = DankFilesService.daemonVersion;
        if (!version)
            return "dfiles";
        if (/^[\d.]+$/.test(version))
            return `dfiles v${version}`;
        return `dfiles ${version}`;
    }

    readonly property var featureRows: [
        {
            "id": "hostFilesystem",
            "label": I18n.tr("Host filesystem", "capability row on the about page")
        },
        {
            "id": "trash",
            "label": I18n.tr("Trash", "capability row on the about page")
        },
        {
            "id": "thumbnails",
            "label": I18n.tr("Thumbnails", "capability row on the about page")
        },
        {
            "id": "devices",
            "label": I18n.tr("Devices", "sidebar section listing drives and volumes")
        },
        {
            "id": "network",
            "label": I18n.tr("Network", "sidebar section listing network locations")
        },
        {
            "id": "terminal",
            "label": I18n.tr("Open in terminal", "context menu entry opening a terminal")
        },
        {
            "id": "run",
            "label": I18n.tr("Run programs", "capability row on the about page")
        },
        {
            "id": "openWith",
            "label": I18n.tr("Open with", "open with dialog header")
        },
        {
            "id": "setDefault",
            "label": I18n.tr("Default applications", "capability row on the about page")
        },
        {
            "id": "elevation",
            "label": I18n.tr("Retry as administrator", "file operation error dialog action")
        }
    ]

    function statusFor(id) {
        const feature = SandboxService.features[id];
        if (!feature)
            return {
                "text": I18n.tr("Unknown", "capability status when the daemon did not report it"),
                "color": Theme.surfaceVariantText
            };
        if (!feature.available)
            return {
                "text": FilesFormat.featureUnavailable(feature.reason),
                "color": Theme.error
            };
        if (feature.degraded)
            return {
                "text": FilesFormat.featureUnavailable(feature.reason),
                "color": Theme.tertiary
            };
        return {
            "text": I18n.tr("Available", "capability status on the about page"),
            "color": Theme.success
        };
    }

    component InfoCard: StyledRect {
        id: card

        default property alias content: cardColumn.data
        property string heading: ""
        property string icon: ""

        width: parent.width
        height: cardColumn.implicitHeight + Theme.spacingL * 2
        radius: Theme.cornerRadiusL
        color: Theme.surfaceContainerHigh

        Column {
            id: cardColumn

            anchors.fill: parent
            anchors.margins: Theme.spacingL
            spacing: Theme.spacingM

            Row {
                width: parent.width
                spacing: Theme.spacingM

                DankIcon {
                    name: card.icon
                    size: Theme.iconSize
                    color: Theme.primary
                    anchors.verticalCenter: parent.verticalCenter
                }

                StyledText {
                    text: card.heading
                    font.pixelSize: Theme.fontSizeLarge
                    font.weight: Theme.fontWeightMedium
                    color: Theme.surfaceText
                    anchors.verticalCenter: parent.verticalCenter
                }
            }
        }
    }

    component FactColumn: Column {
        property string label: ""
        property string value: ""

        spacing: Theme.spacingXXS

        StyledText {
            text: parent.label
            font.pixelSize: Theme.fontSizeSmall
            color: Theme.surfaceVariantText
        }

        StyledText {
            text: parent.value
            font.pixelSize: Theme.fontSizeMedium
            font.weight: Theme.fontWeightMedium
            color: Theme.surfaceText
        }
    }

    DankFlickable {
        anchors.fill: parent
        clip: true
        contentHeight: mainColumn.height + Theme.spacingXL
        contentWidth: width

        Column {
            id: mainColumn
            topPadding: Theme.spacingL

            width: Math.min(SettingsMetrics.contentMaxWidth, parent.width - Theme.spacingL * 2)
            anchors.horizontalCenter: parent.horizontalCenter
            spacing: Theme.spacingXL

            StyledRect {
                width: parent.width
                height: headerSection.implicitHeight + Theme.spacingL * 2
                radius: Theme.cornerRadiusL
                color: Theme.surfaceContainerHigh

                Column {
                    id: headerSection

                    anchors.fill: parent
                    anchors.margins: Theme.spacingL
                    spacing: Theme.spacingM

                    Rectangle {
                        anchors.horizontalCenter: parent.horizontalCenter
                        width: Theme.avatarSize * 2
                        height: width
                        radius: Theme.fullRadius(width, height)
                        color: Theme.withAlpha(Theme.primary, Theme.tonalTintAlpha)

                        DankIcon {
                            anchors.centerIn: parent
                            name: "folder"
                            size: Theme.iconSizeLarge
                            color: Theme.primary
                        }
                    }

                    StyledText {
                        text: I18n.tr("Files", "main window title")
                        font.pixelSize: Theme.fontSizeXXLarge
                        font.weight: Theme.fontWeightBold
                        color: Theme.surfaceText
                        horizontalAlignment: Text.AlignHCenter
                        width: parent.width
                    }

                    StyledText {
                        text: aboutPage.versionText
                        font.pixelSize: Theme.fontSizeLarge
                        font.weight: Theme.fontWeightMedium
                        color: Theme.surfaceText
                        horizontalAlignment: Text.AlignHCenter
                        width: parent.width
                    }

                    StyledText {
                        text: I18n.tr("Part of the Dank Linux suite.", "app tagline on about page")
                        font.pixelSize: Theme.fontSizeMedium
                        font.italic: true
                        color: Theme.surfaceVariantText
                        horizontalAlignment: Text.AlignHCenter
                        width: parent.width
                    }

                    Row {
                        anchors.horizontalCenter: parent.horizontalCenter
                        spacing: Theme.spacingS

                        DankButton {
                            text: I18n.tr("Docs", "about page resource button")
                            iconName: "menu_book"
                            iconSize: Theme.chipIconSize
                            backgroundColor: Theme.secondaryContainer
                            textColor: Theme.onSecondaryContainer
                            onClicked: DankFilesService.openUri(aboutPage.docsUrl)
                        }

                        DankButton {
                            text: I18n.tr("GitHub", "about page resource button")
                            iconName: "code"
                            iconSize: Theme.chipIconSize
                            backgroundColor: Theme.secondaryContainer
                            textColor: Theme.onSecondaryContainer
                            onClicked: DankFilesService.openUri(aboutPage.githubUrl)
                        }
                    }
                }
            }

            InfoCard {
                heading: I18n.tr("Backend", "about page daemon section header")
                icon: "dns"

                Row {
                    spacing: Theme.spacingL

                    FactColumn {
                        label: I18n.tr("Version", "about page daemon version column label")
                        value: DankFilesService.daemonVersion || "—"
                    }

                    Rectangle {
                        width: Theme.dividerWidth
                        height: Theme.iconSizeLarge
                        color: Theme.outlineVariant
                    }

                    FactColumn {
                        label: I18n.tr("API", "about page daemon api version column label")
                        value: DankFilesService.apiVersion > 0 ? `v${DankFilesService.apiVersion}` : "—"
                    }

                    Rectangle {
                        width: Theme.dividerWidth
                        height: Theme.iconSizeLarge
                        color: Theme.outlineVariant
                    }

                    FactColumn {
                        label: I18n.tr("Status", "about page daemon status column label")
                        value: DankFilesService.connected ? I18n.tr("Connected", "about page daemon status value") : I18n.tr("Offline", "about page daemon status value")
                    }
                }

                FactColumn {
                    width: parent.width
                    label: I18n.tr("Socket", "about page daemon socket row label")
                    value: DankFilesService.socketPath || "—"
                }

                FactColumn {
                    width: parent.width
                    visible: SandboxService.sandboxed
                    label: I18n.tr("Sandbox", "about page row naming the flatpak id")
                    value: SandboxService.appId
                }
            }

            InfoCard {
                heading: I18n.tr("Capabilities", "about page daemon capabilities label")
                icon: "verified"

                Column {
                    width: parent.width
                    spacing: Theme.spacingS

                    Repeater {
                        model: aboutPage.featureRows

                        Item {
                            id: featureRow

                            required property var modelData

                            readonly property var status: aboutPage.statusFor(modelData.id)

                            width: parent.width
                            height: Math.max(featureLabel.implicitHeight, featureStatus.implicitHeight)

                            StyledText {
                                id: featureLabel

                                anchors.left: parent.left
                                anchors.right: featureStatus.left
                                anchors.rightMargin: Theme.spacingM
                                anchors.verticalCenter: parent.verticalCenter
                                text: featureRow.modelData.label
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.surfaceText
                                elide: Text.ElideRight
                            }

                            StyledText {
                                id: featureStatus

                                anchors.right: parent.right
                                anchors.verticalCenter: parent.verticalCenter
                                width: Math.min(implicitWidth, featureRow.width / 2)
                                text: featureRow.status.text
                                font.pixelSize: Theme.fontSizeSmall
                                color: featureRow.status.color
                                horizontalAlignment: Text.AlignRight
                                elide: Text.ElideRight
                            }
                        }
                    }
                }
            }

            StyledText {
                anchors.horizontalCenter: parent.horizontalCenter
                text: I18n.tr("MIT License", "about page license footer link")
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.primary

                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.PointingHandCursor
                    onClicked: DankFilesService.openUri(aboutPage.licenseUrl)
                }
            }
        }
    }
}
