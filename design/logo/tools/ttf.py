"""Minimal TrueType outline reader, so the wordmark can ship as paths.

fontTools is not installed in this runtime and the network is closed, but a
logo must not depend on a font being installed. This module reads just enough of
a TrueType file (cmap 4/12, loca, glyf, hmtx) to turn a string into SVG path
data, including composite glyphs.

Only simple quadratic outlines are produced (M/L/Q/Z), which is exactly what
TrueType stores, so nothing is approximated.
"""

from __future__ import annotations

import re
import struct
from dataclasses import dataclass
from pathlib import Path

ARG_1_AND_2_ARE_WORDS = 0x0001
ARGS_ARE_XY_VALUES = 0x0002
WE_HAVE_A_SCALE = 0x0008
MORE_COMPONENTS = 0x0020
WE_HAVE_AN_X_AND_Y_SCALE = 0x0040
WE_HAVE_A_TWO_BY_TWO = 0x0080


@dataclass
class Point:
    x: float
    y: float
    on: bool


class TTF:
    def __init__(self, path: str | Path):
        self.data = Path(path).read_bytes()
        self.tables = self._read_table_directory()
        head = self._table("head")
        self.units_per_em = struct.unpack_from(">H", self.data, head + 18)[0]
        self.index_to_loc_format = struct.unpack_from(">h", self.data, head + 50)[0]
        maxp = self._table("maxp")
        self.num_glyphs = struct.unpack_from(">H", self.data, maxp + 4)[0]
        hhea = self._table("hhea")
        self.number_of_hmetrics = struct.unpack_from(">H", self.data, hhea + 34)[0]
        self.ascender = struct.unpack_from(">h", self.data, hhea + 4)[0]
        self.descender = struct.unpack_from(">h", self.data, hhea + 6)[0]
        self._loca = self._read_loca()
        self.cmap = self._read_cmap()
        self._advances = self._read_hmtx()
        self._glyph_cache: dict[int, list[list[Point]]] = {}

    # -- tables ----------------------------------------------------------- #
    def _read_table_directory(self) -> dict[str, int]:
        num_tables = struct.unpack_from(">H", self.data, 4)[0]
        tables = {}
        for i in range(num_tables):
            off = 12 + i * 16
            tag = self.data[off:off + 4].decode("latin-1")
            tables[tag] = struct.unpack_from(">I", self.data, off + 8)[0]
        return tables

    def _table(self, tag: str) -> int:
        if tag not in self.tables:
            raise KeyError(f"missing table {tag}")
        return self.tables[tag]

    def _read_loca(self) -> list[int]:
        base = self._table("loca")
        if self.index_to_loc_format == 0:
            return [struct.unpack_from(">H", self.data, base + 2 * i)[0] * 2 for i in range(self.num_glyphs + 1)]
        return [struct.unpack_from(">I", self.data, base + 4 * i)[0] for i in range(self.num_glyphs + 1)]

    def _read_hmtx(self) -> list[int]:
        base = self._table("hmtx")
        advances = []
        last = 0
        for i in range(self.num_glyphs):
            if i < self.number_of_hmetrics:
                last = struct.unpack_from(">H", self.data, base + 4 * i)[0]
            advances.append(last)
        return advances

    def _read_cmap(self) -> dict[int, int]:
        base = self._table("cmap")
        num = struct.unpack_from(">H", self.data, base + 2)[0]
        best = None
        for i in range(num):
            platform, encoding, offset = struct.unpack_from(">HHI", self.data, base + 4 + 8 * i)
            fmt = struct.unpack_from(">H", self.data, base + offset)[0]
            score = 0
            if (platform, encoding) in ((3, 10), (0, 4), (0, 6)) and fmt == 12:
                score = 3
            elif (platform, encoding) in ((3, 1), (0, 3)) and fmt == 4:
                score = 2
            if best is None or score > best[0]:
                best = (score, base + offset, fmt)
        if best is None:
            raise ValueError("no usable cmap")
        _, offset, fmt = best
        return self._parse_cmap_subtable(offset, fmt)

    def _parse_cmap_subtable(self, offset: int, fmt: int) -> dict[int, int]:
        out: dict[int, int] = {}
        if fmt == 4:
            seg_x2 = struct.unpack_from(">H", self.data, offset + 6)[0]
            seg = seg_x2 // 2
            end_base = offset + 14
            start_base = end_base + seg_x2 + 2
            delta_base = start_base + seg_x2
            range_base = delta_base + seg_x2
            for s in range(seg):
                end = struct.unpack_from(">H", self.data, end_base + 2 * s)[0]
                start = struct.unpack_from(">H", self.data, start_base + 2 * s)[0]
                delta = struct.unpack_from(">h", self.data, delta_base + 2 * s)[0]
                range_off = struct.unpack_from(">H", self.data, range_base + 2 * s)[0]
                if start > end:
                    continue
                for c in range(start, min(end, 0xFFFF) + 1):
                    if range_off == 0:
                        gid = (c + delta) & 0xFFFF
                    else:
                        addr = range_base + 2 * s + range_off + 2 * (c - start)
                        if addr + 2 > len(self.data):
                            continue
                        gid = struct.unpack_from(">H", self.data, addr)[0]
                        if gid:
                            gid = (gid + delta) & 0xFFFF
                    if gid:
                        out[c] = gid
        elif fmt == 12:
            groups = struct.unpack_from(">I", self.data, offset + 12)[0]
            for g in range(groups):
                start, end, start_gid = struct.unpack_from(">III", self.data, offset + 16 + 12 * g)
                for c in range(start, end + 1):
                    out[c] = start_gid + (c - start)
        else:
            raise ValueError(f"unsupported cmap format {fmt}")
        return out

    # -- glyphs ------------------------------------------------------------ #
    def glyph_id(self, char: str) -> int:
        return self.cmap.get(ord(char), 0)

    def advance(self, char: str) -> int:
        return self._advances[self.glyph_id(char)]

    def contours(self, gid: int) -> list[list[Point]]:
        if gid in self._glyph_cache:
            return self._glyph_cache[gid]
        glyf = self._table("glyf")
        start, end = self._loca[gid], self._loca[gid + 1]
        if end <= start:
            self._glyph_cache[gid] = []
            return []
        result = self._parse_glyph(glyf + start, glyf)
        self._glyph_cache[gid] = result
        return result

    def _parse_glyph(self, offset: int, glyf: int, depth: int = 0) -> list[list[Point]]:
        if depth > 5:
            return []
        num_contours = struct.unpack_from(">h", self.data, offset)[0]
        if num_contours >= 0:
            return self._parse_simple(offset, num_contours)
        return self._parse_composite(offset, glyf, depth)

    def _parse_simple(self, offset: int, num_contours: int) -> list[list[Point]]:
        pos = offset + 10
        end_pts = list(struct.unpack_from(f">{num_contours}H", self.data, pos))
        pos += 2 * num_contours
        instruction_len = struct.unpack_from(">H", self.data, pos)[0]
        pos += 2 + instruction_len
        num_points = end_pts[-1] + 1 if end_pts else 0

        flags = []
        while len(flags) < num_points:
            flag = self.data[pos]
            pos += 1
            flags.append(flag)
            if flag & 0x08:  # repeat
                repeat = self.data[pos]
                pos += 1
                flags.extend([flag] * repeat)

        xs = []
        x = 0
        for flag in flags:
            if flag & 0x02:  # x short
                dx = self.data[pos]
                pos += 1
                x += dx if flag & 0x10 else -dx
            elif not flag & 0x10:  # not same x
                x += struct.unpack_from(">h", self.data, pos)[0]
                pos += 2
            xs.append(x)

        ys = []
        y = 0
        for flag in flags:
            if flag & 0x04:  # y short
                dy = self.data[pos]
                pos += 1
                y += dy if flag & 0x20 else -dy
            elif not flag & 0x20:  # not same y
                y += struct.unpack_from(">h", self.data, pos)[0]
                pos += 2
            ys.append(y)

        contours, start = [], 0
        for end in end_pts:
            pts = [Point(xs[i], ys[i], bool(flags[i] & 0x01)) for i in range(start, end + 1)]
            if pts:
                contours.append(pts)
            start = end + 1
        return contours

    def _parse_composite(self, offset: int, glyf: int, depth: int) -> list[list[Point]]:
        pos = offset + 10
        contours: list[list[Point]] = []
        while True:
            flags, gid = struct.unpack_from(">HH", self.data, pos)
            pos += 4
            if flags & ARG_1_AND_2_ARE_WORDS:
                a1, a2 = struct.unpack_from(">hh", self.data, pos)
                pos += 4
            else:
                a1 = struct.unpack_from(">b", self.data, pos)[0]
                a2 = struct.unpack_from(">b", self.data, pos + 1)[0]
                pos += 2
            a, b, c, d = 1.0, 0.0, 0.0, 1.0
            if flags & WE_HAVE_A_SCALE:
                a = d = struct.unpack_from(">h", self.data, pos)[0] / 16384
                pos += 2
            elif flags & WE_HAVE_AN_X_AND_Y_SCALE:
                a, d = (v / 16384 for v in struct.unpack_from(">hh", self.data, pos))
                pos += 4
            elif flags & WE_HAVE_A_TWO_BY_TWO:
                a, b, c, d = (v / 16384 for v in struct.unpack_from(">hhhh", self.data, pos))
                pos += 8

            dx, dy = (a1, a2) if flags & ARGS_ARE_XY_VALUES else (0.0, 0.0)
            for contour in self.contours(gid):
                contours.append([Point(p.x * a + p.y * c + dx, p.x * b + p.y * d + dy, p.on) for p in contour])
            if not flags & MORE_COMPONENTS:
                break
        return contours


def _rotate_to_on_curve(points: list[Point]) -> tuple[list[Point], Point] | None:
    """Return the contour starting at an on-curve point, plus that start point."""
    n = len(points)
    if n == 0:
        return None
    for i, p in enumerate(points):
        if p.on:
            return points[i:] + points[:i], p
    # all off-curve: synthesise a start at the midpoint of the first two
    mid = Point((points[0].x + points[1].x) / 2, (points[0].y + points[1].y) / 2, True)
    return [mid] + points, mid


def contour_to_path(points: list[Point]) -> str:
    rotated = _rotate_to_on_curve(points)
    if rotated is None:
        return ""
    pts, start = rotated
    out = [f"M{start.x:g} {start.y:g}"]
    i = 1
    n = len(pts)
    while i < n + 1:
        p = pts[i % n]
        if p.on:
            out.append(f"L{p.x:g} {p.y:g}")
            i += 1
        else:
            nxt = pts[(i + 1) % n]
            if nxt.on:
                out.append(f"Q{p.x:g} {p.y:g} {nxt.x:g} {nxt.y:g}")
                i += 2
            else:
                mx, my = (p.x + nxt.x) / 2, (p.y + nxt.y) / 2
                out.append(f"Q{p.x:g} {p.y:g} {mx:g} {my:g}")
                i += 1
    out.append("Z")
    return "".join(out)


def path_bbox(d: str, steps: int = 12) -> tuple[float, float, float, float]:
    """Ink bounds of absolute M/L/Q path data (as produced by text_to_path)."""
    tokens = re.findall(r"[MLQZ]|-?\d*\.?\d+(?:[eE][-+]?\d+)?", d)
    xs: list[float] = []
    ys: list[float] = []
    i = 0
    cur = (0.0, 0.0)
    start = (0.0, 0.0)
    while i < len(tokens):
        cmd = tokens[i]
        i += 1
        if cmd == "M":
            cur = (float(tokens[i]), float(tokens[i + 1]))
            start = cur
            xs.append(cur[0])
            ys.append(cur[1])
            i += 2
        elif cmd == "L":
            cur = (float(tokens[i]), float(tokens[i + 1]))
            xs.append(cur[0])
            ys.append(cur[1])
            i += 2
        elif cmd == "Q":
            cx, cy = float(tokens[i]), float(tokens[i + 1])
            ex, ey = float(tokens[i + 2]), float(tokens[i + 3])
            i += 4
            for s in range(1, steps + 1):
                t = s / steps
                mt = 1 - t
                xs.append(mt * mt * cur[0] + 2 * mt * t * cx + t * t * ex)
                ys.append(mt * mt * cur[1] + 2 * mt * t * cy + t * t * ey)
            cur = (ex, ey)
        elif cmd == "Z":
            cur = start
    return min(xs), min(ys), max(xs), max(ys)


def text_to_path(
    font: TTF,
    text: str,
    size: float,
    tracking: float = 0.0,
    pairs: dict[tuple[str, str], float] | None = None,
    baseline: float = 0.0,
) -> tuple[str, float]:
    """Return (path data, advance width) for `text`, y down, baseline at `baseline`."""
    scale = size / font.units_per_em
    pairs = pairs or {}
    x = 0.0
    parts = []
    for i, ch in enumerate(text):
        gid = font.glyph_id(ch)
        for contour in font.contours(gid):
            moved = [Point(p.x * scale + x, baseline - p.y * scale, p.on) for p in contour]
            parts.append(contour_to_path(moved))
        advance = font.advance(ch) * scale
        x += advance + tracking
        if i + 1 < len(text):
            x += pairs.get((ch, text[i + 1]), 0.0)
    return "".join(parts), x

