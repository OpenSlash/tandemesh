"""Copy the kit into the places the product actually uses it.

Web:     web/public/* + the <head> links in web/index.html are handled there.
Desktop: build/appicon.png, build/windows/icon.ico, build/darwin/icons.icns,
         build/ios/icon.png, build/android mipmaps, build/appicon.icon vector.

Run: python design/logo/tools/install_assets.py
"""

from __future__ import annotations

import re
import shutil
from pathlib import Path

import raster
from kit import FIN_H, FIN_W, INK, KIT, fin_path, svg, tile_svg

ROOT = KIT.parent
REPO = ROOT.parent.parent
DESKTOP = REPO / "apps" / "desktop" / "build"
WEB = REPO / "web" / "public"


def web() -> None:
    WEB.mkdir(parents=True, exist_ok=True)
    mapping = {
        "../favicon.ico": "favicon.ico",
        "favicon.svg": "favicon.svg",
        "favicon-16.png": "favicon-16.png",
        "favicon-32.png": "favicon-32.png",
        "apple-touch-icon.png": "apple-touch-icon.png",
        "icon-192.png": "icon-192.png",
        "icon-512.png": "icon-512.png",
        "icon-maskable-512.png": "icon-maskable-512.png",
        "../site.webmanifest": "site.webmanifest",
    }
    for src, dst in mapping.items():
        shutil.copyfile((KIT / "web" / src).resolve(), WEB / dst)
        print(f"web/public/{dst}")


def desktop() -> None:
    # Wails uses appicon.png as the source for generated platform icons
    (DESKTOP / "appicon.png").write_bytes((KIT / "png" / "app-icon-1024.png").read_bytes())
    (DESKTOP / "windows" / "icon.ico").write_bytes((KIT / "favicon.ico").read_bytes())
    (DESKTOP / "darwin" / "icons.icns").write_bytes((KIT / "app.icns").read_bytes())
    (DESKTOP / "ios" / "icon.png").write_bytes((KIT / "png" / "app-icon-1024.png").read_bytes())

    for density, size in (("mdpi", 48), ("hdpi", 72), ("xhdpi", 96), ("xxhdpi", 144), ("xxxhdpi", 192)):
        res = DESKTOP / "android" / "app" / "src" / "main" / "res" / f"mipmap-{density}"
        if res.exists():
            for name in ("ic_launcher.png", "ic_launcher_round.png"):
                (res / name).write_bytes(raster_bytes(tile_svg(), size))
    for path in [DESKTOP / "appicon.png", DESKTOP / "windows" / "icon.ico", DESKTOP / "darwin" / "icons.icns"]:
        print(f"apps/desktop/{path.relative_to(DESKTOP).as_posix()}")

    # macOS 26 Icon Composer template: keep the viewBox, swap the silhouette
    vector = DESKTOP / "appicon.icon" / "Assets" / "wails_icon_vector.svg"
    if vector.exists():
        w, h = 583, 533
        s = 0.86 * h / FIN_H
        tx = (w - FIN_W * s) / 2 - 48 * s
        ty = (h - FIN_H * s) / 2 - 40 * s
        vector.write_text(
            '<?xml version="1.0" encoding="UTF-8" standalone="no"?>\n'
            f'<svg width="100%" height="100%" viewBox="0 0 {w} {h}" version="1.1" '
            'xmlns="http://www.w3.org/2000/svg">\n'
            f'  <g transform="translate({tx:.3f},{ty:.3f}) scale({s:.5f})">\n'
            f'    <path d="{fin_path()}"/>\n  </g>\n</svg>\n',
            encoding="utf-8",
        )
        print("apps/desktop/build/appicon.icon/Assets/wails_icon_vector.svg")


def raster_bytes(svg_text: str, size: int) -> bytes:
    from io import BytesIO

    buf = BytesIO()
    raster.render_svg(svg_text, size).save(buf, format="PNG")
    return buf.getvalue()


def head_links() -> None:
    index = REPO / "web" / "index.html"
    html = index.read_text(encoding="utf-8")
    if 'rel="icon"' in html:
        print("web/index.html already wired")
        return
    block = (
        '    <link rel="icon" href="/favicon.ico" sizes="32x32" />\n'
        '    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />\n'
        '    <link rel="apple-touch-icon" href="/apple-touch-icon.png" />\n'
        '    <link rel="manifest" href="/site.webmanifest" />\n'
        '    <meta name="theme-color" content="#0B0C0E" />\n'
    )
    html = html.replace('    <title>Tandemesh</title>\n', block + '    <title>Tandemesh</title>\n')
    index.write_text(html, encoding="utf-8")
    print("web/index.html head updated")


if __name__ == "__main__":
    web()
    desktop()
    head_links()
