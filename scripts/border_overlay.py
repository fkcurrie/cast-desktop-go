#!/usr/bin/env python3
"""
Cast Desktop - Visual Privacy Screen Border Overlay
Draws a subtle glowing border around the captured screen area while casting.
Click-through and always-on-top.
"""

import argparse
import signal
import sys
import gi

gi.require_version("Gtk", "3.0")
gi.require_version("Gdk", "3.0")
from gi.repository import Gtk, Gdk, GLib
import cairo


class PrivacyBorderWindow(Gtk.Window):
    def __init__(self, x=0, y=0, width=1920, height=1080, r=0.0, g=0.9, b=1.0, border_width=4):
        super().__init__(type=Gtk.WindowType.POPUP)

        self.border_x = x
        self.border_y = y
        self.border_w = width
        self.border_h = height
        self.r = r
        self.g = g
        self.b = b
        self.border_width = border_width

        self.set_title("Cast Desktop Privacy Border")
        self.set_app_paintable(True)
        self.set_decorated(False)
        self.set_keep_above(True)
        self.set_accept_focus(False)

        screen = self.get_screen()
        visual = screen.get_rgba_visual()
        if visual:
            self.set_visual(visual)

        self.move(self.border_x, self.border_y)
        self.set_default_size(self.border_w, self.border_h)

        self.connect("draw", self.on_draw)
        self.connect("realize", self.on_realize)

    def on_realize(self, widget):
        # Empty input shape region makes the entire window click-through
        window = self.get_window()
        if window:
            empty_region = cairo.Region()
            window.input_shape_combine_region(empty_region, 0, 0)

    def on_draw(self, widget, cr):
        cr.set_source_rgba(0, 0, 0, 0)
        cr.set_operator(cairo.OPERATOR_SOURCE)
        cr.paint()

        cr.set_operator(cairo.OPERATOR_OVER)
        w = widget.get_allocated_width()
        h = widget.get_allocated_height()
        bw = self.border_width

        # Outer subtle glow
        cr.set_source_rgba(self.r, self.g, self.b, 0.45)
        cr.set_line_width(bw + 2)
        cr.rectangle(1, 1, w - 2, h - 2)
        cr.stroke()

        # Crisp inner border
        cr.set_source_rgba(self.r, self.g, self.b, 0.95)
        cr.set_line_width(bw)
        cr.rectangle(bw / 2, bw / 2, w - bw, h - bw)
        cr.stroke()

        return False


def main():
    parser = argparse.ArgumentParser(description="Cast Desktop Privacy Border Overlay")
    parser.add_argument("--x", type=int, default=0, help="Window X offset")
    parser.add_argument("--y", type=int, default=0, help="Window Y offset")
    parser.add_argument("--width", type=int, default=0, help="Border width")
    parser.add_argument("--height", type=int, default=0, help="Border height")
    args = parser.parse_args()

    screen = Gdk.Screen.get_default()
    w = args.width if args.width > 0 else (screen.get_width() if screen else 1920)
    h = args.height if args.height > 0 else (screen.get_height() if screen else 1080)

    win = PrivacyBorderWindow(x=args.x, y=args.y, width=w, height=h)
    win.show_all()

    def handle_signal(sig, frame):
        Gtk.main_quit()
        sys.exit(0)

    signal.signal(signal.SIGTERM, handle_signal)
    signal.signal(signal.SIGINT, handle_signal)

    # Periodic timer to allow Python signal handlers to run
    GLib.timeout_add(500, lambda: True)

    Gtk.main()


if __name__ == "__main__":
    main()
