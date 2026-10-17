pragma Singleton

import QtQuick
import Quickshell

Singleton {
    function isInside(item, ancestor) {
        for (let current = item; current; current = current.parent) {
            if (current === ancestor)
                return true;
        }
        return false;
    }
}
