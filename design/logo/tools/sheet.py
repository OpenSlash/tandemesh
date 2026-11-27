"""Test sheets and the concept overview board, drawn with Pillow.

Stands in for the skill's preview_sheet.py / concept_sheet.py (their scripts
cannot be fetched in this sandbox), producing PNGs directly so marks can be
judged at true sizes on a neutral artboard.

Usage:
  python sheet.py compare a.svg b.svg --out compare.png --size 220 [--label T]
  python sheet.py sizes a.svg b.svg --out sizes.png
  python sheet.py board --spec board.json --out concepts.png
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

import raster

HERE = Path(__file__).resolve().parent
ROOT = HERE.parent
REPO = ROOT.parent.parent
FONT_REGULAR = REPO / "apps" / "desktop" / "frontend" / "public" / "Inter-Medium.ttf"
FONT_FALLBACK = Path(r"C:\Windows\Fonts\segoeui.ttf")
FONT_BOLD_PATH = Path(r"C:\Windows\Fonts\seguisb.ttf")

INK = (17, 17, 17)
MUTED = (108, 114, 126)
FAINT = (150, 156, 168)
LINE = (228, 230, 234)
BG = (255, 255, 255)
RISK = (150, 112, 40)


def font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont:
    if bold and FONT_BOLD_PATH.exists():
        return ImageFont.truetype(str(FONT_BOLD_PATH), size)
    path = FONT_REGULAR if FONT_REGULAR.exists() else FONT_FALLBACK
    return ImageFont.truetype(str(path), size)


def label(draw: ImageDraw.ImageDraw, xy, text: str, size: int = 14, fill=MUTED, bold: bool = False, anchor="la"):
    draw.text(xy, text, font=font(size, bold), fill=fill, anchor=anchor)


def mark(path: Path, size: int, colour=(17, 17, 17)) -> Image.Image:
    img = raster.render_svg(path.read_text(encoding="utf-8"), size)
    if colour != (17, 17, 17):
        solid = Image.new("RGBA", img.size, colour + (255,))
        solid.putalpha(img.getchannel("A"))
        img = solid
    return img


def wrap(text: str, width: int) -> list[str]:
    words, lines, line = text.split(), [], ""
    for w in words:
        candidate = f"{line} {w}".strip()
        if len(candidate) > width and line:
            lines.append(line)
            line = w
        else:
            line = candidate
    if line:
        lines.append(line)
    return lines


def compare(files: list[Path], out: Path, size: int = 220, note: str | None = None) -> None:
    pad = 30
    width = len(files) * (size + pad * 2)
    top = 46 if note else 16
    height = top + size + pad + 30
    img = Image.new("RGB", (width, height), BG)
    draw = ImageDraw.Draw(img)
    if note:
        label(draw, (pad, 14), note, 15)
    for i, path in enumerate(files):
        x = i * (size + pad * 2) + pad
        draw.rectangle([x - 1, top - 1, x + size, top + size], outline=LINE)
        art = mark(path, size)
        img.paste(art, (x, top), art)
        label(draw, (x, top + size + 10), path.stem, 13)
    img.save(out)
    print(f"wrote {out}")


def sizes(files: list[Path], out: Path) -> None:
    """True-size ladder on light and dark grounds, one row per file."""
    ladder = (16, 24, 32, 48, 64, 96)
    row_h = 126
    pad = 40
    label_w = 190
    width = pad * 2 + label_w + sum(ladder) + 28 * len(ladder)
    height = pad + row_h * len(files) * 2 + 30
    img = Image.new("RGB", (width, height), BG)
    draw = ImageDraw.Draw(img)

    y = pad
    for path in files:
        for bg in ((255, 255, 255), (14, 15, 17)):
            draw.rectangle([pad, y, width - pad, y + row_h - 10], fill=bg)
            light = bg[0] > 128
            fg = (17, 17, 17) if light else (245, 245, 245)
            label(draw, (pad + 14, y + 14), path.stem, 13, FAINT if light else (150, 156, 168))
            label(draw, (pad + 14, y + 36), "on light" if light else "on dark", 10, FAINT)
            x = pad + label_w
            for px in ladder:
                art = mark(path, px, fg)
                img.paste(art, (x, y + 24), art)
                x += px + 28
            y += row_h
    img.save(out)
    print(f"wrote {out}")


STRESS_COLUMNS = ("mark", "mirrored", "rotated 180", "true 16 px", "squint blur")


def stress(files: list[Path], out: Path, size: int = 180) -> None:
    """Reading tests: mirror, rotation, true 16 px, squint blur."""
    from PIL import ImageFilter

    pad = 26
    label_w = 150
    width = label_w + len(STRESS_COLUMNS) * (size + pad) + pad
    height = pad + len(files) * (size + pad + 26) + 20
    img = Image.new("RGB", (width, height), BG)
    draw = ImageDraw.Draw(img)

    for c, name in enumerate(STRESS_COLUMNS):
        label(draw, (label_w + c * (size + pad) + 4, 6), name.upper(), 9)

    y = pad
    for path in files:
        base = mark(path, size)
        cells = [
            base,
            base.transpose(Image.FLIP_LEFT_RIGHT),
            base.rotate(180),
            mark(path, 16).resize((size, size), Image.NEAREST),
            base.filter(ImageFilter.GaussianBlur(5)),
        ]
        label(draw, (8, y + size // 2), path.stem, 13, INK, anchor="lm")
        for c, cell in enumerate(cells):
            x = label_w + c * (size + pad) + pad // 2
            img.paste(cell, (x, y), cell)
        y += size + pad + 26

    img.save(out)
    print(f"wrote {out}")


def board(spec_path: Path, out: Path) -> None:
    spec = json.loads(spec_path.read_text(encoding="utf-8"))
    concepts = spec["concepts"]
    width = spec.get("width", 1560)
    pad = 64
    gutter = 40
    col_w = (width - pad * 2 - gutter * (len(concepts) - 1)) // len(concepts)
    mark_size = min(210, col_w - 80)
    top = 214
    height = 955

    img = Image.new("RGB", (width, height), BG)
    draw = ImageDraw.Draw(img)

    draw.text((pad, 52), spec["title"], font=font(33, True), fill=INK)
    draw.text((pad, 100), spec["subtitle"], font=font(15), fill=MUTED)
    label(draw, (width - pad, 100), spec.get("meta", ""), 13, MUTED, anchor="ra")
    draw.line([pad, top - 46, width - pad, top - 46], fill=LINE, width=1)

    for i, concept in enumerate(concepts):
        x = pad + i * (col_w + gutter)
        cx = x + col_w // 2

        label(draw, (x, top - 36), concept["letter"], 28, INK, True)
        label(draw, (x + 38, top - 32), concept["name"], 20, INK, True)
        label(draw, (x + 38, top - 8), concept["type"], 12, MUTED)
        if concept.get("recommended"):
            pill_w = 112
            draw.rounded_rectangle([x + col_w - pill_w, top - 36, x + col_w, top - 10], 13, fill=INK)
            label(draw, (x + col_w - pill_w / 2, top - 23), "RECOMMENDED", 9, (255, 255, 255), True, anchor="mm")

        y = top + 6
        art = mark(ROOT / concept["symbol"], mark_size)
        img.paste(art, (cx - mark_size // 2, y), art)
        y += mark_size + 26

        if concept.get("tile"):
            tile = mark(ROOT / concept["tile"], 72)
            img.paste(tile, (x + 6, y), tile)
            label(draw, (x + 90, y + 6), "APP ICON", 10)
            label(draw, (x + 90, y + 24), "mono mark", 12, MUTED)
            label(draw, (x + 90, y + 42), "on dark tile", 12, MUTED)
            y += 72 + 24
        else:
            y += 10

        label(draw, (x + 6, y), "LOCKUP", 10)
        draw_lockup(img, draw, ROOT / concept["symbol"], concept.get("wordmark", "Tandemesh"), x + 6, y + 16)
        y += 84

        label(draw, (x + 6, y), "TRUE SIZE  16 / 32 / 64", 10)
        sx = x + 6
        for px in (16, 32, 64):
            small = mark(ROOT / concept["symbol"], px)
            img.paste(small, (sx, y + 26), small)
            sx += px + 30
        y += 112

        lines = wrap(concept["idea"], int(col_w / 7.4))
        for n, line in enumerate(lines):
            label(draw, (x + 6, y + n * 21), line, 13)
        y += 21 * len(lines) + 10

        risk_lines = wrap(concept.get("risk", ""), int(col_w / 6.8))
        for n, line in enumerate(risk_lines):
            label(draw, (x + 6, y + n * 19), line, 12, RISK)

        if i:
            draw.line([x - gutter // 2, top - 36, x - gutter // 2, height - 96], fill=LINE, width=1)

    draw.line([pad, height - 84, width - pad, height - 84], fill=LINE, width=1)
    for n, line in enumerate(spec.get("footer", "").split("\n")):
        label(draw, (pad, height - 62 + n * 20), line, 12)

    img.save(out)
    print(f"wrote {out} ({width}x{height})")


def draw_lockup(img: Image.Image, draw: ImageDraw.ImageDraw, symbol: Path, title: str, x: int, y: int) -> None:
    size = 38
    art = mark(symbol, size)
    img.paste(art, (x, y), art)
    draw.text((x + size + 14, y + size // 2 + 1), title, font=font(26, True), fill=INK, anchor="lm")


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser()
    sub = parser.add_subparsers(dest="mode", required=True)

    cmp_parser = sub.add_parser("compare")
    cmp_parser.add_argument("files", nargs="+")
    cmp_parser.add_argument("--out", required=True)
    cmp_parser.add_argument("--size", type=int, default=220)
    cmp_parser.add_argument("--label")

    sizes_parser = sub.add_parser("sizes")
    sizes_parser.add_argument("files", nargs="+")
    sizes_parser.add_argument("--out", required=True)

    board_parser = sub.add_parser("board")
    board_parser.add_argument("--spec", required=True)
    board_parser.add_argument("--out", required=True)

    stress_parser = sub.add_parser("stress")
    stress_parser.add_argument("files", nargs="+")
    stress_parser.add_argument("--out", required=True)
    stress_parser.add_argument("--size", type=int, default=170)

    args = parser.parse_args(argv)
    if args.mode == "compare":
        compare([Path(f) for f in args.files], Path(args.out), args.size, args.label)
    elif args.mode == "sizes":
        sizes([Path(f) for f in args.files], Path(args.out))
    elif args.mode == "stress":
        stress([Path(f) for f in args.files], Path(args.out), args.size)
    else:
        board(Path(args.spec), Path(args.out))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))

