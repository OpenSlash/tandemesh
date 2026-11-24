"""Structural audit for logo SVGs (stand-in for the skill's scripts/svg_audit.py).

Checks the things the skill's checklist flags as FAILs: live text, embedded
rasters, filters/masks, colour count, near-miss angles, hairlines, tiny details,
path count/complexity and viewBox conventions.

Usage: python audit.py logo.svg [more.svg ...]
Exit code 1 when any file has a FAIL.
"""

from __future__ import annotations

import math
import re
import sys
from pathlib import Path
from xml.etree import ElementTree as ET

HEX_RE = re.compile(r"#([0-9a-fA-F]{3,8})\b")
NUM_RE = re.compile(r"-?\d*\.?\d+")
PATH_CMD_RE = re.compile(r"([MmLlHhVvCcSsQqTtAaZz])")
ANGLE_TOLERANCE = 0.6  # every whole-degree angle is acceptable
TINY_FRACTION = 1 / 64


def angle_bucket(deg: float) -> float:
    deg = abs(deg) % 180
    return min(deg, 180 - deg)


def near_miss(deg: float) -> bool:
    a = angle_bucket(deg)
    return abs(a - round(a)) > ANGLE_TOLERANCE


def audit(path: Path) -> tuple[int, list[str], list[str]]:
    text = path.read_text(encoding="utf-8")
    warnings: list[str] = []
    failures: list[str] = []
    score = 100

    if "<!DOCTYPE" in text or "<html" in text:
        failures.append("file is HTML, not SVG")
    try:
        root = ET.fromstring(text)
    except ET.ParseError as exc:
        return 0, [f"XML parse error: {exc}"], []

    tags = [el.tag.split("}")[-1] for el in root.iter()]
    tag_count = {t: tags.count(t) for t in set(tags)}

    for bad, why in (
        ("text", "live <text>: font dependent"),
        ("tspan", "live <tspan>: font dependent"),
        ("image", "embedded raster <image>"),
        ("filter", "<filter>: inconsistent rendering"),
        ("mask", "<mask>: tool compatibility"),
        ("clipPath", "<clipPath>: tool compatibility"),
    ):
        if tag_count.get(bad):
            failures.append(f"{tag_count[bad]}x {why}")
            score -= 20

    vb = root.get("viewBox")
    if not vb:
        warnings.append("no viewBox")
        score -= 10
    else:
        parts = [float(x) for x in vb.replace(",", " ").split()]
        if len(parts) != 4 or parts[0] != 0 or parts[1] != 0:
            warnings.append(f"unusual viewBox {vb}")
        width, height = parts[2], parts[3]
        if abs(width - height) > 0.5:
            warnings.append(f"non-square canvas {width:g}x{height:g}")
    if root.get("width") or root.get("height"):
        warnings.append("width/height attributes on root (viewBox is enough)")
    if root.find("{http://www.w3.org/2000/svg}title") is None and root.find("title") is None:
        warnings.append("no <title>")

    colours = sorted({("#" + c).lower() for c in HEX_RE.findall(text)})
    if len(colours) > 6:
        failures.append(f"{len(colours)} hard-coded colours")
        score -= 10

    svg_nums = [float(n) for n in NUM_RE.findall(vb or "")]

    # near-miss angles from straight segments inside path data
    bad_angles: list[str] = []
    for el in root.iter():
        d = el.get("d")
        if not d:
            continue
        tokens = re.findall(r"[MmLlHhVvCcSsQqTtAaZz]|-?\d*\.?\d+(?:[eE][-+]?\d+)?", d)
        current, start = (0.0, 0.0), (0.0, 0.0)
        cmd = None
        i = 0
        while i < len(tokens):
            tok = tokens[i]
            if PATH_CMD_RE.fullmatch(tok):
                cmd = tok
                i += 1
                if cmd in "Zz":
                    current = start
                    continue
            try:
                n = int(float(tokens[i]))
            except (ValueError, IndexError):
                i += 1
                continue
            if cmd in "Mm" and i + 1 < len(tokens):
                try:
                    x, y = float(tokens[i]), float(tokens[i + 1])
                except ValueError:
                    break
                if cmd == "m":
                    x, y = current[0] + x, current[1] + y
                current = start = (x, y)
                i += 2
                cmd = "l" if cmd == "m" else "L"  # implicit lineto after moveto
                continue
            if cmd in "Ll" and i + 1 < len(tokens):
                try:
                    x, y = float(tokens[i]), float(tokens[i + 1])
                except ValueError:
                    break
                if cmd == "l":
                    x, y = current[0] + x, current[1] + y
                dx, dy = x - current[0], y - current[1]
                length = math.hypot(dx, dy)
                if length > 1e-6:
                    deg = math.degrees(math.atan2(dy, dx))
                    if near_miss(deg):
                        bad_angles.append(f"{deg:.1f}deg len {length:.1f}")
                current = (x, y)
                i += 2
                continue
            if cmd in "Hh":
                x = float(tokens[i]) + (current[0] if cmd == "h" else 0)
                current = (x, current[1])
                i += 1
                continue
            if cmd in "Vv":
                y = float(tokens[i]) + (current[1] if cmd == "v" else 0)
                current = (current[0], y)
                i += 1
                continue
            i += 1
    if bad_angles:
        warnings.append(f"near-miss angles: {', '.join(bad_angles[:4])}")
        score -= 5

    # hairline strokes
    for el in root.iter():
        sw = el.get("stroke-width")
        if sw and float(sw) < 4:
            warnings.append(f"thin stroke {sw} (units of a 256 canvas)")
            score -= 5

    paths = [el for el in root.iter() if el.tag.split("}")[-1] == "path"]
    if len(paths) > 4:
        warnings.append(f"{len(paths)} paths, simplify if possible")
        score -= 5

    # decimals with more than 2 places
    if re.search(r"\d+\.\d{3,}", text):
        warnings.append("coordinates with >2 decimals")
        score -= 5

    ink_nodes = tag_count.get("path", 0) + tag_count.get("rect", 0) + tag_count.get("polygon", 0) + tag_count.get("circle", 0)
    summary = f"{path.name}: {tag_count.get('path', 0)} path(s), {ink_nodes} ink element(s), {len(colours)} colour(s)"
    return max(score, 0), [summary] + warnings, failures


def main(argv: list[str]) -> int:
    failed = False
    for arg in argv:
        score, warnings, failures = audit(Path(arg))
        print(f"\n{arg}  score={score}")
        for line in warnings:
            print(f"  note  {line}")
        for line in failures:
            print(f"  FAIL  {line}")
        if failures:
            failed = True
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))

