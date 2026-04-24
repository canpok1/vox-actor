import { useEffect, useRef, useState } from "react";

interface SilentBadgeProps {
  reason: string;
}

// SilentBadge は無音モードで動作中であることを示すヘッダーバッジ。
// ホバー／タップで reason をそのまま改行込みで表示する独自 popover を持つ。
// ネイティブ title 属性を使わないのは、モバイルのタップ表示・改行・表示タイミング制御のため。
export function SilentBadge({ reason }: SilentBadgeProps) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLSpanElement>(null);

  // popover が開いている間、バッジの外をクリック／タップしたら閉じる（モバイル対応）。
  useEffect(() => {
    if (!open) return;
    const onPointerDown = (e: PointerEvent): void => {
      const target = e.target as Node | null;
      if (rootRef.current && target && !rootRef.current.contains(target)) {
        setOpen(false);
      }
    };
    document.addEventListener("pointerdown", onPointerDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
    };
  }, [open]);

  return (
    <span
      ref={rootRef}
      className="relative inline-flex"
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
    >
      <button
        type="button"
        aria-haspopup="true"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        className="inline-flex cursor-pointer items-center rounded-full bg-ctp-yellow px-2 py-[0.1rem] text-xs font-semibold text-ctp-base"
      >
        🔇 無音モード
      </button>
      {open && reason !== "" && (
        <div
          role="tooltip"
          className="absolute top-full left-0 z-10 mt-1 w-max max-w-[min(90vw,28rem)] whitespace-pre-wrap rounded-md border border-ctp-overlay bg-ctp-mantle p-2 text-xs text-ctp-text shadow-lg"
        >
          {reason}
        </div>
      )}
    </span>
  );
}
