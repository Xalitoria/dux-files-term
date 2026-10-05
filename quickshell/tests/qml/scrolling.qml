pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Window
import Quickshell
import qs.Services
import qs.DCommon.FileBrowser

// Delegates are audited through a real window because the views only lay out
// and the DListView frame sweep only runs with a render loop behind them.
ShellRoot {
    id: fixture

    readonly property var log: Log.scoped("fixture")
    property int failures: 0
    property int step: 0
    property var defects: []

    function check(label, actual, expected) {
        const ok = JSON.stringify(actual) === JSON.stringify(expected);
        if (!ok) {
            failures++;
            log.error(`FAIL ${label}: got ${JSON.stringify(actual)} want ${JSON.stringify(expected)}`);
            return;
        }
        log.info(`ok ${label}`);
    }

    function entry(i) {
        return {
            "name": "file-" + i + ".txt",
            "displayName": "",
            "path": "/home/me/file-" + i + ".txt",
            "isDir": i % 7 === 0,
            "untrusted": false,
            "isSymlink": false,
            "symlinkTarget": "",
            "symlinkBroken": false,
            "size": 12,
            "mtimeMs": 1000,
            "ctimeMs": 1000,
            "mode": "-rw-r--r--",
            "owner": "me",
            "group": "me",
            "hidden": false,
            "isExecutable": false,
            "extension": "txt",
            "mime": "text/plain",
            "iconName": "text-plain",
            "thumbnail": "",
            "thumbnailable": false,
            "unreadable": false
        };
    }

    readonly property ListModel entries: ListModel {}
    readonly property SelectionModel selection: SelectionModel {}

    readonly property Window host: Window {
        width: 1200
        height: 400
        visible: true

        FileListView {
            id: list

            width: 600
            height: parent.height
            selection: fixture.selection
            model: fixture.entries
        }

        FileGridView {
            id: grid

            x: 600
            width: 600
            height: parent.height
            selection: fixture.selection
            model: fixture.entries
        }
    }

    function claimedInViewport(view) {
        const kids = view.contentItem.children;
        const claimed = [];
        for (let k = 0; k < kids.length; k++) {
            const c = kids[k];
            if (!c || c.index === undefined || view.itemAtIndex(c.index) !== c)
                continue;
            if (c.y + c.height > view.contentY && c.y < view.contentY + view.height)
                claimed.push(c);
        }
        return claimed.sort((a, b) => a.index - b.index);
    }

    function nextCell(view, prev) {
        if (view === list)
            return Qt.point(0, prev.y + view.rowPitch);
        if ((prev.index + 1) % view.columnsPerRow === 0)
            return Qt.point(0, prev.y + view.cellHeight);
        return Qt.point(prev.x + view.cellWidth, prev.y);
    }

    function audit(view, name, geometry, settled) {
        const rows = claimedInViewport(view);
        if (rows.length === 0) {
            defects.push(`${name} step ${step}: nothing laid out in the viewport`);
            return;
        }
        for (const c of rows) {
            if (!c.visible)
                defects.push(`${name} step ${step}: delegate ${c.index} hidden`);
            if (settled && c.opacity < 0.99)
                defects.push(`${name} step ${step}: delegate ${c.index} stuck at opacity ${c.opacity}`);
        }
        if (!geometry)
            return;
        const first = rows[0];
        const last = rows[rows.length - 1];
        if (first.y > view.contentY + 0.5 && first.index > 0)
            defects.push(`${name} step ${step}: empty space above delegate ${first.index}`);
        if (last.y + last.height < view.contentY + view.height - 0.5 && last.index < view.count - 1)
            defects.push(`${name} step ${step}: empty space below delegate ${last.index}`);
        for (let i = 1; i < rows.length; i++) {
            const want = nextCell(view, rows[i - 1]);
            if (rows[i].index !== rows[i - 1].index + 1)
                defects.push(`${name} step ${step}: delegate ${rows[i - 1].index} is followed by ${rows[i].index}`);
            else if (Math.abs(rows[i].x - want.x) > 0.5 || Math.abs(rows[i].y - want.y) > 0.5)
                defects.push(`${name} step ${step}: gap between delegates ${rows[i - 1].index} and ${rows[i].index}`);
        }
        if (settled)
            check(`${name} visibleRange matches the laid out rows`, view.visibleRange(), [first.index, last.index]);
    }

    function scroll(view, delta) {
        view.contentY = Math.max(view.originY, Math.min(view.maximumContentY, view.contentY + delta));
    }

    function rowAtTop(view) {
        return Math.max(0, Math.floor((view.contentY - view.contentOrigin) / view.rowPitch));
    }

    // Each frame audits what the previous frame laid out, then mutates.
    readonly property FrameAnimation driver: FrameAnimation {
        running: true
        onTriggered: {
            const s = fixture.step;
            if (s > 0 && s < 34) {
                const settling = s > 4 && s < 20;
                fixture.audit(list, "list", !settling, false);
                fixture.audit(grid, "grid", !settling, false);
            }
            switch (true) {
            case s === 0:
                for (let i = 0; i < 300; i++)
                    fixture.entries.append(fixture.entry(i));
                break;
            case s < 4:
                fixture.scroll(list, 300);
                fixture.scroll(grid, 300);
                break;
            case s === 4:
                fixture.entries.remove(0, 4);
                fixture.entries.remove(fixture.rowAtTop(list) + 2, 3);
                fixture.entries.insert(fixture.rowAtTop(list) + 1, fixture.entry(1000));
                break;
            case s < 20:
                fixture.scroll(list, s % 2 ? 250 : -170);
                fixture.scroll(grid, s % 2 ? 250 : -170);
                break;
            case s < 34:
                fixture.scroll(list, s % 3 ? -400 : 650);
                fixture.scroll(grid, s % 3 ? -400 : 650);
                break;
            case s < 42:
                fixture.step++;
                return;
            case s === 42:
                fixture.audit(list, "list", true, true);
                fixture.audit(grid, "grid", true, true);
                fixture.step++;
                return;
            default:
                running = false;
                fixture.check("no delegate goes missing, hidden, invisible or misplaced while scrolling", fixture.defects, []);
                if (fixture.failures === 0)
                    fixture.log.info("FIXTURE PASS");
                Qt.callLater(Qt.quit);
                return;
            }
            fixture.step++;
        }
    }
}
