.pragma library
.import "../DCommon/Common/fzf.js" as Fzf
.import "../DCommon/FileBrowser/PathCompletion.js" as PathCompletion

const hour = 3600000;
const day = 24 * hour;
const week = 7 * day;
const maxEntries = 1000;
const maxTotalVisits = 10000;
const nameTier = 1000000;
const exactNameBonus = 1000;
const frecencyScale = 16;
const frecencyCap = 96;
const scoreMatch = 16;
const bonusBoundary = 8;
const bonusCamel = 7;
const bonusConsecutive = 4;
const penaltyGapStart = 3;
const penaltyGapExtension = 1;
const boundaries = "/-_. ";

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
        const display = PathCompletion.compact(entry.path, home).normalize();
        const parts = PathCompletion.split(display);
        const name = display.substring(parts.start, parts.end);
        return {
            "entry": entry,
            "display": display,
            "lower": display.toLowerCase(),
            "name": name,
            "nameLower": name.toLowerCase(),
            "nameStart": parts.start
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

function compile(query) {
    return Fzf.parseTerms(true, "smart-case", false, query).map(set => {
        const terms = set.map(term => ({
                    "type": term.typ,
                    "inverse": term.inv,
                    "caseSensitive": term.caseSensitive,
                    "text": String.fromCodePoint(...term.text)
                }));
        return {
            "terms": terms,
            "excludes": terms.every(term => term.inverse),
            "anchored": terms.some(term => !term.inverse && (term.type === Fzf.TermType.Prefix || term.type === Fzf.TermType.Equal))
        };
    });
}

function rate(original, positions) {
    let total = 0;
    let previous = -2;
    positions.forEach((at, index) => {
        const before = at > 0 ? original[at - 1] : "/";
        const boundary = boundaries.includes(before);
        const camel = !boundary && before === before.toLowerCase() && original[at] !== original[at].toLowerCase();
        total += scoreMatch + (boundary ? bonusBoundary * (index === 0 ? 2 : 1) : camel ? bonusCamel : 0);
        if (at === previous + 1)
            total += bonusConsecutive;
        else if (index > 0)
            total -= penaltyGapStart + penaltyGapExtension * (at - previous - 2);
        previous = at;
    });
    return total;
}

function run(start, length) {
    const positions = [];
    for (let index = 0; index < length; index++)
        positions.push(start + index);
    return positions;
}

function locate(text, term) {
    const needle = term.text;
    switch (term.type) {
    case Fzf.TermType.Exact:
        {
            const at = text.indexOf(needle);
            return at < 0 ? null : run(at, needle.length);
        }
    case Fzf.TermType.Prefix:
        return text.startsWith(needle) ? run(0, needle.length) : null;
    case Fzf.TermType.Suffix:
        return text.endsWith(needle) ? run(text.length - needle.length, needle.length) : null;
    case Fzf.TermType.Equal:
        return text === needle ? run(0, needle.length) : null;
    }
    const positions = [];
    let from = 0;
    for (let index = 0; index < needle.length; index++) {
        const at = text.indexOf(needle[index], from);
        if (at < 0)
            return null;
        positions.push(at);
        from = at + 1;
    }
    for (let index = positions.length - 2; index >= 0; index--)
        positions[index] = text.lastIndexOf(needle[index], positions[index + 1] - 1);
    return positions;
}

function hitIn(original, lower, terms) {
    for (const term of terms) {
        const positions = locate(term.caseSensitive ? original : lower, term);
        if (term.inverse && !positions)
            return {
                "score": 0,
                "positions": []
            };
        if (!term.inverse && positions)
            return {
                "score": rate(original, positions),
                "positions": positions
            };
    }
    return null;
}

function score(item, sets) {
    let nameTerms = 0;
    let quality = 0;
    const positions = [];
    for (const set of sets) {
        const inPath = hitIn(item.display, item.lower, set.terms);
        if (set.excludes) {
            if (!inPath)
                return null;
            continue;
        }
        if (!inPath && !set.anchored)
            return null;
        const inName = hitIn(item.name, item.nameLower, set.terms);
        if (inName) {
            nameTerms++;
            quality += inName.score;
            inName.positions.forEach(at => positions.push(at + item.nameStart));
            continue;
        }
        if (!inPath)
            return null;
        quality += inPath.score;
        inPath.positions.forEach(at => positions.push(at));
    }
    return {
        "nameTerms": nameTerms,
        "quality": quality,
        "positions": positions
    };
}

function match(prepared, query, now, limit, exclude) {
    const trimmed = query.trim().normalize();
    const sets = trimmed === "" ? [] : compile(trimmed);
    if (sets.length === 0)
        return top(prepared, now, limit, exclude);
    const exact = trimmed.toLowerCase();
    const scored = [];
    for (const item of prepared) {
        if (item.entry.path === exclude)
            continue;
        const hit = score(item, sets);
        if (!hit)
            continue;
        const bonus = item.nameLower === exact ? exactNameBonus : 0;
        const bias = Math.min(frecencyCap, Math.log1p(frecency(item.entry, now)) * frecencyScale);
        scored.push({
            "item": item,
            "positions": hit.positions,
            "rank": hit.nameTerms * nameTier + bonus + hit.quality + bias
        });
    }
    scored.sort((a, b) => b.rank - a.rank || b.item.entry.last - a.item.entry.last);
    return scored.slice(0, limit).map(hit => ({
                "path": hit.item.entry.path,
                "display": hit.item.display,
                "positions": hit.positions
            }));
}
