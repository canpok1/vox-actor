import { useEffect, useRef, useState } from "react";

// Dynamic threshold parameters
const WINDOW_MS = 500;
const THRESHOLD_RATIO = 0.5;
const MIN_THRESHOLD = 0.01;

interface LipSyncImageProps {
  mouthClosedUrl: string;
  mouthOpenUrl: string;
  volume: number;
  hysteresisMs?: number;
}

/**
 * LipSyncImage は音量に応じて口閉/口開の2枚の画像を切り替える。
 * 動的閾値（envelope follower）を使用して、音節単位での開閉を実現する。
 * - hysteresisMs: 開→閉切替時のヒステリシス時間（デフォルト 80ms）
 */
export function LipSyncImage({
  mouthClosedUrl,
  mouthOpenUrl,
  volume,
  hysteresisMs = 80,
}: LipSyncImageProps) {
  const [isOpen, setIsOpen] = useState(false);
  const closeTimerRef = useRef<NodeJS.Timeout | null>(null);
  const recentSamplesRef = useRef<Array<{ time: number; value: number }>>([]);

  useEffect(() => {
    const now = performance.now();

    // Ring buffer: 直近 WINDOW_MS のボリュームサンプルをトラッキング
    recentSamplesRef.current.push({ time: now, value: volume });
    recentSamplesRef.current = recentSamplesRef.current.filter(
      (s) => now - s.time <= WINDOW_MS,
    );

    // 動的閾値: 直近のピークに係数を掛ける
    const recentMax =
      recentSamplesRef.current.length > 0
        ? Math.max(...recentSamplesRef.current.map((s) => s.value))
        : 0;
    const dynamicThreshold = Math.max(
      MIN_THRESHOLD,
      recentMax * THRESHOLD_RATIO,
    );

    // 音量が動的閾値を超えたら口開
    if (volume >= dynamicThreshold) {
      if (closeTimerRef.current) {
        clearTimeout(closeTimerRef.current);
        closeTimerRef.current = null;
      }
      setIsOpen(true);
    } else {
      // 音量が動的閾値以下になったらヒステリシス時間待ってから口閉
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
  }, [volume, hysteresisMs]);

  return (
    <div className="relative h-full w-full overflow-hidden">
      <img
        src={mouthClosedUrl}
        alt="character mouth closed"
        className="absolute inset-0 h-full w-full object-contain"
        style={{ display: isOpen ? "none" : "block" }}
      />
      <img
        src={mouthOpenUrl}
        alt="character mouth open"
        className="absolute inset-0 h-full w-full object-contain"
        style={{ display: isOpen ? "block" : "none" }}
      />
    </div>
  );
}
