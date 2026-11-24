"""Build the Tandemesh icon kit from the chosen direction (A 閳?fin).

Emits SVGs (symbol, small-size cut, reversed, lockups, tiles, colour variants),
PNG icon sets, favicon.ico, an icns for macOS and a web manifest 閳?all derived
from the same geometry so the set can be regenerated after any tweak.

Run: python design/logo/tools/kit.py
"""

from __future__ import annotations

import json
import struct
from pathlib import Path

from PIL import Image

import raster
from ttf import TTF, path_bbox, text_to_path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parent
REPO = ROOT.parent.parent
KIT = ROOT / "kit"
FONT = REPO / "apps" / "tandemesh-desktop" / "frontend" / "public" / "Inter-Medium.ttf"

INK = "0B0C0E"
PAPER = "F5F6F7"
ABYSS = "14424E"

# element order: symbol, gap, wordmark, padding 閳?all in the 256-unit lockup grid
LOCK_PAD = 26


# --------------------------------------------------------------------------- #
# marks
# --------------------------------------------------------------------------- #
def fin_path() -> str:
    return "M48 216 C66 152 120 84 188 40 C174 110 182 168 208 216 Z"


def fin_small_path() -> str:
    """16 px cut: blunter tip, shallower scoop, slightly wider base."""
    return "M44 216 C62 156 112 96 176 58 C168 116 178 170 206 216 Z"


def svg(width: float, height: float, body: str, title: str) -> str:
    w = f"{width:g}"
    h = f"{height:g}"
    return (
        f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {w} {h}" width="{w}" height="{h}" '
        f'role="img" aria-labelledby="title">\n  <title id="title">{title}</title>\n  {body}\n</svg>\n'
    )


def symbol_svg(colour: str = INK, scale: float = 1.0, small: bool = False, title: str = "Tandemesh") -> str:
    d = fin_small_path() if small else fin_path()
    if scale == 1.0:
        body = f'<path fill="#{colour}" d="{d}"/>'
    else:
        offset = 128 * (1 - scale)
        body = (
            f'<g transform="translate({offset:g},{offset:g}) scale({scale:g})">'
            f'<path fill="#{colour}" d="{d}"/></g>'
        )
    return svg(256, 256, body, title)


def tile_svg(colour: str = INK, mark: str = PAPER, scale: float = 0.82, radius: float = 56,
             bleed: bool = False, dy: float = -3.0, small: bool = False,
             title: str = "Tandemesh") -> str:
    tx = 128 - 128 * scale
    ty = 128 + dy - 128 * scale
    path = fin_small_path() if small else fin_path()
    body = (
        f'<rect x="-1" y="-1" width="258" height="258" rx="{0 if bleed else radius:g}" fill="#{colour}"/>\n'
        f'  <g transform="translate({tx:g},{ty:g}) scale({scale:g})">'
        f'<path fill="#{mark}" d="{path}"/></g>'
    )
    return svg(256, 256, body, title)


# --------------------------------------------------------------------------- #
# wordmark
# --------------------------------------------------------------------------- #
PAIRS = {
    ("T", "a"): -0.020,
    ("a", "n"): -0.004,
    ("n", "d"): -0.004,
    ("d", "e"): -0.006,
    ("e", "m"): -0.006,
    ("m", "e"): -0.008,
    ("e", "s"): -0.004,
    ("s", "h"): -0.006,
}


def wordmark(size: float, colour: str = INK, text: str = "Tandemesh",
             tracking: float = -0.014) -> tuple[str, float]:
    font = TTF(FONT)
    pairs = {k: v * size for k, v in PAIRS.items()}
    d, width = text_to_path(font, text, size, tracking=tracking * size, pairs=pairs, baseline=0)
    return f'<path fill="#{colour}" d="{d}"/>', width


FIN = (48.0, 40.0, 208.0, 216.0)  # ink bounds of fin_path() in the 256 box
FIN_W, FIN_H = FIN[2] - FIN[0], FIN[3] - FIN[1]


def place_fin(x: float, y: float, height: float) -> str:
    """Place the fin with its ink top-left at (x, y) and the given ink height."""
    s = height / FIN_H
    tx = x - FIN[0] * s
    ty = y - FIN[1] * s
    return (
        f'<g transform="translate({tx:g},{ty:g}) scale({s:g})">'
        f'<path fill="#{INK}" d="{fin_path()}"/></g>'
    )


def lockup_horizontal(colour: str = INK, size: float = 92) -> str:
    """Symbol left of the wordmark; 256 high, width follows."""
    body_wm, _advance = wordmark(size, colour)
    bx0, by0, bx1, by1 = path_bbox(body_wm)
    cap = 0.727 * size
    mark_h = 96.0
    mark_w = mark_h * FIN_W / FIN_H
    gap = 24.0

    mark_x = LOCK_PAD
    wm_x = mark_x + mark_w + gap - bx0
    width = wm_x + bx1 + LOCK_PAD

    baseline = 128 + cap / 2
    mark_y = baseline + 2 - mark_h
    body = place_fin(mark_x, mark_y, mark_h).replace(f'fill="#{INK}"', f'fill="#{colour}"')
    body += f'\n  <g transform="translate({wm_x:g},{baseline:g})">{body_wm}</g>'
    return svg(round(width, 2), 256, body, "Tandemesh")


def lockup_stacked(colour: str = INK, size: float = 96) -> str:
    body_wm, _advance = wordmark(size, colour)
    bx0, by0, bx1, by1 = path_bbox(body_wm)
    cap = 0.727 * size
    mark_h = 126.0
    mark_w = mark_h * FIN_W / FIN_H

    wm_w = bx1 - bx0
    width = max(mark_w, wm_w) + 2 * LOCK_PAD
    mark_x = (width - mark_w) / 2
    mark_y = 26.0
    baseline = mark_y + mark_h + 26 + cap

    body = place_fin(mark_x, mark_y, mark_h).replace(f'fill="#{INK}"', f'fill="#{colour}"')
    body += f'\n  <g transform="translate({(width - wm_w) / 2 - bx0:g},{baseline:g})">{body_wm}</g>'
    return svg(round(width, 2), round(baseline + 26, 2), body, "Tandemesh")


# --------------------------------------------------------------------------- #
# rendering + icon formats
# --------------------------------------------------------------------------- #
def render(svg_text: str, width: int, out: Path) -> Image.Image:
    img = raster.render_svg(svg_text, width)
    out.parent.mkdir(parents=True, exist_ok=True)
    img.save(out)
    return img


def png(svg_text: str, size: int, out: Path) -> None:
    render(svg_text, size, out)


def ico(images: list[tuple[int, bytes]], out: Path) -> None:
    """ICO with PNG-compressed entries (Vista+), one per requested size."""
    header = struct.pack("<HHH", 0, 1, len(images))
    entries, blobs, offset = b"", b"", 6 + 16 * len(images)
    for size, data in images:
        w = 0 if size >= 256 else size
        entries += struct.pack("<BBBBHHII", w, w, 0, 0, 1, 32, len(data), offset)
        blobs += data
        offset += len(data)
    out.write_bytes(header + entries + blobs)


def icns(images: list[tuple[str, bytes]], out: Path) -> None:
    body = b"".join(t.encode("ascii") + struct.pack(">I", len(d) + 8) + d for t, d in images)
    out.write_bytes(b"icns" + struct.pack(">I", len(body) + 8) + body)


def png_bytes(img: Image.Image) -> bytes:
    from io import BytesIO

    buf = BytesIO()
    img.save(buf, format="PNG")
    return buf.getvalue()


def export_app_icons() -> None:
    """Full icon set used by the browser, the desktop app and the OSes."""
    tile = tile_svg()
    maskable = tile_svg(bleed=True, scale=0.52)
    small = tile_svg(scale=0.70, small=True)

    sizes = (16, 24, 32, 48, 64, 128, 180, 192, 256, 512, 1024)
    for size in sizes:
        png(small if size <= 32 else tile, size, KIT / "png" / f"app-icon-{size}.png")

    # transparent symbol, for docs and places that already have a surface
    for size in (64, 128, 256, 512, 1024):
        png(symbol_svg(), size, KIT / "png" / f"symbol-{size}.png")

    icon_sizes = (16, 24, 32, 48, 64, 128, 256)
    entries = [(s, png_bytes(raster.render_svg(small if s <= 32 else tile, s))) for s in icon_sizes]
    ico(entries, KIT / "favicon.ico")

    icns_entries = [
        ("icp4", png_bytes(raster.render_svg(small, 16))),
        ("icp5", png_bytes(raster.render_svg(small, 32))),
        ("icp6", png_bytes(raster.render_svg(small, 64))),
        ("ic07", png_bytes(raster.render_svg(tile, 128))),
        ("ic08", png_bytes(raster.render_svg(tile, 256))),
        ("ic09", png_bytes(raster.render_svg(tile, 512))),
        ("ic10", png_bytes(raster.render_svg(tile, 1024))),
    ]
    icns(icns_entries, KIT / "app.icns")

    manifest = {
        "name": "Tandemesh",
        "short_name": "Tandemesh",
        "description": "An agent execution plane.",
        "start_url": "/",
        "display": "standalone",
        "background_color": f"#{INK}",
        "theme_color": f"#{INK}",
        "icons": [
            {"src": "/icon-192.png", "sizes": "192x192", "type": "image/png"},
            {"src": "/icon-512.png", "sizes": "512x512", "type": "image/png"},
            {"src": "/icon-maskable-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable"},
        ],
    }
    (KIT / "site.webmanifest").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")

    png(tile, 192, KIT / "web" / "icon-192.png")
    png(tile, 512, KIT / "web" / "icon-512.png")
    png(maskable, 512, KIT / "web" / "icon-maskable-512.png")
    png(tile, 180, KIT / "web" / "apple-touch-icon.png")
    png(small, 32, KIT / "web" / "favicon-32.png")
    png(small, 16, KIT / "web" / "favicon-16.png")
    (KIT / "web" / "favicon.svg").write_text(small, encoding="utf-8")


def main() -> None:
    KIT.mkdir(parents=True, exist_ok=True)
    (KIT / "svg").mkdir(exist_ok=True)

    symbols = {
        "tandemesh-symbol-ink.svg": symbol_svg(INK),
        "tandemesh-symbol-abyss.svg": symbol_svg(ABYSS),
        "tandemesh-symbol-paper.svg": symbol_svg(PAPER),
        "tandemesh-symbol-16.svg": symbol_svg(INK, small=True),
        "tandemesh-symbol-reversed.svg": symbol_svg(PAPER, scale=0.97),
        "tandemesh-wordmark-ink.svg": svg(0, 0, "", "placeholder"),
    }
    wm, wm_w = wordmark(100, INK)
    symbols["tandemesh-wordmark-ink.svg"] = svg(wm_w + 4, 140, f'<g transform="translate(2,104)">{wm}</g>', "Tandemesh")
    symbols["tandemesh-lockup-horizontal-ink.svg"] = lockup_horizontal(INK)
    symbols["tandemesh-lockup-horizontal-paper.svg"] = lockup_horizontal(PAPER)
    symbols["tandemesh-lockup-horizontal-abyss.svg"] = lockup_horizontal(ABYSS)
    symbols["tandemesh-lockup-stacked-ink.svg"] = lockup_stacked(INK)
    symbols["tandemesh-lockup-stacked-paper.svg"] = lockup_stacked(PAPER)
    symbols["tandemesh-tile.svg"] = tile_svg()
    symbols["tandemesh-tile-abyss.svg"] = tile_svg(ABYSS)
    symbols["tandemesh-tile-paper.svg"] = tile_svg(PAPER, INK)

    for name, text in symbols.items():
        (KIT / "svg" / name).write_text(text, encoding="utf-8")
        print(f"svg/{name}")

    export_app_icons()
    print("png + favicon.ico + app.icns + site.webmanifest")


if __name__ == "__main__":
    main()




