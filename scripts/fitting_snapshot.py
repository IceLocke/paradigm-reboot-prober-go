#!/usr/bin/env python3
"""Extract only fitting inputs from a local pg_dump archive (never restore SQL).

Requires pg_restore and openpyxl. Output contains anonymized player IDs and must
remain local. Votes are matched by exact title + difficulty; ambiguous, CTC,
and unmatched entries are reported rather than guessed.
"""
import argparse
import json
import re
import subprocess

import openpyxl


def unescape(value):
    if value == r"\N":
        return None
    escapes = {"t": "\t", "n": "\n", "r": "\r", "b": "\b", "f": "\f", "v": "\v", "\\": "\\"}
    return re.sub(r"\\([0-7]{1,3}|x[0-9a-fA-F]{1,2}|.)", lambda m:
                  chr(int(m[1], 16)) if m[1].startswith("x") else
                  chr(int(m[1], 8)) if m[1][0] in "01234567" else
                  escapes.get(m[1], m[1]), value)


def rows(dump, table):
    process = subprocess.Popen(["pg_restore", "--data-only", "--file=-", "--table=" + table, dump],
                               stdout=subprocess.PIPE, text=True)
    columns = None
    try:
        for line in process.stdout:
            if line.startswith("COPY public."):
                columns = re.match(r"COPY public\.\w+ \((.*?)\)", line)[1].split(", ")
            elif line.startswith(r"\."):
                columns = None
            elif columns:
                row = dict(zip(columns, map(unescape, line.rstrip("\n").split("\t"))))
                if row.get("deleted_at") is None:
                    yield row
    finally:
        process.stdout.close()
        if process.wait() != 0:
            raise RuntimeError("pg_restore failed for " + table)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dump", required=True)
    parser.add_argument("--votes", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    songs = {s["id"]: s for s in rows(args.dump, "songs")}
    charts = {c["id"]: dict(id=int(c["id"]), song_id=int(c["song_id"]),
              level=float(c["level"]), title=c["override_title"] or songs[c["song_id"]]["title"],
              difficulty=c["difficulty"], stored=float(c["fitting_level"]) if c["fitting_level"] else None)
              for c in rows(args.dump, "charts") if c["song_id"] in songs}
    best = {b["play_record_id"]: b for b in rows(args.dump, "best_play_records") if b["chart_id"] in charts}
    users, records = {}, []
    for r in rows(args.dump, "play_records"):
        b = best.get(r["id"])
        if b is None or r["chart_id"] != b["chart_id"] or r["username"] != b["username"]:
            continue
        records.append(dict(user=users.setdefault(r["username"], len(users)),
                            chart=int(r["chart_id"]), score=int(r["score"]), rating=int(r["rating"])))
    votes, unmatched = [], []
    workbook = openpyxl.load_workbook(args.votes, data_only=True, read_only=True)
    for row in list(workbook.active.values)[1:]:
        title, vote, count = row[:3]
        if not isinstance(title, str) or not isinstance(vote, (int, float)):
            continue
        difficulty = "reboot" if title.endswith(" [RBT]") else "massive"
        normalized = title.removesuffix(" [RBT]").removesuffix(" [MSV]")
        matches = [c for c in charts.values() if c["title"] == normalized and c["difficulty"] == difficulty]
        if len(matches) == 1:
            votes.append(dict(chart=matches[0]["id"], level=vote, count=count))
        else:
            unmatched.append(title)
    workbook.close()
    with open(args.output, "w") as f:
        json.dump(dict(charts=sorted(charts.values(), key=lambda c: c["id"]),
                       records=records, votes=votes), f, ensure_ascii=False)
    print(json.dumps(dict(charts=len(charts), records=len(records), players=len(users),
                          votes=len(votes), unmatched=unmatched), ensure_ascii=False))


if __name__ == "__main__":
    main()
