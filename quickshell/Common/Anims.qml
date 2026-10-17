pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell

Singleton {
    readonly property int durShort: Theme.expressiveDurations.normal
    readonly property int durLong: 600

    readonly property var standard: Theme.expressiveCurves.standard
    readonly property var emphasized: Theme.expressiveCurves.emphasized
}
