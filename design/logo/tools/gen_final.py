"""Generate the three checkpoint symbols (and their dark-tile variants).

Everything is computed from parameters so proportions can be tuned without
hand-editing path data. Output goes to design/logo/concepts/.

Symbols are drawn on a 256 canvas, black on transparent, with the ink bounding
box optically centred.
"""

from __future__ import annotations

from pathlib import Path

OUT = Path(__file__).resolve().parent.parent / "concepts"
INK = "#111111"


def svg(body: str, title: str) -> str:
    return (
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" role="img" aria-labelledby="title">\n'
        f'  <title id="title">{title}</title>\n  {body}\n</svg>\n'
    )


def write(name: str, body: str, title: str) -> None:
    (OUT / f"{name}.svg").write_text(svg(body, title), encoding="utf-8")
    print(f"wrote {name}.svg")


def poly(points, fill_rule: str = "") -> str:
    d = "M" + " L".join(f"{x:g} {y:g}" for x, y in points) + " Z"
    rule = f' fill-rule="{fill_rule}"' if fill_rule else ""
    return f'<path fill="{INK}"{rule} d="{d}"/>'


# --------------------------------------------------------------------------- #
# A - fin: leading edge convex, trailing edge scooped, tip swept back
# --------------------------------------------------------------------------- #
def fin() -> str:
    d = "M48 216 C66 152 120 84 188 40 C174 110 182 168 208 216 Z"
    return f'<path fill="{INK}" d="{d}"/>'


# --------------------------------------------------------------------------- #
# B - sheared T: one T, sheared 10 degrees, bbox centred
# --------------------------------------------------------------------------- #
def sheared_t(k: float = 0.1763) -> str:  # k = tan 10 deg
    x0, y0 = 40, 48
    x1, y1 = 216, 208
    cb = 56  # crossbar thickness
    stem = 56  # stem width
    cx = (x0 + x1) / 2
    points = [
        (x0, y0), (x1, y0), (x1, y0 + cb),
        (cx + stem / 2, y0 + cb), (cx + stem / 2, y1),
        (cx - stem / 2, y1), (cx - stem / 2, y0 + cb),
        (x0, y0 + cb),
    ]
    sheared = [(round(x - k * (y - 128)), y) for x, y in points]
    min_x = min(p[0] for p in sheared)
    max_x = max(p[0] for p in sheared)
    shift = round(128 - (min_x + max_x) / 2)
    sheared = [(x + shift, y) for x, y in sheared]
    return poly(sheared)


# --------------------------------------------------------------------------- #
# C - stream: two chevrons, one direction
# --------------------------------------------------------------------------- #
def stream() -> str:
    arm = 60     # horizontal run of one arm (60 deg slope at this height)
    thick = 40   # chevron thickness
    gap = 16
    width = 2 * (arm + thick) + gap
    x = (256 - width) / 2
    top, bottom = 24, 232
    mid = (top + bottom) / 2
    paths = []
    for i in range(2):
        x0 = x + i * (arm + thick + gap)
        d = (
            f"M{x0:g} {top:g} L{x0 + arm:g} {mid:g} L{x0:g} {bottom:g} "
            f"L{x0 + thick:g} {bottom:g} L{x0 + arm + thick:g} {mid:g} L{x0 + thick:g} {top:g} Z"
        )
        paths.append(f'<path fill="{INK}" d="{d}"/>')
    return "<g>" + "".join(paths) + "</g>"


# --------------------------------------------------------------------------- #
# dark app tile: mono mark knocked out of a rounded square
# --------------------------------------------------------------------------- #
def tile(symbol: str, scale: float, bg: str = "0B0C0E", fg: str = "FFFFFF", radius: int = 56) -> str:
    inner = symbol.replace(f'fill="{INK}"', f'fill="#{fg}"')
    size = 256 * scale
    offset = (256 - size) / 2
    return (
        f'<rect width="256" height="256" rx="{radius}" fill="#{bg}"/>\n'
        f'  <g transform="translate({offset:g},{offset:g}) scale({scale:g})">{inner}</g>'
    )


def main() -> None:
    symbols = {
        "a-fin": ("Tandemesh concept A - fin", fin(), 0.60),
        "b-shear": ("Tandemesh concept B - sheared T", sheared_t(), 0.58),
        "c-stream": ("Tandemesh concept C - stream", stream(), 0.54),
    }
    for name, (title, body, scale) in symbols.items():
        write(name, body, title)
        tile_path = OUT / f"tile-{name[0]}.svg"
        tile_path.write_text(svg(tile(body, scale), f"{title} (app tile)"), encoding="utf-8")
        print(f"wrote {tile_path.name}")


if __name__ == "__main__":
    main()

