.pragma library
.import "../DCommon/FileBrowser/PathCompletion.js" as PathCompletion

const hour = 3600000;
const day = 24 * hour;
const week = 7 * day;
const maxEntries = 1000;
const maxTotalVisits = 10000;

function frecency(entry, now) {
    const age = now - entry.last;
    if (age < hour)
        return entry.visits * 4;
    if (age < day)
        return entry.visits * 2;
    if (age < week)
        return entry.visits * 0.5;
    return entry.visits * 0.25;
}

function byFrecency(now) {
    return (a, b) => frecency(b, now) - frecency(a, now) || b.last - a.last;
}

function aged(entries) {
    const total = entries.reduce((sum, entry) => sum + entry.visits, 0);
    if (total <= maxTotalVisits)
        return entries;
    const factor = 0.9 * maxTotalVisits / total;
    return entries.map(entry => ({
                "path": entry.path,
                "visits": Math.round(entry.visits * factor * 100) / 100,
                "last": entry.last
            })).filter(entry => entry.visits >= 1);
}

function capped(entries, now) {
    if (entries.length <= maxEntries)
        return entries;
    return entries.slice().sort(byFrecency(now)).slice(0, maxEntries);
}

function visit(entries, path, now) {
    const base = aged(entries);
    const found = base.some(entry => entry.path === path);
    const next = found ? base.map(entry => entry.path !== path ? entry : {
            "path": path,
            "visits": entry.visits + 1,
            "last": now
        }) : base.concat([
        {
            "path": path,
            "visits": 1,
            "last": now
        }
    ]);
    return capped(next, now);
}

function forget(entries, paths) {
    return entries.filter(entry => !paths.includes(entry.path));
}

function seed(paths, now) {
    return paths.filter(path => typeof path === "string" && path.startsWith("/")).slice(0, maxEntries).map((path, index) => ({
                "path": path,
                "visits": 1,
                "last": now - index * 1000
            }));
}

function parse(text) {
    let data = null;
    try {
        data = JSON.parse(text);
    } catch (e) {
        return null;
    }
    if (!Array.isArray(data?.folders))
        return null;
    return data.folders.filter(entry => typeof entry?.path === "string" && entry.path !== "" && entry.visits > 0 && Number.isFinite(entry.last)).map(entry => ({
                "path": entry.path,
                "visits": entry.visits,
                "last": entry.last
            }));
}

function storedPath(path, mounts) {
    for (const mount of mounts) {
        if (!mount.path || !mount.uri)
            continue;
        if (path !== mount.path && !path.startsWith(mount.path + "/"))
            continue;
        return mount.uri.replace(/\/+$/, "") + (path.substring(mount.path.length) || "/");
    }
    return path;
}

function prepare(entries, home) {
    return entries.map(entry => {
        return {
            "entry": entry,
            "display": PathCompletion.compact(entry.path, home).normalize()
        };
    });
}

function top(prepared, now, limit, exclude) {
    const order = byFrecency(now);
    return prepared.filter(item => item.entry.path !== exclude).sort((a, b) => order(a.entry, b.entry)).slice(0, limit).map(item => ({
                "path": item.entry.path,
                "display": item.display,
                "positions": []
            }));
}
