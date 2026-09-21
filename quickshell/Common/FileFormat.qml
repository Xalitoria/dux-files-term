pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell

Singleton {
    id: root

    function size(bytes) {
        if (bytes < 0)
            return "";
        const units = [I18n.tr("B", "file size unit"), I18n.tr("kB", "file size unit"), I18n.tr("MB", "file size unit"), I18n.tr("GB", "file size unit"), I18n.tr("TB", "file size unit")];
        let value = bytes;
        let unit = 0;
        while (value >= 1000 && unit < units.length - 1) {
            value /= 1000;
            unit++;
        }
        return `${value.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
    }

    function operationError(code, fallback) {
        switch (code) {
        case "ENOENT":
            return I18n.tr("This item no longer exists", "file operation error");
        case "EACCES":
            return I18n.tr("You do not have permission to do this", "file operation error");
        case "EEXIST":
            return I18n.tr("This item already exists", "file conflict dialog header");
        case "EBUSY":
            return I18n.tr("This item is in use", "file operation error");
        case "EXDEV":
            return I18n.tr("This item cannot be moved here directly", "file operation error");
        case "NOTSUPPORTED":
        case "":
            return fallback || I18n.tr("This item could not be handled", "file operation error");
        default:
            return I18n.tr("This item could not be handled", "file operation error");
        }
    }

    function elevationUnavailable(reason) {
        switch (reason) {
        case "SANDBOXED":
            return I18n.tr("Administrator access is not available inside the sandbox", "file operation error dialog explanation");
        case "NOPKEXEC":
            return I18n.tr("Administrator access is not available because pkexec is not installed", "file operation error dialog explanation");
        case "NOPOLICY":
            return I18n.tr("Administrator access is not available because the Dank Files polkit policy is not installed", "file operation error dialog explanation");
        default:
            return I18n.tr("Administrator access is not available", "file operation error dialog explanation");
        }
    }

    function featureUnavailable(reason) {
        switch (reason) {
        case "SANDBOXED":
            return I18n.tr("Not available inside the sandbox", "explanation for a feature the flatpak sandbox blocks");
        case "NOHOSTFS":
            return I18n.tr("Not available without access to the host filesystem", "explanation for a feature that needs the flatpak host filesystem permission");
        case "NOUDISKS":
            return I18n.tr("Drives are unavailable because the sandbox cannot reach udisks2", "explanation when the flatpak lacks the udisks2 permission");
        case "NOGVFS":
            return I18n.tr("Network locations are unavailable because the sandbox cannot reach gvfs", "explanation when the flatpak lacks the gvfs permissions");
        case "NOHOSTSPAWN":
            return I18n.tr("Not available because the sandbox cannot start programs on the host", "explanation for a feature that needs flatpak-spawn");
        case "PORTALCHOOSER":
            return I18n.tr("The system application chooser opens instead", "explanation when open with is handed to the desktop portal");
        default:
            return I18n.tr("Not available", "explanation for a feature the environment does not permit");
        }
    }

    // A file the user would run rather than open: the execute bit plus a type
    // that is either a program or a script.
    function isRunnable(mime, executable) {
        if (!executable)
            return false;
        switch (true) {
        case mime === "":
        case mime.startsWith("text/"):
        case mime.endsWith("script"):
        case mime === "application/x-executable":
        case mime === "application/x-sharedlib":
        case mime === "application/x-pie-executable":
            return true;
        }
        return false;
    }

    function modified(mtimeMs) {
        if (!mtimeMs)
            return "";
        return new Date(mtimeMs).toLocaleString(Qt.locale(), Locale.ShortFormat);
    }
}
