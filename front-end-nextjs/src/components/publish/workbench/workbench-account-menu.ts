import type { Dispatch, RefObject, SetStateAction } from "react";
import { useEffect } from "react";

export function useWorkbenchAccountMenuDismiss(
  accountMenuOpen: boolean,
  accountMenuRef: RefObject<HTMLDivElement | null>,
  setAccountMenuOpen: Dispatch<SetStateAction<boolean>>,
) {
  useEffect(() => {
    if (!accountMenuOpen) {
      return;
    }

    function handlePointerDown(event: PointerEvent) {
      const target = event.target;
      if (target instanceof Node && !accountMenuRef.current?.contains(target)) {
        setAccountMenuOpen(false);
      }
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setAccountMenuOpen(false);
      }
    }

    document.addEventListener("pointerdown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("pointerdown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [accountMenuOpen, accountMenuRef, setAccountMenuOpen]);
}
