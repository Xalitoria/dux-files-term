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
