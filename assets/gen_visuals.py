#!/usr/bin/env python3
"""Generate gailwind hero visual (GitHub dark palette, no-cringe flat design)."""
from PIL import Image, ImageDraw, ImageFont

BG = (13, 17, 23)          # #0d1117
FG = (230, 237, 243)       # #e6edf3
MUTED = (139, 148, 158)    # #8b949e
GO = (0, 173, 216)         # Go cyan #00ADD8
TW = (56, 189, 248)        # Tailwind sky #38bdf8
BORDER = (48, 54, 61)      # #30363d

SANS = "/usr/share/fonts/truetype/noto/NotoSans-Bold.ttf"
SANS_R = "/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf"
MONO = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"


def font(path, size):
    return ImageFont.truetype(path, size)


def draw_wind(d, cx, cy, s, color):
    """Three flat wind strokes (Tailwind-like), geometric, no curves."""
    lw = max(8, int(s * 0.07))
    # top stroke with hook
    d.line([cx - s * 0.45, cy - s * 0.28, cx + s * 0.25, cy - s * 0.28], fill=color, width=lw)
    d.arc([cx + s * 0.05, cy - s * 0.48, cx + s * 0.45, cy - s * 0.08], 270, 90, fill=color, width=lw)
    # middle stroke, longer
    d.line([cx - s * 0.45, cy, cx + s * 0.40, cy], fill=color, width=lw)
    d.arc([cx + s * 0.20, cy - s * 0.20, cx + s * 0.60, cy + s * 0.20], 270, 90, fill=color, width=lw)
    # bottom stroke with downward hook
    d.line([cx - s * 0.45, cy + s * 0.28, cx + s * 0.15, cy + s * 0.28], fill=color, width=lw)
    d.arc([cx - s * 0.05, cy + s * 0.08, cx + s * 0.35, cy + s * 0.48], 270, 90, fill=color, width=lw)


def hero():
    W, H = 1280, 640
    img = Image.new("RGB", (W, H), BG)
    d = ImageDraw.Draw(img)
    for x in range(0, W, 64):
        d.line([x, 0, x, H], fill=(20, 25, 32))
    for y in range(0, H, 64):
        d.line([0, y, W, y], fill=(20, 25, 32))
    draw_wind(d, 1100, 265, 280, TW)
    d.text((90, 210), "gailwind", font=font(SANS, 120), fill=FG)
    d.text((94, 370), "Tailwind utility CSS, compiled in Go.", font=font(SANS_R, 44), fill=MUTED)
    d.text((94, 500), "$ go install github.com/ofershap/gailwind/cmd/gailwind@latest",
           font=font(MONO, 28), fill=GO)
    img.save("assets/hero.png")


hero()
