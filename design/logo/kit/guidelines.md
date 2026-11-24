# Tandemesh icon — brand guidelines (v1)

## The idea

**A fin breaking a plane.** Orcas hunt in pods: no one shouts orders, each animal takes
a role and hands the work on. That is the product — many machines, one plane, work moving
forward. The mark is one fin: a pod member, not a mascot, and it carries the internal
codename *Orca* without drawing a whale.

One sentence, for the pitch: *a pod of machines running as one.*

## Construction

- Canvas `0 0 256 256`; the ink box is `160 × 176` (x 48–208, y 40–216), so the symbol is
  a touch taller than wide, like a real dorsal fin.
- Two cubic curves and a baseline. Leading edge `M48 216 C66 152 120 84 188 40`; trailing
  edge `C174 110 182 168 208 216`; close on the base.
- The trailing edge is concave and the tip leans ~46° forward; that overhang is what makes
  the shape read as a fin rather than a sail. Do not straighten it.
- Angles are whole degrees, coordinates integers. The SVG audit passes with no findings.

## Colour

| Name  | HEX       | Use                                              |
|-------|-----------|--------------------------------------------------|
| Ink   | `#0B0C0E` | primary mark, tile, text on light                |
| Paper | `#F5F6F7` | reversed mark, text on dark                      |
| Abyss | `#14424E` | optional accent (fills, charts, links); 10.9:1 on white |

Ink on paper is 18.1:1; paper on ink is the same. Abyss is 10.9:1 on white and 10.1:1 on
paper — past AAA for body text. The mark never relies on hue: it is a silhouette, so
greyscale and one-colour printing are the same file.

## Files

```
svg/tandemesh-symbol-ink.svg              master symbol, ink
svg/tandemesh-symbol-paper.svg            symbol, paper
svg/tandemesh-symbol-abyss.svg            symbol, abyss
svg/tandemesh-symbol-reversed.svg         on dark: scaled 0.97 to counter irradiation
svg/tandemesh-symbol-16.svg               16–32 px cut (blunter tip, wider base)
svg/tandemesh-wordmark-ink.svg            wordmark only, outlined
svg/tandemesh-lockup-horizontal-*.svg     symbol + wordmark (ink, paper, abyss)
svg/tandemesh-lockup-stacked-*.svg        stacked (ink, paper)
svg/tandemesh-tile*.svg                   256 app tile (ink, abyss, paper)
png/app-icon-{16..1024}.png               tiles at every size
png/symbol-{64..1024}.png                 transparent symbols
favicon.ico                               16–256, PNG entries
app.icns                                  macOS
web/                                      favicon.ico, favicon.svg, apple-touch-icon,
                                          icon-192, icon-512, icon-maskable-512
site.webmanifest                          PWA manifest
```

## Lockups

- Horizontal: symbol height `96`, wordmark `92` (cap height 66.9), gap `24`, baseline
  aligned to the fin's base. Canvas height 256, width follows.
- Stacked: symbol `126` above the wordmark at `96`, 26 units of air.
- Clear space: `0.5 ×` the symbol's height on all sides. Nothing enters it — no text, no
  rules, no image edges.
- Minimum sizes: symbol `16 px`; horizontal lockup `120 px` wide; never stack a lockup
  below `96 px` wide.
- Small sizes: use `tandemesh-symbol-16.svg` below 32 px. The standard tip thins out at
  16 px; the cut blunts it by raising the tip 18 units and widening the base.

## Wordmark

Set in **Inter Medium**, converted to outlines (`ttf.py` reads the font file directly, so
nothing depends on the font being installed). Tracking `-0.014 em`, with optical pairs for
`Ta`, `an`, `nd`, `de`, `em`, `me`, `es`, `sh`. Inter is licensed under the SIL Open Font
License, which permits logo use; the outlines are baked into the SVG, so no font file ships.

## Misuse

- Do not rotate, italicise, outline, or add a shadow/gradient/glow.
- Do not stretch: scale proportionally from the masters.
- Do not put the ink mark on a dark photograph — use the paper/reversed version.
- Do not put the mark in a second container (a circle inside the app tile, etc.).
- Do not recolour the fin with a gradient or two-tone; it is a silhouette.
- Do not re-space the lockup; regenerate it from `kit.py`.

## Open items

- **Trademark**: an orca fin is a common marine motif. Run a clearance search (EUIPO/USPTO,
  plus a reverse image search) before launch. Nothing here is legal advice.
- **Print**: if the mark is ever screen-printed or embroidered, convert to a true single
  outline and check the tip at the target size (it is the part that disappears first).
- **macOS 26**: `build/appicon.icon` holds the Icon Composer document; the vector was
  swapped for the fin but the fill/shadow presets in `icon.json` are still Apple's defaults.

## Regenerate

```powershell
$py = "C:\Users\<you>\.dsh\dsh-runtimes\dsh-primary-runtime\dependencies\python\python.exe"
& $py design\logo\tools\kit.py             # svg + png + ico + icns + manifest
& $py design\logo\tools\kitboard.py        # renders/kit-board.png
& $py design\logo\tools\install_assets.py  # web/public + apps/tandemesh-desktop/build
```
