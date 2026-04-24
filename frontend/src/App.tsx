import { useCallback, useEffect, useRef, useState } from "react";
import { SilentBadge } from "./components/SilentBadge";
import { StatusBadge } from "./components/StatusBadge";
import { StreamPanel } from "./components/StreamPanel";
import { type TabName, Tabs } from "./components/Tabs";
import { TestPanel } from "./components/TestPanel";
import { VolumeControls } from "./components/VolumeControls";
import { useEventSource } from "./hooks/useEventSource";
import { usePersistedState } from "./hooks/usePersistedState";
import { usePlaybackQueue } from "./hooks/usePlaybackQueue";
import {
  isApiStatus,
  isClipEvent,
  isErrorEventPayload,
  type Speaker,
  type TimelineEntry,
} from "./types/api";

const STORAGE_KEYS = {
  historySize: "vox-actor.stream.historySize",
  volume: "vox-actor.stream.volume",
  activeTab: "vox-actor.stream.activeTab",
  testSpeakerId: "vox-actor.stream.testSpeakerId",
  showSpeakerName: "vox-actor.stream.showSpeakerName",
  showStyleName: "vox-actor.stream.showStyleName",
  showTimestamp: "vox-actor.stream.showTimestamp",
} as const;

const DEFAULT_HISTORY_SIZE = 20;
const DEFAULT_VOLUME = 50;
const HISTORY_SIZE_OPTIONS: readonly number[] = [10, 20, 30, 50, 100, 200];
const TEST_ERROR_DISPLAY_MS = 4000;

const parseTab = (raw: string): TabName | null =>
  raw === "stream" || raw === "test" ? raw : null;

const parseHistorySize = (raw: string): number | null => {
  const n = parseInt(raw, 10);
  return Number.isFinite(n) && HISTORY_SIZE_OPTIONS.includes(n) ? n : null;
};

const parseVolume = (raw: string): number | null => {
  const n = parseInt(raw, 10);
  return Number.isInteger(n) && n >= 0 && n <= 100 ? n : null;
};

const parseBool = (raw: string): boolean | null =>
  raw === "true" ? true : raw === "false" ? false : null;

const passthrough = (v: string): string => v;

// 履歴件数上限に合わせて古い項目から削除する。
// 再生中のクリップ項目（kind:"clip" かつ id が playingId と一致）は削除しない。
// エラー項目は再生対象ではないため、常に削除対象となる。
function trimTimeline(
  entries: TimelineEntry[],
  size: number,
  playingId: number | null,
): TimelineEntry[] {
  let next = entries;
  while (next.length > size) {
    const head = next[0];
    if (head.kind === "clip" && head.id === playingId) break;
    next = next.slice(1);
  }
  return next;
}

export function App() {
  const audioRef = useRef<HTMLAudioElement>(null);

  const [activeTab, setActiveTab] = usePersistedState<TabName>(
    STORAGE_KEYS.activeTab,
    "stream",
    parseTab,
    passthrough,
  );
  const [historySize, setHistorySize] = usePersistedState<number>(
    STORAGE_KEYS.historySize,
    DEFAULT_HISTORY_SIZE,
    parseHistorySize,
    String,
  );
  const [volume, setVolume] = usePersistedState<number>(
    STORAGE_KEYS.volume,
    DEFAULT_VOLUME,
    parseVolume,
    String,
  );
  const [muted, setMuted] = useState(true);
  const [showSpeakerName, setShowSpeakerName] = usePersistedState(
    STORAGE_KEYS.showSpeakerName,
    true,
    parseBool,
    String,
  );
  const [showStyleName, setShowStyleName] = usePersistedState(
    STORAGE_KEYS.showStyleName,
    true,
    parseBool,
    String,
  );
  const [showTimestamp, setShowTimestamp] = usePersistedState(
    STORAGE_KEYS.showTimestamp,
    true,
    parseBool,
    String,
  );
  const [testSpeakerId, setTestSpeakerId] = usePersistedState<string>(
    STORAGE_KEYS.testSpeakerId,
    "",
    passthrough,
    passthrough,
  );

  const [speakers, setSpeakers] = useState<Speaker[]>([]);
  const [silent, setSilent] = useState(false);
  const [silentReason, setSilentReason] = useState("");
  const [entries, setEntries] = useState<TimelineEntry[]>([]);
  const [testError, setTestError] = useState<string>("");
  const testErrorTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const { playingClipId, enqueue } = usePlaybackQueue(
    audioRef,
    activeTab === "stream",
  );
  const playingClipIdRef = useRef<number | null>(null);
  playingClipIdRef.current = playingClipId;

  useEffect(() => {
    const audio = audioRef.current;
    if (audio) audio.volume = volume / 100;
  }, [volume]);

  useEffect(() => {
    const audio = audioRef.current;
    if (audio) audio.muted = muted;
  }, [muted]);

  // 履歴上限の変更時のみトリム。再生状態の変化ではトリムしない
  // （再生終了直後の playingClipId=null 経由で過去のハイライト対象が消えるのを防ぐ）。
  useEffect(() => {
    setEntries((prev) => trimTimeline(prev, historySize, playingClipIdRef.current));
  }, [historySize]);

  const showTestError = useCallback((msg: string): void => {
    setTestError(msg);
    if (testErrorTimerRef.current !== null) {
      clearTimeout(testErrorTimerRef.current);
    }
    testErrorTimerRef.current = setTimeout(() => {
      setTestError("");
      testErrorTimerRef.current = null;
    }, TEST_ERROR_DISPLAY_MS);
  }, []);

  useEffect(() => {
    return () => {
      if (testErrorTimerRef.current !== null) {
        clearTimeout(testErrorTimerRef.current);
        testErrorTimerRef.current = null;
      }
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    const load = async (): Promise<void> => {
      try {
        const resp = await fetch("/api/status");
        if (!resp.ok) throw new Error(`status ${resp.status}`);
        const data: unknown = await resp.json();
        if (!isApiStatus(data)) {
          throw new Error("invalid api status payload");
        }
        if (cancelled) return;
        setSpeakers(data.speakers);
        setSilent(data.silent);
        setSilentReason(data.silentReason);
      } catch (err) {
        console.error("failed to load api status", err);
        if (!cancelled) showTestError("サーバー状態の取得に失敗しました");
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [showTestError]);

  useEffect(() => {
    if (speakers.length === 0) return;
    const exists = speakers.some((s) => String(s.id) === testSpeakerId);
    if (!exists) {
      setTestSpeakerId(String(speakers[0].id));
    }
  }, [speakers, testSpeakerId, setTestSpeakerId]);

  // 無音モード時は「音声テスト」タブを非表示にするため、選択中だった場合は配信タブに戻す。
  useEffect(() => {
    if (silent && activeTab === "test") {
      setActiveTab("stream");
    }
  }, [silent, activeTab, setActiveTab]);

  const handleClip = useCallback(
    (data: string): void => {
      let parsed: unknown;
      try {
        parsed = JSON.parse(data);
      } catch (err) {
        console.error("invalid clip payload", err);
        return;
      }
      if (!isClipEvent(parsed)) {
        console.error("invalid clip payload", parsed);
        return;
      }
      const clip = parsed;
      setEntries((prev) =>
        trimTimeline(
          [...prev, { kind: "clip", ...clip }],
          historySize,
          playingClipIdRef.current,
        ),
      );
      // clip.url が空の場合は無音モード配信なのでタイムラインには載せつつ再生キューには積まない。
      if (clip.url !== "") {
        enqueue(clip);
      }
    },
    [historySize, enqueue],
  );

  const handleServerError = useCallback(
    (data: string): void => {
      let parsed: unknown;
      try {
        parsed = JSON.parse(data);
      } catch (err) {
        console.error("invalid error payload", err);
        return;
      }
      if (!isErrorEventPayload(parsed)) {
        console.error("invalid error payload", parsed);
        return;
      }
      setEntries((prev) =>
        trimTimeline(
          [...prev, { kind: "error", ...parsed }],
          historySize,
          playingClipIdRef.current,
        ),
      );
    },
    [historySize],
  );

  const connected = useEventSource("/events", handleClip, handleServerError);

  const handleTestPlay = useCallback((): void => {
    if (!testSpeakerId) {
      showTestError("話者が選択されていません");
      return;
    }
    setTestError("");
    const audio = audioRef.current;
    if (!audio) return;
    audio.pause();
    audio.src = `/test-clip?speaker=${encodeURIComponent(testSpeakerId)}`;
    audio.play().catch((err: unknown) => {
      console.error("test play failed", err);
      showTestError("合成に失敗しました");
    });
  }, [testSpeakerId, showTestError]);

  return (
    <main className="mx-auto my-2 max-w-[1200px] rounded-md bg-ctp-surface p-3 sm:my-4 sm:p-4 md:my-8 md:rounded-lg md:p-6">
      <div className="mb-2 flex flex-wrap items-center gap-3">
        <h1 className="m-0 text-[1.1rem] md:text-[1.4rem]">vox-actor stream</h1>
        <StatusBadge connected={connected} />
        {silent && <SilentBadge reason={silentReason} />}
      </div>
      <VolumeControls
        volume={volume}
        muted={muted}
        onVolumeChange={setVolume}
        onMuteChange={setMuted}
      />
      <Tabs active={activeTab} onChange={setActiveTab} hideTest={silent} />
      <StreamPanel
        hidden={activeTab !== "stream"}
        entries={entries}
        playingClipId={playingClipId}
        historySize={historySize}
        historySizeOptions={HISTORY_SIZE_OPTIONS}
        onHistorySizeChange={setHistorySize}
        showSpeakerName={showSpeakerName}
        showStyleName={showStyleName}
        showTimestamp={showTimestamp}
        onShowSpeakerNameChange={setShowSpeakerName}
        onShowStyleNameChange={setShowStyleName}
        onShowTimestampChange={setShowTimestamp}
      />
      {!silent && (
        <TestPanel
          hidden={activeTab !== "test"}
          speakers={speakers}
          selectedSpeakerId={testSpeakerId}
          onSpeakerChange={setTestSpeakerId}
          onPlay={handleTestPlay}
          error={testError}
        />
      )}
      <audio ref={audioRef} muted />
    </main>
  );
}
