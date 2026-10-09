#!/usr/bin/env python3
import os
import gi
gi.require_version("Rsvg", "2.0")
from gi.repository import Rsvg
import cairo

SVG_CONTENT = """<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" width="512" height="512">
  <defs>
    <radialGradient id="screenGrad" cx="50%" cy="30%" r="80%">
      <stop offset="0%" stop-color="#1c2438" />
      <stop offset="100%" stop-color="#0b0e17" />
    </radialGradient>
    <linearGradient id="bezelGrad" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" stop-color="#3c4962" />
      <stop offset="100%" stop-color="#182030" />
    </linearGradient>
    <linearGradient id="castGrad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#00f0ff" />
      <stop offset="45%" stop-color="#2979ff" />
      <stop offset="100%" stop-color="#7c4dff" />
    </linearGradient>
    <linearGradient id="castGradSubtle" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#2979ff" />
      <stop offset="100%" stop-color="#651fff" />
    </linearGradient>
    <linearGradient id="standGrad" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" stop-color="#4a5568" />
      <stop offset="100%" stop-color="#2d3748" />
    </linearGradient>
    <linearGradient id="borderGlow" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#2979ff" />
      <stop offset="100%" stop-color="#00e5ff" />
    </linearGradient>
  </defs>

  <!-- Monitor Stand Base -->
  <path d="M 152 444 L 360 444 C 370 444 378 452 376 462 L 370 476 C 368 482 362 486 354 486 L 158 486 C 150 486 144 482 142 476 L 136 462 C 134 452 142 444 152 444 Z" fill="url(#standGrad)" />
  <!-- Monitor Neck -->
  <polygon points="228,348 284,348 296,448 216,448" fill="#141a26" />
  <polygon points="234,348 278,348 288,446 224,446" fill="#2d3748" />

  <!-- Monitor Outer Bezel -->
  <rect x="28" y="40" width="456" height="312" rx="28" ry="28" fill="url(#bezelGrad)" stroke="#4a5a78" stroke-width="4" />

  <!-- Inner Display Screen -->
  <rect x="44" y="56" width="424" height="268" rx="16" ry="16" fill="url(#screenGrad)" stroke="#1a2233" stroke-width="3" />

  <!-- Screen Content: Cast Broadcast Waves & Screen -->
  <g transform="translate(100, 84)">
    <!-- Screen Outline Top-Right -->
    <path d="M 170 36 L 246 36 C 258 36 266 44 266 56 L 266 156 C 266 168 258 176 246 176 L 176 176" 
          fill="none" stroke="url(#castGradSubtle)" stroke-width="16" stroke-linecap="round" stroke-linejoin="round" />
    <path d="M 36 36 L 114 36" 
          fill="none" stroke="url(#castGradSubtle)" stroke-width="16" stroke-linecap="round" />

    <!-- Cast Waves -->
    <!-- Wave 2 (Outer) -->
    <path d="M 36 84 A 112 112 0 0 1 148 196" 
          fill="none" stroke="url(#castGrad)" stroke-width="18" stroke-linecap="round" />
    <!-- Wave 1 (Inner) -->
    <path d="M 36 136 A 60 60 0 0 1 96 196" 
          fill="none" stroke="url(#castGrad)" stroke-width="18" stroke-linecap="round" />
    <!-- Center Dot -->
    <circle cx="36" cy="196" r="20" fill="url(#castGrad)" />
  </g>

  <!-- Power Status LED (Cyan Glow) -->
  <circle cx="256" cy="336" r="3.5" fill="#00f0ff" />
</svg>
"""

def generate_icons(base_dir):
    os.makedirs(os.path.join(base_dir, "scalable"), exist_ok=True)
    svg_path = os.path.join(base_dir, "scalable", "cast-desktop.svg")
    with open(svg_path, "w") as f:
        f.write(SVG_CONTENT.strip())
    print(f"Written SVG: {svg_path}")

    handle = Rsvg.Handle.new_from_data(SVG_CONTENT.encode("utf-8"))

    sizes = [16, 24, 32, 48, 64, 128, 256, 512]
    for size in sizes:
        size_dir = os.path.join(base_dir, f"{size}x{size}")
        os.makedirs(size_dir, exist_ok=True)
        png_path = os.path.join(size_dir, "cast-desktop.png")

        surf = cairo.ImageSurface(cairo.FORMAT_ARGB32, size, size)
        ctx = cairo.Context(surf)
        rect = Rsvg.Rectangle()
        rect.x = 0
        rect.y = 0
        rect.width = size
        rect.height = size
        handle.render_document(ctx, rect)
        surf.write_to_png(png_path)
        print(f"Generated {size}x{size} PNG: {png_path}")

if __name__ == "__main__":
    out_dir = os.path.abspath("assets/icons")
    generate_icons(out_dir)
