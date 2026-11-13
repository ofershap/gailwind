#!/usr/bin/env python3
"""Parity harness: compare gailwind output vs real tailwindcss v3 output.

Both stylesheets are flattened to (media-context, selector) -> sorted
declarations, then compared rule by rule. Prints measured coverage and the
lists of mismatched, missing and extra rules. Exit 1 below threshold.

Usage: parity.py REF.css GW.css [min_coverage_pct]
"""
import re
import sys


def flatten(css):
    """Return {(media_tuple, selector): sorted [decl strings]}"""
    out = {}
    # tokenize braces
    stack = []
    i = 0
    n = len(css)
    while i < n:
        j = css.find("{", i)
        if j == -1:
            break
        head = css[i:j].strip()
        # find matching close
        depth = 1
        k = j + 1
        while k < n and depth:
            if css[k] == "{":
                depth += 1
            elif css[k] == "}":
                depth -= 1
            k += 1
        body = css[j + 1 : k - 1]
        if head.startswith("@media") or head.startswith("@supports") or head.startswith("@keyframes"):
            stack.append(head)
            inner = flatten(body)
            for (media, sel), decls in inner.items():
                out[(tuple(stack) + media, sel)] = decls
            stack.pop()
        else:
            decls = sorted(
                d.strip() for d in body.split(";") if d.strip() and ":" in d
            )
            out[(tuple(stack), head)] = decls
        i = k
    return out


def norm(decls):
    return [re.sub(r"\s+", " ", d).strip().lower() for d in decls]


def main():
    ref = flatten(open(sys.argv[1]).read())
    gw = flatten(open(sys.argv[2]).read())
    threshold = float(sys.argv[3]) if len(sys.argv) > 3 else 95.0

    match, mismatch, missing = [], [], []
    for key, rdecls in ref.items():
        if key not in gw:
            missing.append((key, rdecls))
        elif norm(gw[key]) != norm(rdecls):
            mismatch.append((key, rdecls, gw[key]))
        else:
            match.append(key)
    extra = [k for k in gw if k not in ref]

    total = len(ref)
    pct = 100.0 * len(match) / total if total else 0.0
    print(f"ref rules: {total}  gailwind rules: {len(gw)}")
    print(
        f"match: {len(match)} ({pct:.1f}%)  mismatch: {len(mismatch)}  "
        f"missing: {len(missing)}  extra: {len(extra)}"
    )
    for label, rows in (("MISMATCH", mismatch), ("MISSING", missing), ("EXTRA", [(e, None) for e in extra])):
        if rows:
            print(f"\n== {label} ({len(rows)}) ==")
            for row in rows[:200]:
                print(" ", row[0][1], "@", " ".join(row[0][0]) or "(root)")
                if label == "MISMATCH":
                    print("   ref:", row[1])
                    print("   gw: ", row[2])
    sys.exit(0 if pct >= threshold else 1)


if __name__ == "__main__":
    main()
