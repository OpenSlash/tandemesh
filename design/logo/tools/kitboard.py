"""Presentation board for the finished icon kit: marks, mockups, palette.

Usage: python design/logo/tools/kitboard.py
"""

from __future__ import annotations

from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

import raster

HERE = Path(__file__).resolve().parent
ROOT = HERE.parent
KIT = ROOT / "kit"
REPO = ROOT.parent.parent
FONT = REPO / "apps" / "desktop" / "frontend" / "public" / "Inter-Medium.ttf"
FONT_BOLD = Path(r"C:\Windows\Fonts\seguisb.ttf")
MONO = Path(r"C:\Windows\Fonts\consola.ttf")
FONT_FALLBACK = Path(r"C:\Windows\Fonts\segoeui.ttf")

INK = (11, 12, 14)
ABYSS = (20, 66, 78)
PAPER = (245, 246, 247)
MUTED = (108, 114, 126)
FAINT = (150, 156, 168)
LINE = (228, 230, 234)
WHITE = (255, 255, 255)

W, H = 1600, 1245


def f(size: int, bold: bool = False) -> ImageFont.FreeTypeFont:
    path = FONT_BOLD if bold and FONT_BOLD.exists() else (FONT if FONT.exists() else FONT_FALLBACK)
    return ImageFont.truetype(str(path), size)


def mono(size: int) -> ImageFont.FreeTypeFont:
    return ImageFont.truetype(str(MONO), size)


def text(draw, xy, s, size=14, fill=MUTED, bold=False, anchor="la", font_obj=None):
    draw.text(xy, s, font=font_obj or f(size, bold), fill=fill, anchor=anchor)


def svg_render(name: str, width: int) -> Image.Image:
    return raster.render_svg((KIT / "svg" / name).read_text(encoding="utf-8"), width)


def card(img, box, fill=WHITE, radius=18, outline=LINE, width=1):
    layer = Image.new("RGBA", img.size, (0, 0, 0, 0))
    ImageDraw.Draw(layer).rounded_rectangle(box, radius, fill=fill + (255,), outline=outline + (255,), width=width)
    img.alpha_composite(layer)


def paste(img, art, xy, anchor="lt"):
    if anchor == "mm":
        xy = (xy[0] - art.width // 2, xy[1] - art.height // 2)
    elif anchor == "lm":
        xy = (xy[0], xy[1] - art.height // 2)
    img.alpha_composite(art, (int(xy[0]), int(xy[1])))


def build() -> Image.Image:
    img = Image.new("RGBA", (W, H), WHITE + (255,))
    draw = ImageDraw.Draw(img)

    # header
    text(draw, (72, 58), "Tandemesh — icon kit", 34, INK, True)
    text(draw, (72, 104), "Fin · monochrome · drawn on a 256 px grid · every file regenerated from one path", 15)
    text(draw, (W - 72, 104), "v1 · 2026-11-24", 13, MUTED, anchor="ra")
    draw.line([72, 148, W - 72, 148], fill=LINE, width=1)

    # 1. the marks
    tile = svg_render("tandemesh-tile.svg", 196)
    paste(img, tile, (72, 182))
    text(draw, (72, 390), "APP ICON", 10)
    text(draw, (72, 408), "favicon.ico, .icns, taskbar", 12)

    card(img, [296, 182, 1020, 372], PAPER, 18, (235, 237, 240))
    lockup = svg_render("tandemesh-lockup-horizontal-ink.svg", 440)
    paste(img, lockup, (296 + (1020 - 296 - 440) // 2, 182 + (190 - lockup.height) // 2))
    text(draw, (316, 390), "PRIMARY LOCKUP", 10)
    text(draw, (316, 408), "symbol + wordmark, 256 units high", 12)
    text(draw, (1000, 390), "clear space = ½ symbol height", 12, MUTED, anchor="ra")

    card(img, [1040, 182, W - 72, 372], PAPER, 18, (235, 237, 240))
    stacked = svg_render("tandemesh-lockup-stacked-ink.svg", 300)
    paste(img, stacked, (1040 + (W - 72 - 1040 - 300) // 2, 182 + (190 - stacked.height) // 2))
    text(draw, (1060, 390), "STACKED LOCKUP", 10)
    text(draw, (1060, 408), "square formats, avatars, merch", 12)

    # 2. in context
    text(draw, (72, 470), "IN CONTEXT", 11, MUTED, True)
    draw.line([72, 492, W - 72, 492], fill=LINE, width=1)

    readme = [72, 520, 540, 760]
    card(img, readme, WHITE, 14, LINE)
    small_lock = svg_render("tandemesh-lockup-horizontal-ink.svg", 168)
    paste(img, small_lock, (100, 548))
    draw.rounded_rectangle([100, 616, 512, 618], 1, fill=LINE)
    text(draw, (100, 634), "An agent execution plane.", 17, INK, True)
    text(draw, (100, 664), "Route work from Jira, Linear and GitHub to agents", 13, MUTED)
    text(draw, (100, 684), "running on many machines — and write the results back.", 13, MUTED)
    draw.rounded_rectangle([100, 714, 236, 742], 6, fill=(238, 240, 243))
    text(draw, (110, 720), "self-hosted", 11, MUTED)
    text(draw, (72, 776), "README header", 11)

    term = [580, 520, 1048, 760]
    card(img, term, (14, 15, 17), 14, (30, 32, 36))
    draw.rounded_rectangle([580, 520, 1048, 566], 14, fill=(20, 21, 24))
    draw.rectangle([580, 550, 1048, 566], fill=(20, 21, 24))
    dot_x = 600
    for c in ((235, 87, 87), (242, 201, 76), (76, 183, 130)):
        draw.ellipse([dot_x, 536, dot_x + 10, 546], fill=c)
        dot_x += 18
    paste(img, svg_render("tandemesh-tile.svg", 18), (760, 533))
    text(draw, (786, 541), "tandemesh — zsh", 11, (150, 156, 168), anchor="lm")
    text(draw, (600, 596), "~  tandemesh run --repo=acme/api", 13, (200, 205, 214), font_obj=mono(13))
    text(draw, (600, 622), "   queued  run_01HQ…  → agentd dev-1", 13, (120, 200, 160), font_obj=mono(13))
    text(draw, (600, 648), "   running  …", 13, (150, 156, 168), font_obj=mono(13))
    paste(img, svg_render("tandemesh-lockup-horizontal-paper.svg", 212), (600, 664))
    text(draw, (580, 776), "Terminal / dark UI", 11)

    bar = [1088, 520, W - 72, 616]
    card(img, bar, (35, 37, 41), 12, (48, 50, 55))
    for i in range(4):
        x = 1112 + i * 64
        if i == 1:
            paste(img, svg_render("tandemesh-tile.svg", 44), (x, 546))
        else:
            draw.rounded_rectangle([x, 546, x + 44, 590], 11, fill=(56, 59, 65))
    text(draw, (1088, 624), "Taskbar, 44 px", 11)

    avatar = svg_render("tandemesh-tile.svg", 200).resize((160, 160), Image.LANCZOS)
    mask = Image.new("L", (160, 160), 0)
    ImageDraw.Draw(mask).ellipse([0, 0, 159, 159], fill=255)
    circle = Image.new("RGBA", (160, 160), (0, 0, 0, 0))
    circle.paste(avatar, (0, 0), mask)
    paste(img, circle.resize((84, 84), Image.LANCZOS), (1120, 646))
    text(draw, (1216, 656), "Avatar, circular crop", 12)
    text(draw, (1216, 678), "GitHub, Slack, Linear", 12, FAINT)

    paste(img, svg_render("tandemesh-tile.svg", 64), (W - 148, 646))
    text(draw, (W - 148, 722), "OG / 64 px", 11)

    # 3. colour and variants
    text(draw, (72, 812), "PALETTE", 11, MUTED, True)
    text(draw, (486, 812), "MONO + REVERSED", 11, MUTED, True)
    text(draw, (1160, 812), "SMALL SIZES", 11, MUTED, True)
    draw.line([72, 834, W - 72, 834], fill=LINE, width=1)

    chips = [
        ("Ink", "0B0C0E", INK, "primary mark, tile"),
        ("Paper", "F5F6F7", PAPER, "reversed mark"),
        ("Abyss", "14424E", ABYSS, "optional accent"),
    ]
    for i, (name, hexv, rgb, note) in enumerate(chips):
        y = 852 + i * 76
        draw.rounded_rectangle([72, y, 148, y + 60], 10, fill=rgb, outline=(220, 222, 226), width=1)
        text(draw, (166, y + 6), name, 15, INK, True)
        text(draw, (166, y + 30), f"#{hexv}", 12, MUTED, font_obj=mono(12))
        text(draw, (300, y + 30), note, 12, FAINT)

    for i, (bg, label_, svg_name) in enumerate(
        (
            (WHITE, "on paper", "tandemesh-symbol-ink.svg"),
            (PAPER, "on light grey", "tandemesh-symbol-ink.svg"),
            (INK, "on ink", "tandemesh-symbol-reversed.svg"),
        )
    ):
        x = 486 + i * 222
        draw.rounded_rectangle([x, 852, x + 200, 1012], 12, fill=bg, outline=LINE if bg == WHITE else None)
        paste(img, svg_render(svg_name, 108), (x + 46, 892))
        text(draw, (x + 100, 1024), label_, 11, MUTED, anchor="ma")

    x = 1160
    for size, label_ in ((16, "16"), (24, "24"), (32, "32"), (48, "48"), (64, "64")):
        paste(img, svg_render("tandemesh-tile.svg", size), (x, 900))
        text(draw, (x + size // 2, 972), label_, 10, FAINT, anchor="ma")
        x += size + 26
    text(draw, (1160, 996), "16–32 px uses the blunter small-size cut.", 11, MUTED)
    text(draw, (1160, 1014), "Lockup minimum width 120 px; never stack below 96 px.", 11, MUTED)

    # footer
    draw.line([72, H - 96, W - 72, H - 96], fill=LINE, width=1)
    text(
        draw,
        (72, H - 74),
        "Files: design/logo/kit — svg/ (masters), png/, favicon.ico, app.icns, site.webmanifest, guidelines.md",
        12,
    )
    text(
        draw,
        (72, H - 54),
        "Wordmark set in Inter Medium and delivered as outlines. Trademark clearance is still open.",
        12,
        FAINT,
    )

    return img.convert("RGB")


def main() -> None:
    out = ROOT / "renders" / "kit-board.png"
    build().save(out)
    print(f"wrote {out}")


if __name__ == "__main__":
    main()
