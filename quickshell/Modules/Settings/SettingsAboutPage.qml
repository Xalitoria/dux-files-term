pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.DCommon.Widgets

Item {
    id: root

    LayoutMirroring.enabled: I18n.isRtl
    LayoutMirroring.childrenInherit: true

    readonly property string githubUrl: "https://github.com/AvengeMedia/dankfiles"
    readonly property string docsUrl: "https://danklinux.com/docs"
    readonly property string kofiUrl: "https://ko-fi.com/danklinux"
    readonly property string licenseUrl: githubUrl + "/blob/master/LICENSE"
    // Only a release number belongs in the headline; dev builds keep their full string in the Backend rows
    readonly property string version: DankFilesService.daemonVersion.match(/^v?(\d+\.\d+(\.\d+)?)$/)?.[1] ?? ""

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
        },
        {
            "id": "indexSearch",
            "label": I18n.tr("Search with the danksearch index", "capability row on the about page")
        }
    ]

    function host(url) {
        return url.replace(/^https?:\/\//, "").replace(/\/$/, "");
    }

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

    SettingsPage {
        DHeroCard {
            brand: I18n.tr("Files", "main window title")
            accent: root.version
            caption: I18n.tr("Part of the Dank Linux suite.", "app tagline on about page")

            DButton {
                anchors.horizontalCenter: parent.horizontalCenter
                text: I18n.tr("Support Dank Linux", "about page hero button, opens the Ko-fi donation page")
                iconName: "favorite"
                backgroundColor: Theme.primary
                textColor: Theme.onPrimary
                onClicked: DankFilesService.openUri(root.kofiUrl)
            }
        }

        SettingsGroup {
            SettingsLinkRow {
                iconName: "menu_book"
                title: I18n.tr("Docs", "about page resource button")
                subtitle: root.host(root.docsUrl)
                url: root.docsUrl
            }

            SettingsLinkRow {
                iconName: "code"
                title: I18n.tr("GitHub", "about page resource button")
                subtitle: root.host(root.githubUrl)
                url: root.githubUrl
            }
        }

        SettingsSectionLabel {
            text: I18n.tr("Backend", "about page daemon section header")
        }

        SettingsGroup {
            SettingsRow {
                title: I18n.tr("Version", "about page daemon version column label")
                trailingBadge: DankFilesService.daemonVersion || "—"
            }

            SettingsRow {
                title: I18n.tr("API", "about page daemon api version column label")
                trailingBadge: DankFilesService.apiVersion > 0 ? `v${DankFilesService.apiVersion}` : "—"
            }

            SettingsRow {
                title: I18n.tr("Status", "about page daemon status column label")
                trailingBadge: DankFilesService.connected ? I18n.tr("Connected", "about page daemon status value") : I18n.tr("Offline", "about page daemon status value")

                DBadge {
                    color: DankFilesService.connected ? Theme.success : Theme.error
                }
            }

            SettingsRow {
                title: I18n.tr("Socket", "about page daemon socket row label")
                subtitle: DankFilesService.socketPath || "—"
            }

            SettingsRow {
                visible: SandboxService.sandboxed
                title: I18n.tr("Sandbox", "about page row naming the flatpak id")
                subtitle: SandboxService.appId
            }
        }

        SettingsSectionLabel {
            text: I18n.tr("Capabilities", "about page daemon capabilities label")
        }

        SettingsGroup {
            Repeater {
                model: root.featureRows

                SettingsRow {
                    required property var modelData

                    readonly property var status: root.statusFor(modelData.id)

                    title: modelData.label
                    trailingBadge: status.text
                    trailingBadgeColor: status.color
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
                onClicked: DankFilesService.openUri(root.licenseUrl)
            }
        }
    }
}
