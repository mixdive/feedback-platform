# Mixdive logo assets

Brand assets for the Mixdive feedback portal. Hand-tuned SVG sources plus rasterized PNGs for favicon, app icon, and social use.

## Brand colors

| Role | Hex | Notes |
|------|-----|-------|
| Primary navy | `#0B2545` | Wordmark and the upper two depth bars |
| Teal accent | `#2EC4B6` | Deepest bar — the only place the accent appears |
| White | `#FFFFFF` | Used in inverted variants and inside the favicon plate |

## File index

### Icon (no wordmark)
- `mixdive-icon.svg` — full-color (navy + teal). Default everywhere unless you have a reason to use another variant.
- `mixdive-icon-mono.svg` — single-color (navy only). For print, embossing, single-ink runs.
- `mixdive-icon-white.svg` — single-color (white). For use directly on a navy or photographic background.
- `mixdive-favicon.svg` — icon on a navy rounded-square plate. The browser-tab / app-icon version.

### Lockups
- `mixdive-lockup-horizontal.svg` — primary lockup (icon left, wordmark right). Default for headers, sites, signature.
- `mixdive-lockup-horizontal-white.svg` — same lockup, wordmark in white. Use on dark surfaces.
- `mixdive-lockup-stacked.svg` — icon above the wordmark, both centered. Use in narrow / square layouts.

### Raster exports (in `png/`)
Generated from the SVG sources. Regenerate via `make logos` (see below).

| File | Use |
|------|-----|
| `favicon-16.png`, `favicon-32.png`, `favicon-48.png` | Browser tabs |
| `apple-touch-icon-180.png` | iOS home screen |
| `app-icon-192.png`, `app-icon-512.png` | PWA / Android |
| `social-og-1200x630.png` | OpenGraph / Twitter cards |
| `social-avatar-400.png` | Twitter, GitHub, Discord profile |
| `social-linkedin-300.png` | LinkedIn company logo |

## Construction notes

The icon is three rounded bars stacked vertically. Bar widths grow 20 → 40 → 60 (a 1:2:3 ratio); thickness is 10, gap is 8. The bottom bar — the "deepest" — carries the teal accent. Reads as a depth gauge / sonar marker.

Bar geometry is locked. Don't recolor a different bar with the accent; the metaphor breaks.

The wordmark uses a sans-serif at weight 500 with -1.2 letter-spacing in the horizontal lockup, -0.7 in the stacked. It currently renders as live `<text>` with a system font stack preferring Inter. **Before final brand use, convert text to outlines** in your editor (Figma: select text → right-click → "Outline stroke"; Inkscape: Path → Object to Path) so the SVG is independent of the rendering system's font.

## Usage rules

1. **Don't recolor the icon.** Navy + teal only, except for the explicit `-mono`, `-white`, and `-on-dark` variants.
2. **Don't add effects.** No drop shadows, gradients, glows, or strokes. The bars are flat fills.
3. **Don't squish or stretch.** Keep aspect ratio; scale uniformly.
4. **Minimum sizes.** Horizontal lockup: 80px tall minimum. Icon alone: 16px minimum (favicon plate version recommended below 24px). Stacked lockup: 64px tall minimum.
5. **Clear space.** Reserve clear space equal to one bar height on all sides of the lockup.
6. **Backgrounds.** Use the white/colored-plate variants on navy, photographic, or busy backgrounds — never sit the full-color icon on dark without checking contrast.

## Regenerating PNGs

PNGs were exported from the SVG sources using `rsvg-convert`. To regenerate:

```sh
cd assets/logo/png
rsvg-convert -h 16  ../mixdive-favicon.svg          -o favicon-16.png
rsvg-convert -h 32  ../mixdive-favicon.svg          -o favicon-32.png
rsvg-convert -h 48  ../mixdive-favicon.svg          -o favicon-48.png
rsvg-convert -h 180 ../mixdive-favicon.svg          -o apple-touch-icon-180.png
rsvg-convert -h 192 ../mixdive-favicon.svg          -o app-icon-192.png
rsvg-convert -h 512 ../mixdive-favicon.svg          -o app-icon-512.png
rsvg-convert -h 400 ../mixdive-favicon.svg          -o social-avatar-400.png
rsvg-convert -h 300 ../mixdive-favicon.svg          -o social-linkedin-300.png
# OG card is composed: navy background + centered horizontal lockup
```

Or use ImageMagick (`magick`) / Inkscape (`inkscape --export-type=png`) — any SVG rasterizer.

## Where to use which file

- **Console / Portal React apps** — copy `mixdive-icon.svg` and `mixdive-lockup-horizontal.svg` into `web/console/public` and `web/portal/public`, reference via `<img src="/mixdive-icon.svg" />`.
- **Embedded into the Go binary** — `go:embed` from `assets/logo/` once the repo build pipeline is wired up.
- **README / GitHub social card** — use `social-og-1200x630.png`.
- **`<link rel="icon">`** — point at `mixdive-favicon.svg` (modern browsers prefer SVG); list `favicon-32.png` as a fallback.
