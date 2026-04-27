// @ts-nocheck - renderHook type inference limitation with rerender and playingClipId
import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { CharacterEntry, TimelineEntry } from "../types/api";
import { QUEUE_EMPTY_MS, useCharacterStage } from "./useCharacterStage";

function makeChar(speakerName: string, styleName = "ノーマル"): CharacterEntry {
  return {
    speakerName,
    styleName,
    mouthClosed: `${speakerName}-closed.png`,
    mouthOpen: `${speakerName}-open.png`,
  };
}

function makeClipEntry(
  id: number,
  speakerName: string,
  styleName = "ノーマル",
): TimelineEntry {
  return {
    kind: "clip",
    id,
    url: `http://example.com/${id}.wav`,
    text: `text ${id}`,
    speakerName,
    styleName,
    timestamp: id * 1000,
  };
}

describe("useCharacterStage", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it("初期状態ではスロットがすべてnullでisMultiSlot=false", () => {
    const { result } = renderHook(() => useCharacterStage(null, [], []));
    expect(result.current.slots).toEqual([null, null, null, null]);
    expect(result.current.isMultiSlot).toBe(false);
  });

  it("1人目の発話でinner-right(slot 0)に配置される", async () => {
    const charA = makeChar("キャラA");
    const entry1 = makeClipEntry(1, "キャラA");

    const { result, rerender } = renderHook(
      ({
        playingClipId,
        entries,
        characters,
      }: {
        playingClipId: number | null;
        entries: TimelineEntry[];
        characters: CharacterEntry[];
      }) => useCharacterStage(playingClipId, entries, characters),
      {
        initialProps: {
          playingClipId: null,
          entries: [entry1],
          characters: [charA],
        },
      },
    );

    await act(async () => {
      rerender({
        playingClipId: 1,
        entries: [entry1],
        characters: [charA],
      });
    });

    expect(result.current.slots[0]).toEqual({ character: charA }); // inner-right
    expect(result.current.slots[1]).toBeNull();
    expect(result.current.slots[2]).toBeNull();
    expect(result.current.slots[3]).toBeNull();
    expect(result.current.isMultiSlot).toBe(false);
  });

  it("2人目の発話でinner-left(slot 1)に配置されisMultiSlot=true", async () => {
    const charA = makeChar("キャラA");
    const charB = makeChar("キャラB");
    const entry1 = makeClipEntry(1, "キャラA");
    const entry2 = makeClipEntry(2, "キャラB");

    const { result, rerender } = renderHook(
      ({
        playingClipId,
        entries,
        characters,
      }: {
        playingClipId: number | null;
        entries: TimelineEntry[];
        characters: CharacterEntry[];
      }) => useCharacterStage(playingClipId, entries, characters),
      {
        initialProps: {
          playingClipId: null,
          entries: [entry1, entry2],
          characters: [charA, charB],
        },
      },
    );

    // charA speaks
    await act(async () => {
      rerender({
        playingClipId: 1,
        entries: [entry1, entry2],
        characters: [charA, charB],
      });
    });
    expect(result.current.isMultiSlot).toBe(false);

    // charB speaks
    await act(async () => {
      rerender({
        playingClipId: 2,
        entries: [entry1, entry2],
        characters: [charA, charB],
      });
    });

    expect(result.current.slots[0]).toEqual({ character: charA }); // inner-right
    expect(result.current.slots[1]).toEqual({ character: charB }); // inner-left
    expect(result.current.isMultiSlot).toBe(true);
  });

  it("入場順にinner-right→inner-left→outer-right→outer-leftと配置される", async () => {
    const chars = ["A", "B", "C", "D"].map((n) => makeChar(`キャラ${n}`));
    const entries = chars.map((c, i) => makeClipEntry(i + 1, c.speakerName));

    const { result, rerender } = renderHook(
      ({
        playingClipId,
        entries,
        characters,
      }: {
        playingClipId: number | null;
        entries: TimelineEntry[];
        characters: CharacterEntry[];
      }) => useCharacterStage(playingClipId, entries, characters),
      { initialProps: { playingClipId: null, entries, characters: chars } },
    );

    for (let id = 1; id <= 4; id++) {
      await act(async () => {
        rerender({ playingClipId: id, entries, characters: chars });
      });
    }

    expect(result.current.slots[0]?.character.speakerName).toBe("キャラA"); // inner-right
    expect(result.current.slots[1]?.character.speakerName).toBe("キャラB"); // inner-left
    expect(result.current.slots[2]?.character.speakerName).toBe("キャラC"); // outer-right
    expect(result.current.slots[3]?.character.speakerName).toBe("キャラD"); // outer-left
  });

  it("同じキャラが再発話してもスロットは変わらず、カウンタがリセットされる", async () => {
    const charA = makeChar("キャラA");
    const charB = makeChar("キャラB");
    const entries: TimelineEntry[] = [
      makeClipEntry(1, "キャラA"),
      makeClipEntry(2, "キャラB"),
      makeClipEntry(3, "キャラA"),
    ];
    const chars = [charA, charB];

    const { result, rerender } = renderHook(
      ({
        playingClipId,
        entries,
        characters,
      }: {
        playingClipId: number | null;
        entries: TimelineEntry[];
        characters: CharacterEntry[];
      }) => useCharacterStage(playingClipId, entries, characters),
      { initialProps: { playingClipId: null, entries, characters: chars } },
    );

    await act(async () => {
      rerender({ playingClipId: 1, entries, characters: chars });
    });
    await act(async () => {
      rerender({ playingClipId: 2, entries, characters: chars });
    });
    await act(async () => {
      rerender({ playingClipId: 3, entries, characters: chars });
    }); // charA再発話

    // charA はinner-rightのまま
    expect(result.current.slots[0]?.character.speakerName).toBe("キャラA");
  });

  it("他キャラのクリップが5本完了したら退場する", async () => {
    const charA = makeChar("キャラA");
    const charB = makeChar("キャラB");
    // charA: clip 1, charB: clips 2-7 (charBが5本完了するにはclip 7スタートが必要)
    const entries: TimelineEntry[] = [
      makeClipEntry(1, "キャラA"),
      ...Array.from({ length: 7 }, (_, i) => makeClipEntry(i + 2, "キャラB")),
    ];
    const chars = [charA, charB];

    const { result, rerender } = renderHook(
      ({
        playingClipId,
        entries,
        characters,
      }: {
        playingClipId: number | null;
        entries: TimelineEntry[];
        characters: CharacterEntry[];
      }) => useCharacterStage(playingClipId, entries, characters),
      { initialProps: { playingClipId: null, entries, characters: chars } },
    );

    // charA speaks clip 1
    await act(async () => {
      rerender({ playingClipId: 1, entries, characters: chars });
    });
    // charB speaks clips 2-6
    // count: 2→3(完了=B, A.count=1), 3→4(A.count=2), 4→5(A.count=3), 5→6(A.count=4)
    for (let id = 2; id <= 6; id++) {
      await act(async () => {
        rerender({ playingClipId: id, entries, characters: chars });
      });
    }
    expect(result.current.slots[0]?.character.speakerName).toBe("キャラA"); // count=4 still there

    // charB speaks clip 7 (clip 6完了 = 5本目 → charA evicted)
    await act(async () => {
      rerender({ playingClipId: 7, entries, characters: chars });
    });

    expect(result.current.slots[0]).toBeNull(); // charA evicted
    expect(result.current.slots[1]?.character.speakerName).toBe("キャラB"); // charB still there
  });

  it("再生キューが空になってから15秒で全員退場する", async () => {
    const charA = makeChar("キャラA");
    const entries: TimelineEntry[] = [makeClipEntry(1, "キャラA")];
    const chars = [charA];

    const { result, rerender } = renderHook(
      ({
        playingClipId,
        entries,
        characters,
      }: {
        playingClipId: number | null;
        entries: TimelineEntry[];
        characters: CharacterEntry[];
      }) => useCharacterStage(playingClipId, entries, characters),
      { initialProps: { playingClipId: null, entries, characters: chars } },
    );

    await act(async () => {
      rerender({ playingClipId: 1, entries, characters: chars });
    });
    expect(result.current.slots[0]?.character.speakerName).toBe("キャラA");

    // Queue becomes empty
    await act(async () => {
      rerender({ playingClipId: null, entries, characters: chars });
    });
    expect(result.current.slots[0]?.character.speakerName).toBe("キャラA"); // still there

    // 15 seconds pass
    await act(async () => {
      vi.advanceTimersByTime(QUEUE_EMPTY_MS);
    });

    expect(result.current.slots).toEqual([null, null, null, null]);
    expect(result.current.isMultiSlot).toBe(false);
  });

  it("15秒タイマー中に新クリップが来たらタイマーキャンセル", async () => {
    const charA = makeChar("キャラA");
    const entries: TimelineEntry[] = [
      makeClipEntry(1, "キャラA"),
      makeClipEntry(2, "キャラA"),
    ];
    const chars = [charA];

    const { result, rerender } = renderHook(
      ({
        playingClipId,
        entries,
        characters,
      }: {
        playingClipId: number | null;
        entries: TimelineEntry[];
        characters: CharacterEntry[];
      }) => useCharacterStage(playingClipId, entries, characters),
      { initialProps: { playingClipId: null, entries, characters: chars } },
    );

    await act(async () => {
      rerender({ playingClipId: 1, entries, characters: chars });
    });
    await act(async () => {
      rerender({ playingClipId: null, entries, characters: chars });
    }); // queue empty

    // 10 seconds pass (under 15s)
    await act(async () => {
      vi.advanceTimersByTime(10_000);
    });
    expect(result.current.slots[0]?.character.speakerName).toBe("キャラA");

    // New clip starts - timer should be cancelled
    await act(async () => {
      rerender({ playingClipId: 2, entries, characters: chars });
    });

    // 15 more seconds pass (total >15 from first pause, but timer was cancelled)
    await act(async () => {
      vi.advanceTimersByTime(QUEUE_EMPTY_MS);
    });
    expect(result.current.slots[0]?.character.speakerName).toBe("キャラA"); // still there
  });

  it("4枠埋まった状態で5人目が発話すると最古入場キャラを退場させる", async () => {
    const chars = ["A", "B", "C", "D", "E"].map((n) => makeChar(`キャラ${n}`));
    const entries = chars.map((c, i) => makeClipEntry(i + 1, c.speakerName));

    const { result, rerender } = renderHook(
      ({
        playingClipId,
        entries,
        characters,
      }: {
        playingClipId: number | null;
        entries: TimelineEntry[];
        characters: CharacterEntry[];
      }) => useCharacterStage(playingClipId, entries, characters),
      { initialProps: { playingClipId: null, entries, characters: chars } },
    );

    // A, B, C, D enter in order
    for (let id = 1; id <= 4; id++) {
      await act(async () => {
        rerender({ playingClipId: id, entries, characters: chars });
      });
    }
    expect(result.current.slots[0]?.character.speakerName).toBe("キャラA");

    // E enters - should evict A (oldest)
    await act(async () => {
      rerender({ playingClipId: 5, entries, characters: chars });
    });

    expect(result.current.slots[0]?.character.speakerName).toBe("キャラE"); // E took A's slot (innermost available = 0)
    expect(result.current.slots[1]?.character.speakerName).toBe("キャラB");
    expect(result.current.slots[2]?.character.speakerName).toBe("キャラC");
    expect(result.current.slots[3]?.character.speakerName).toBe("キャラD");
  });

  it("4スロットレイアウト中に1人だけ残ってもisMultiSlotはtrueのまま", async () => {
    const charA = makeChar("キャラA");
    const charB = makeChar("キャラB");
    const entries: TimelineEntry[] = [
      makeClipEntry(1, "キャラA"),
      makeClipEntry(2, "キャラB"),
      ...Array.from({ length: 6 }, (_, i) => makeClipEntry(i + 3, "キャラA")),
    ];

    const { result, rerender } = renderHook(
      ({ playingClipId }: { playingClipId: number | null }) =>
        useCharacterStage(playingClipId, entries, [charA, charB]),
      { initialProps: { playingClipId: null } },
    );

    await act(async () => {
      rerender({ playingClipId: 1 });
    }); // A enters
    await act(async () => {
      rerender({ playingClipId: 2 });
    }); // B enters → isMultiSlot=true
    expect(result.current.isMultiSlot).toBe(true);

    // A speaks 6 clips (clips 3-8 start → clips 3-7 complete = 5 completions → B exits)
    for (let id = 3; id <= 8; id++) {
      await act(async () => {
        rerender({ playingClipId: id });
      });
    }

    expect(result.current.slots[1]).toBeNull(); // B evicted
    expect(result.current.isMultiSlot).toBe(true); // still multi-slot (A is still there)
  });

  it("全員退場後に1人が入場するとisMultiSlot=falseで中央レイアウトに戻る", async () => {
    const charA = makeChar("キャラA");
    const charB = makeChar("キャラB");
    const entries: TimelineEntry[] = [
      makeClipEntry(1, "キャラA"),
      makeClipEntry(2, "キャラB"),
      makeClipEntry(3, "キャラA"),
    ];
    const chars = [charA, charB];

    const { result, rerender } = renderHook(
      ({
        playingClipId,
        entries,
        characters,
      }: {
        playingClipId: number | null;
        entries: TimelineEntry[];
        characters: CharacterEntry[];
      }) => useCharacterStage(playingClipId, entries, characters),
      { initialProps: { playingClipId: null, entries, characters: chars } },
    );

    await act(async () => {
      rerender({ playingClipId: 1, entries, characters: chars });
    });
    await act(async () => {
      rerender({ playingClipId: 2, entries, characters: chars });
    }); // isMultiSlot=true
    await act(async () => {
      rerender({ playingClipId: null, entries, characters: chars });
    }); // queue empty

    // 15s timer fires → all exit, isMultiSlot=false
    await act(async () => {
      vi.advanceTimersByTime(QUEUE_EMPTY_MS);
    });
    expect(result.current.isMultiSlot).toBe(false);

    // A enters again
    await act(async () => {
      rerender({ playingClipId: 3, entries, characters: chars });
    });
    expect(result.current.isMultiSlot).toBe(false); // center layout
    expect(result.current.slots[0]?.character.speakerName).toBe("キャラA");
  });

  it("同一話者がスタイルを変えて発話しても同じスロットを維持し画像が更新される", async () => {
    const charANormal = makeChar("キャラA", "ノーマル");
    const charASweet = {
      ...makeChar("キャラA", "あまあま"),
      mouthClosed: "キャラA-sweet-closed.png",
      mouthOpen: "キャラA-sweet-open.png",
    };
    const entries: TimelineEntry[] = [
      makeClipEntry(1, "キャラA", "ノーマル"),
      makeClipEntry(2, "キャラA", "あまあま"),
    ];
    const chars = [charANormal, charASweet];

    const { result, rerender } = renderHook(
      ({
        playingClipId,
        entries,
        characters,
      }: {
        playingClipId: number | null;
        entries: TimelineEntry[];
        characters: CharacterEntry[];
      }) => useCharacterStage(playingClipId, entries, characters),
      { initialProps: { playingClipId: null, entries, characters: chars } },
    );

    await act(async () => {
      rerender({ playingClipId: 1, entries, characters: chars });
    });
    expect(result.current.slots[0]?.character).toEqual(charANormal);
    expect(result.current.isMultiSlot).toBe(false);

    await act(async () => {
      rerender({ playingClipId: 2, entries, characters: chars });
    });

    expect(result.current.slots[0]?.character).toEqual(charASweet);
    expect(result.current.slots[1]).toBeNull();
    expect(result.current.slots[2]).toBeNull();
    expect(result.current.slots[3]).toBeNull();
    // スタイル変更だけでは2人目入場にならないため
    expect(result.current.isMultiSlot).toBe(false);
  });

  it("キャラクター設定に存在しないスピーカーは無視される", async () => {
    const charA = makeChar("キャラA");
    const entries: TimelineEntry[] = [makeClipEntry(1, "存在しないキャラ")];
    const chars = [charA];

    const { result, rerender } = renderHook(
      ({
        playingClipId,
        entries,
        characters,
      }: {
        playingClipId: number | null;
        entries: TimelineEntry[];
        characters: CharacterEntry[];
      }) => useCharacterStage(playingClipId, entries, characters),
      { initialProps: { playingClipId: null, entries, characters: chars } },
    );

    await act(async () => {
      rerender({ playingClipId: 1, entries, characters: chars });
    });

    expect(result.current.slots).toEqual([null, null, null, null]);
  });
});
