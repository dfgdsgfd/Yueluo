import { useEffect, useState } from "react";

const KEYBOARD_OFFSET_THRESHOLD = 80;
const MAX_KEYBOARD_OFFSET_RATIO = 0.6;
const MOBILE_KEYBOARD_QUERY = "(max-width: 767px) and (pointer: coarse)";

/**
 * Returns the current virtual-keyboard height (in px) on mobile browsers.
 *
 * Uses the Visual Viewport API: when a virtual keyboard opens the
 * `visualViewport.height` shrinks while `window.innerHeight` stays the same.
 * The difference gives us the keyboard height which callers can use to push
 * sticky / fixed elements above the keyboard.
 *
 * Small viewport changes from browser chrome are ignored, and temporary
 * over-large VisualViewport readings are capped so fixed inputs do not jump
 * out of view during keyboard resize.
 *
 * On desktop, disabled callers, or browsers without the API this returns 0.
 */
export function useKeyboardOffset(enabled = true): number {
  const [offset, setOffset] = useState(0);

  useEffect(() => {
    let frame = 0;

    function resetOffset() {
      window.cancelAnimationFrame(frame);
      frame = window.requestAnimationFrame(() => setOffset(0));
    }

    if (!enabled) {
      resetOffset();
      return () => window.cancelAnimationFrame(frame);
    }

    if (!window.matchMedia(MOBILE_KEYBOARD_QUERY).matches) {
      resetOffset();
      return () => window.cancelAnimationFrame(frame);
    }

    const vv = window.visualViewport;
    if (!vv) {
      resetOffset();
      return () => window.cancelAnimationFrame(frame);
    }

    function handleResize() {
      window.cancelAnimationFrame(frame);
      frame = window.requestAnimationFrame(() => {
        if (!window.matchMedia(MOBILE_KEYBOARD_QUERY).matches) {
          setOffset(0);
          return;
        }

        const vv = window.visualViewport;
        if (!vv) {
          setOffset(0);
          return;
        }

        const layoutHeight = Math.max(
          window.innerHeight,
          document.documentElement.clientHeight,
        );
        if (layoutHeight <= 0 || vv.height <= 0) {
          setOffset(0);
          return;
        }

        // `offsetTop` accounts for URL-bar collapse and visual viewport panning,
        // so the remaining gap represents the keyboard covering the bottom edge.
        const rawKeyboardHeight = Math.max(0, layoutHeight - vv.height - vv.offsetTop);
        const keyboardHeight =
          rawKeyboardHeight < KEYBOARD_OFFSET_THRESHOLD
            ? 0
            : Math.min(rawKeyboardHeight, layoutHeight * MAX_KEYBOARD_OFFSET_RATIO);

        setOffset(Math.round(keyboardHeight));
      });
    }

    // Fire once to pick up the initial state (keyboard already open).
    handleResize();

    vv.addEventListener("resize", handleResize);
    vv.addEventListener("scroll", handleResize);
    window.addEventListener("resize", handleResize);
    window.addEventListener("orientationchange", handleResize);

    return () => {
      window.cancelAnimationFrame(frame);
      vv.removeEventListener("resize", handleResize);
      vv.removeEventListener("scroll", handleResize);
      window.removeEventListener("resize", handleResize);
      window.removeEventListener("orientationchange", handleResize);
    };
  }, [enabled]);

  return offset;
}
