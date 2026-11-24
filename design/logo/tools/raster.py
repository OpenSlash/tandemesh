"""Tiny dependency-free-ish SVG -> PNG rasteriser for evaluating logo marks.

The skill renders with cairosvg / rsvg / Inkscape / headless Chrome. In this
sandbox cairosvg is absent and Chromium cannot start (named pipes are blocked),
so this module flattens the SVG subset used by the concepts (paths, rects,
circles, groups, translate/scale) and rasterises it with Pillow at 4x
supersampling. Fills only; strokes are not part of the final marks.

Public API:
    render_svg(text: str, size: int, bg: tuple | None = None) -> Image
"""

from __future__ import annotations

import math
import re
from pathlib import Path
from xml.etree import ElementTree as ET

from PIL import Image, ImageChops, ImageDraw

SS = 4  # supersampling factor
CURVE_STEPS = 24
NUM = re.compile(r"[-+]?(?:\d*\.\d+|\d+)(?:[eE][-+]?\d+)?")
CMD = re.compile(r"[MmLlHhVvCcSsQqTtAaZz]")
SVG_NS = "{http://www.w3.org/2000/svg}"


# --------------------------------------------------------------------------- #
# path parsing
# --------------------------------------------------------------------------- #
def _tokenize(d: str) -> list[str]:
    return re.findall(r"[MmLlHhVvCcSsQqTtAaZz]|[-+]?(?:\d*\.\d+|\d+)(?:[eE][-+]?\d+)?", d)


def _flatten_cubic(p0, p1, p2, p3, out: list[tuple[float, float]]) -> None:
    for i in range(1, CURVE_STEPS + 1):
        t = i / CURVE_STEPS
        mt = 1 - t
        x = mt**3 * p0[0] + 3 * mt**2 * t * p1[0] + 3 * mt * t**2 * p2[0] + t**3 * p3[0]
        y = mt**3 * p0[1] + 3 * mt**2 * t * p1[1] + 3 * mt * t**2 * p2[1] + t**3 * p3[1]
        out.append((x, y))


def _flatten_quad(p0, p1, p2, out: list[tuple[float, float]]) -> None:
    for i in range(1, CURVE_STEPS + 1):
        t = i / CURVE_STEPS
        mt = 1 - t
        out.append((mt**2 * p0[0] + 2 * mt * t * p1[0] + t**2 * p2[0],
                    mt**2 * p0[1] + 2 * mt * t * p1[1] + t**2 * p2[1]))


def path_subpaths(d: str) -> list[list[tuple[float, float]]]:
    """Flatten path data into a list of closed polygons."""
    tokens = _tokenize(d)
    subpaths: list[list[tuple[float, float]]] = []
    cur: list[tuple[float, float]] = []
    pos = (0.0, 0.0)
    start = (0.0, 0.0)
    last_ctrl = None
    cmd = None
    i = 0

    def take(n: int) -> list[float]:
        nonlocal i
        vals = [float(tokens[i + k]) for k in range(n)]
        i += n
        return vals

    while i < len(tokens):
        tok = tokens[i]
        if CMD.fullmatch(tok):
            cmd = tok
            i += 1
            if cmd in "Zz":
                if cur:
                    cur.append(start)
                    subpaths.append(cur)
                cur = []
                pos = start
                last_ctrl = None
                continue
        if cmd is None:
            i += 1
            continue

        rel = cmd.islower()
        c = cmd.upper()
        if c == "M":
            x, y = take(2)
            if rel:
                x, y = pos[0] + x, pos[1] + y
            if cur:
                cur.append(start)
                subpaths.append(cur)
            cur = [(x, y)]
            pos = start = (x, y)
            cmd = "l" if rel else "L"  # subsequent pairs are implicit lineto
            last_ctrl = None
        elif c == "L":
            x, y = take(2)
            if rel:
                x, y = pos[0] + x, pos[1] + y
            cur.append((x, y))
            pos = (x, y)
            last_ctrl = None
        elif c == "H":
            (x,) = take(1)
            x = pos[0] + x if rel else x
            cur.append((x, pos[1]))
            pos = (x, pos[1])
            last_ctrl = None
        elif c == "V":
            (y,) = take(1)
            y = pos[1] + y if rel else y
            cur.append((pos[0], y))
            pos = (pos[0], y)
            last_ctrl = None
        elif c == "C":
            x1, y1, x2, y2, x, y = take(6)
            if rel:
                x1, y1 = pos[0] + x1, pos[1] + y1
                x2, y2 = pos[0] + x2, pos[1] + y2
                x, y = pos[0] + x, pos[1] + y
            _flatten_cubic(pos, (x1, y1), (x2, y2), (x, y), cur)
            last_ctrl = (x2, y2)
            pos = (x, y)
        elif c == "S":
            x2, y2, x, y = take(4)
            if rel:
                x2, y2 = pos[0] + x2, pos[1] + y2
                x, y = pos[0] + x, pos[1] + y
            if last_ctrl:
                x1, y1 = 2 * pos[0] - last_ctrl[0], 2 * pos[1] - last_ctrl[1]
            else:
                x1, y1 = pos
            _flatten_cubic(pos, (x1, y1), (x2, y2), (x, y), cur)
            last_ctrl = (x2, y2)
            pos = (x, y)
        elif c == "Q":
            x1, y1, x, y = take(4)
            if rel:
                x1, y1 = pos[0] + x1, pos[1] + y1
                x, y = pos[0] + x, pos[1] + y
            _flatten_quad(pos, (x1, y1), (x, y), cur)
            last_ctrl = (x1, y1)
            pos = (x, y)
        elif c == "T":
            x, y = take(2)
            if rel:
                x, y = pos[0] + x, pos[1] + y
            if last_ctrl:
                x1, y1 = 2 * pos[0] - last_ctrl[0], 2 * pos[1] - last_ctrl[1]
            else:
                x1, y1 = pos
            _flatten_quad(pos, (x1, y1), (x, y), cur)
            last_ctrl = (x1, y1)
            pos = (x, y)
        else:  # A/a - approximated by a straight line; concepts avoid arcs
            take(7)
            cur.append(pos)
            last_ctrl = None
    if cur:
        cur.append(start)
        subpaths.append(cur)
    return subpaths


# --------------------------------------------------------------------------- #
# shapes
# --------------------------------------------------------------------------- #
def _rect_points(x, y, w, h, rx) -> list[tuple[float, float]]:
    if not rx:
        return [(x, y), (x + w, y), (x + w, y + h), (x, y + h)]
    k = 0.5523 * rx
    pts: list[tuple[float, float]] = []
    _flatten_cubic((x + rx, y), (x + rx - k, y), (x, y + rx - k), (x, y + rx), pts)
    _flatten_cubic((x, y + h - rx), (x, y + h - rx + k), (x + rx - k, y + h), (x + rx, y + h), pts)
    _flatten_cubic((x + w - rx, y + h), (x + w - rx + k, y + h), (x + w, y + h - rx + k), (x + w, y + h - rx), pts)
    _flatten_cubic((x + w, y + rx), (x + w, y + rx - k), (x + w - rx + k, y), (x + w - rx, y), pts)
    return pts


def _ellipse_points(cx, cy, rx, ry) -> list[tuple[float, float]]:
    k = 0.5523
    pts: list[tuple[float, float]] = []
    _flatten_cubic((cx + rx, cy), (cx + rx, cy + k * ry), (cx + k * rx, cy + ry), (cx, cy + ry), pts)
    _flatten_cubic((cx, cy + ry), (cx - k * rx, cy + ry), (cx - rx, cy + k * ry), (cx - rx, cy), pts)
    _flatten_cubic((cx - rx, cy), (cx - rx, cy - k * ry), (cx - k * rx, cy - ry), (cx, cy - ry), pts)
    _flatten_cubic((cx, cy - ry), (cx + k * rx, cy - ry), (cx + rx, cy - k * ry), (cx + rx, cy), pts)
    return pts


def _parse_colour(value: str | None) -> tuple[int, int, int] | None:
    if not value or value == "none":
        return None
    value = value.strip().lower()
    if value.startswith("#"):
        h = value[1:]
        if len(h) == 3:
            h = "".join(c * 2 for c in h)
        return tuple(int(h[i:i + 2], 16) for i in (0, 2, 4))  # type: ignore[return-value]
    named = {
        "black": (0, 0, 0), "white": (255, 255, 255),
        "currentcolor": (17, 17, 17), "red": (255, 0, 0),
    }
    return named.get(value)


def _transform_of(el) -> tuple[float, float, float]:
    """Return (tx, ty, scale) for translate/scale transforms on an element."""
    t = el.get("transform")
    tx = ty = 0.0
    scale = 1.0
    if not t:
        return tx, ty, scale
    for name, args in re.findall(r"(translate|scale|matrix)\s*\(([^)]*)\)", t):
        vals = [float(v) for v in re.findall(r"[-+]?(?:\d*\.\d+|\d+)", args)]
        if name == "translate":
            tx += vals[0]
            ty += vals[1] if len(vals) > 1 else 0.0
        elif name == "scale":
            scale *= vals[0]
            if len(vals) > 1 and vals[1] != vals[0]:
                raise ValueError("non-uniform scale is not supported")
        else:
            raise ValueError("matrix transforms are not supported")
    return tx, ty, scale


def _collect(el, inherited: dict, out: list) -> None:
    tag = el.tag.replace(SVG_NS, "")
    attrs = dict(inherited)
    for key in ("fill", "fill-rule", "opacity", "fill-opacity"):
        if el.get(key) is not None:
            attrs[key] = el.get(key)
    tx, ty, scale = _transform_of(el)
    parent_tx = attrs.get("_tx", 0.0)
    parent_ty = attrs.get("_ty", 0.0)
    parent_scale = attrs.get("_scale", 1.0)
    # p' = (p * s + t) * S + T  ==  p * (s*S) + (t*S + T)
    attrs["_tx"] = parent_tx + parent_scale * tx
    attrs["_ty"] = parent_ty + parent_scale * ty
    attrs["_scale"] = parent_scale * scale

    if tag in ("g", "svg"):
        for child in el:
            _collect(child, attrs, out)
        return
    if tag in ("title", "desc", "defs", "metadata", "style"):
        return

    shapes: list[list[tuple[float, float]]] = []
    if tag == "path":
        shapes = path_subpaths(el.get("d", ""))
    elif tag == "rect":
        shapes = [_rect_points(
            float(el.get("x", 0)), float(el.get("y", 0)),
            float(el.get("width", 0)), float(el.get("height", 0)),
            float(el.get("rx", 0) or 0),
        )]
    elif tag == "circle":
        r = float(el.get("r", 0))
        shapes = [_ellipse_points(float(el.get("cx", 0)), float(el.get("cy", 0)), r, r)]
    elif tag == "ellipse":
        shapes = [_ellipse_points(
            float(el.get("cx", 0)), float(el.get("cy", 0)),
            float(el.get("rx", 0)), float(el.get("ry", 0)),
        )]
    elif tag == "polygon":
        nums = [float(v) for v in re.findall(r"[-+]?(?:\d*\.\d+|\d+)", el.get("points", ""))]
        shapes = [list(zip(nums[0::2], nums[1::2]))] if len(nums) >= 6 else []
    else:
        return
    colour = _parse_colour(attrs.get("fill", "#000000"))
    if colour is None:
        return
    out.append({
        "subpaths": shapes,
        "fill": colour,
        "evenodd": (attrs.get("fill-rule", "nonzero") == "evenodd"),
        "transform": (attrs["_tx"], attrs["_ty"], attrs["_scale"]),
        "fill_opacity": float(attrs.get("fill-opacity", 1) or 1),
    })


def _fill_polygons(polygons: list[list[tuple[float, float]]], width: int, height: int, evenodd: bool):
    """Scanline fill with real winding rules; returns a numpy uint8 mask."""
    import numpy as np

    mask = np.zeros((height, width), dtype=np.uint8)
    rows_idx: list[int] = []
    xs: list[float] = []
    dirs: list[int] = []
    for poly in polygons:
        n = len(poly)
        for i in range(n):
            x1, y1 = poly[i]
            x2, y2 = poly[(i + 1) % n]
            if y1 == y2:
                continue
            direction = 1 if y2 > y1 else -1
            ylo, yhi = (y1, y2) if y1 < y2 else (y2, y1)
            r0 = max(0, int(math.ceil(ylo - 0.5)))
            r1 = min(height - 1, int(math.ceil(yhi - 0.5)) - 1)
            if r1 < r0:
                continue
            rows = np.arange(r0, r1 + 1)
            ys = rows + 0.5
            xcross = x1 + (ys - y1) * (x2 - x1) / (y2 - y1)
            rows_idx.append(rows)
            xs.append(xcross)
            dirs.append(np.full(rows.shape, direction, dtype=np.int32))

    if not rows_idx:
        return mask
    rows_all = np.concatenate(rows_idx)
    xs_all = np.concatenate(xs)
    dirs_all = np.concatenate(dirs)

    order = np.lexsort((xs_all, rows_all))
    rows_sorted = rows_all[order]
    xs_sorted = xs_all[order]
    dirs_sorted = dirs_all[order]

    boundaries = np.flatnonzero(np.diff(rows_sorted)) + 1
    starts = np.concatenate(([0], boundaries))
    ends = np.concatenate((boundaries, [len(rows_sorted)]))
    for s, e in zip(starts, ends):
        row = int(rows_sorted[s])
        row_xs = xs_sorted[s:e]
        row_dirs = dirs_sorted[s:e]
        if len(row_xs) < 2:
            continue
        if evenodd:
            inside = (np.arange(len(row_xs)) % 2) == 0
        else:
            winding = np.cumsum(row_dirs)
            inside = winding != 0
        # spans run from crossing i to i+1 while the region after crossing i is inside
        span_starts = row_xs[:-1][inside[:-1]]
        span_ends = row_xs[1:][inside[:-1]]
        for x0, x1 in zip(span_starts, span_ends):
            a = max(0, int(round(x0)))
            b = min(width, int(round(x1)))
            if b > a:
                mask[row, a:b] = 255
    return mask


def render_svg(text: str, size: int, bg: tuple[int, int, int] | None = None) -> Image.Image:
    """Render at `size` px wide, keeping the viewBox aspect ratio."""
    root = ET.fromstring(text)
    vb = [float(v) for v in (root.get("viewBox") or f"0 0 {size} {size}").replace(",", " ").split()]
    vx, vy, vw, vh = vb
    width = size * SS
    height = max(1, round(size * SS * vh / vw))
    sx = width / vw
    sy = height / vh
    img = Image.new("RGBA", (width, height), bg + (255,) if bg else (0, 0, 0, 0))
    ops: list = []
    _collect(root, {}, ops)

    for op in ops:
        tx, ty, scale = op["transform"]
        polygons = []
        for sub in op["subpaths"]:
            if len(sub) < 3:
                continue
            polygons.append([((px * scale + tx - vx) * sx, (py * scale + ty - vy) * sy) for px, py in sub])
        if not polygons:
            continue
        mask = _fill_polygons(polygons, width, height, op["evenodd"])
        layer = Image.fromarray(mask, mode="L")
        colour = op["fill"] + (int(round(255 * op["fill_opacity"])),)
        swatch = Image.new("RGBA", (width, height), colour)
        img.paste(swatch, (0, 0), layer)

    return img.resize((size, max(1, round(size * vh / vw))), Image.LANCZOS)


def render_file(path: str | Path, size: int, out: str | Path, bg: tuple[int, int, int] | None = None) -> None:
    img = render_svg(Path(path).read_text(encoding="utf-8"), size, bg)
    Path(out).parent.mkdir(parents=True, exist_ok=True)
    img.save(out)


if __name__ == "__main__":
    import sys

    render_file(sys.argv[1], int(sys.argv[2]), sys.argv[3])
    print(f"wrote {sys.argv[3]}")

