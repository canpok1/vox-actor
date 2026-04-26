import { useEffect, useRef, useState } from "react";

interface LipSyncImageProps {
  mouthClosedUrl: string;
  mouthOpenUrl: string;
  volume: number;
  threshold?: number;
  hysteresisMs?: number;
}

/**
 * LipSyncImage は音量に応じて口閉/口開の2枚の画像を切り替える。
 * - threshold: 開閉の音量閾値（デフォルト 0.02）
 * - hysteresisMs: 開→閉切替時のヒステリシス時間（デフォルト 80ms）
 */
export function LipSyncImage({
  mouthClosedUrl,
  mouthOpenUrl,
  volume,
  threshold = 0.02,
  hysteresisMs = 80,
}: LipSyncImageProps) {
  const [isOpen, setIsOpen] = useState(false);
  const closeTimerRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    // 音量が閾値を超えたら口開
    if (volume >= threshold) {
      if (closeTimerRef.current) {
        clearTimeout(closeTimerRef.current);
        closeTimerRef.current = null;
      }
      setIsOpen(true);
    } else {
      // 音量が閾値以下になったらヒステリシス時間待ってから口閉
      if (!closeTimerRef.current) {
        closeTimerRef.current = setTimeout(() => {
          setIsOpen(false);
          closeTimerRef.current = null;
        }, hysteresisMs);
      }
    }

    return () => {
      if (closeTimerRef.current) {
        clearTimeout(closeTimerRef.current);
        closeTimerRef.current = null;
      }
    };
  }, [volume, threshold, hysteresisMs]);

  return (
    <div className="relative w-full overflow-hidden" style={{ aspectRatio: "3 / 4" }}>
      <img
        src={mouthClosedUrl}
        alt="character mouth closed"
        className="absolute inset-0 w-full h-full object-contain"
        style={{ display: isOpen ? "none" : "block" }}
      />
      <img
        src={mouthOpenUrl}
        alt="character mouth open"
        className="absolute inset-0 w-full h-full object-contain"
        style={{ display: isOpen ? "block" : "none" }}
      />
    </div>
  );
}
