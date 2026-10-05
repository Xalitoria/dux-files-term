pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Services
import qs.DCommon.FileBrowser

ShellRoot {
    id: fixture

    readonly property var log: Log.scoped("fixture")
    property int failures: 0

    function check(label, actual, expected) {
        const ok = JSON.stringify(actual) === JSON.stringify(expected);
        if (!ok) {
            failures++;
            log.error(`FAIL ${label}: got ${JSON.stringify(actual)} want ${JSON.stringify(expected)}`);
            return;
        }
        log.info(`ok ${label}`);
    }

    function fill(names) {
        entries.clear();
        for (const name of names)
            entries.append({
                "path": `/home/me/${name}`,
                "name": name
            });
    }

    readonly property ListModel entries: ListModel {}

    readonly property SelectionModel selection: SelectionModel {}

    Component.onCompleted: {
        fill(["a", "b", "c", "d", "e"]);

        selection.select("/home/me/b");
        check("select replaces the selection", selection.paths, ["/home/me/b"]);
        check("select moves the cursor and the anchor", [selection.cursorPath, selection.anchorPath], ["/home/me/b", "/home/me/b"]);
        check("contains answers for the selected path", [selection.contains("/home/me/b"), selection.contains("/home/me/c")], [true, false]);

        selection.toggle("/home/me/d");
        check("toggle adds", selection.paths, ["/home/me/b", "/home/me/d"]);
        selection.toggle("/home/me/d");
        check("toggle removes", selection.paths, ["/home/me/b"]);

        selection.select("/home/me/b");
        selection.extendTo(entries, 3);
        check("shift extends from the anchor", selection.paths, ["/home/me/b", "/home/me/c", "/home/me/d"]);
        check("extending keeps the anchor and moves the cursor", [selection.anchorPath, selection.cursorPath], ["/home/me/b", "/home/me/d"]);

        selection.extendTo(entries, 0);
        check("extending back over the anchor replaces the range", selection.paths, ["/home/me/a", "/home/me/b"]);

        selection.select("/home/me/a");
        selection.toggle("/home/me/e");
        selection.extendTo(entries, 2);
        check("extending keeps what was ctrl-picked before the anchor", selection.paths.slice().sort(), ["/home/me/a", "/home/me/c", "/home/me/d", "/home/me/e"]);

        selection.clear();
        selection.selectAll(entries);
        check("select all takes every row", selection.count, 5);

        selection.select("/home/me/a");
        selection.invert(entries);
        check("invert keeps everything else", selection.paths, ["/home/me/b", "/home/me/c", "/home/me/d", "/home/me/e"]);

        // A watch batch removing rows must drop them from the selection without
        // disturbing the rest, which is why the model stores paths not indices.
        selection.paths = ["/home/me/b", "/home/me/c"];
        selection.cursorPath = "/home/me/c";
        selection.anchorPath = "/home/me/c";
        fill(["a", "b", "d", "e"]);
        selection.prune(entries);
        check("prune drops removed paths", selection.paths, ["/home/me/b"]);
        check("prune clears a cursor that is gone", selection.cursorPath, "");
        check("prune clears an anchor that is gone", selection.anchorPath, "");

        selection.paths = ["/home/me/b"];
        fill(["e", "d", "b", "a"]);
        selection.prune(entries);
        check("a re-sort keeps the selection", selection.paths, ["/home/me/b"]);
        check("indexOf follows the new order", selection.indexOf(entries, "/home/me/b"), 2);

        selection.select("/home/me/b");
        selection.keyboardCursor = true;
        check("no ring when the cursor is the one selected row", selection.showsCursor("/home/me/b"), false);
        selection.toggle("/home/me/d");
        check("ring once the selection is more than the cursor", selection.showsCursor("/home/me/d"), true);
        check("ring only on the cursor row", selection.showsCursor("/home/me/b"), false);
        selection.keyboardCursor = false;
        check("no ring after a pointer toggle", selection.showsCursor("/home/me/d"), false);

        selection.clear();
        check("clear empties everything", [selection.count, selection.cursorPath, selection.anchorPath, selection.keyboardCursor], [0, "", "", false]);

        if (failures === 0)
            log.info("FIXTURE PASS");
        Qt.callLater(Qt.quit);
    }
}
