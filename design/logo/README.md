# Tandemesh icon

The mark is **a fin breaking a plane** — a pod of machines running as one. It became the
product icon in round 1 of the logo-design process; this folder holds the concepts it beat,
the kit that ships, and the tooling that regenerates both.

```
concepts/          round 1 shortlist (A fin, B shear, C stream) + every rejected iteration
kit/               the shipped kit: svg masters, png set, favicon.ico, app.icns,
                   site.webmanifest, guidelines.md
renders/           concepts-round1.png, kit-board.png, stress.png and test sheets
tools/             all the code that draws, renders, audits and exports
```

## What ships

`kit/` is the source of truth for the icon. It is already wired into the product:

- `web/public/*` — favicon.ico, favicon.svg, apple-touch-icon, PWA icons and manifest,
  linked from `web/index.html`
- `apps/desktop/build/` — `appicon.png` (Wails source), `windows/icon.ico`,
  `darwin/icons.icns`, `ios/icon.png`, Android mipmaps, and the macOS 26 Icon Composer
  vector

Read [`kit/guidelines.md`](kit/guidelines.md) before using the mark anywhere new.

## Regenerate

```powershell
$py = "C:\Users\<you>\.dsh\dsh-runtimes\dsh-primary-runtime\dependencies\python\python.exe"

& $py design\logo\tools\kit.py             # svg + png + ico + icns + manifest
& $py design\logo\tools\kitboard.py        # renders/kit-board.png
& $py design\logo\tools\install_assets.py  # web/public + apps/desktop/build
& $py design\logo\tools\audit.py design\logo\kit\svg\*.svg
```

Concept work: `gen_final.py` writes the three symbols, `sheet.py` draws the comparison,
size-ladder, stress and concept boards.

## Tooling notes

The skill ships `scripts/render_png.py`, `svg_audit.py`, `preview_sheet.py` and
`concept_sheet.py`. This sandbox has no outbound network from the shell and Chromium cannot
start (its IPC needs named pipes, which are blocked), so those scripts could not be fetched
or run. `tools/` covers the same ground:

- `raster.py` — SVG → PNG with a real winding-rule scanline fill (paths, rects, groups,
  translate/scale, 4× supersampling). Fills only; every mark here is a filled path.
- `audit.py` — live text, rasters, filters/masks, colour count, near-miss angles, hairlines.
- `sheet.py` — compare, true-size ladder, mirror/rotation/blur stress tests, concept board.
- `ttf.py` — reads Inter's TrueType outlines so the wordmark ships as paths, not a font.
- `kit.py` / `kitboard.py` / `install_assets.py` — kit generation, board, installation.

Not done: trademark clearance, and a print proof if the mark is ever screen-printed.
