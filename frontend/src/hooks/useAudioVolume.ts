import { useEffect, useRef, useState } from "react";

/**
 * useAudioVolume は audio 要素から Web Audio API で RMS 音量を取得する hook。
 * AudioContext は初回ユーザー操作後に lazy 生成される（autoplay policy 対策）。
 * ミュート中も音量解析が継続する。
 */
export function useAudioVolume(
  audioRef: React.RefObject<HTMLAudioElement>,
  enabled: boolean = true,
): number {
  const [volume, setVolume] = useState(0);
  const audioContextRef = useRef<AudioContext | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const dataArrayRef = useRef<Uint8Array | null>(null);
  const animationIdRef = useRef<number | null>(null);

  useEffect(() => {
    if (!enabled || !audioRef.current) {
      return;
    }

    // lazy init: AudioContext は初回ユーザー操作後に作成
    const initAudioContext = () => {
      if (audioContextRef.current || !audioRef.current) return;

      const AudioContextConstructor = (window.AudioContext ||
        (window as unknown as { webkitAudioContext: typeof AudioContext })
          .webkitAudioContext) as typeof AudioContext;
      const audioContext = new AudioContextConstructor();
      audioContext.resume();

      const analyser = audioContext.createAnalyser();
      analyser.fftSize = 512;

      const source = audioContext.createMediaElementSource(audioRef.current);
      source.connect(analyser);
      analyser.connect(audioContext.destination);

      audioContextRef.current = audioContext;
      analyserRef.current = analyser;
      dataArrayRef.current = new Uint8Array(analyser.frequencyBinCount);

      updateVolume();
    };

    const updateVolume = () => {
      if (!analyserRef.current || !dataArrayRef.current) {
        animationIdRef.current = requestAnimationFrame(updateVolume);
        return;
      }

      analyserRef.current.getByteTimeDomainData(
        dataArrayRef.current as Uint8Array<ArrayBuffer>,
      );

      // RMS (Root Mean Square) 計算
      let sum = 0;
      for (let i = 0; i < dataArrayRef.current.length; i++) {
        const normalized = (dataArrayRef.current[i] - 128) / 128;
        sum += normalized * normalized;
      }
      const rms = Math.sqrt(sum / dataArrayRef.current.length);

      setVolume(rms);
      animationIdRef.current = requestAnimationFrame(updateVolume);
    };

    // 初回ユーザー操作後に AudioContext を作成
    const handleUserInteraction = () => {
      initAudioContext();
      document.removeEventListener("click", handleUserInteraction);
      document.removeEventListener("touchstart", handleUserInteraction);
    };

    // 既に再生中の場合は即座に初期化
    if (audioRef.current.readyState > 0) {
      initAudioContext();
    } else {
      document.addEventListener("click", handleUserInteraction);
      document.addEventListener("touchstart", handleUserInteraction);
    }

    return () => {
      document.removeEventListener("click", handleUserInteraction);
      document.removeEventListener("touchstart", handleUserInteraction);
      if (animationIdRef.current) {
        cancelAnimationFrame(animationIdRef.current);
      }
    };
  }, [enabled, audioRef]);

  return volume;
}
