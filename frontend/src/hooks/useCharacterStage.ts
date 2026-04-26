import { useEffect, useReducer, useRef } from "react";
import type { CharacterEntry, TimelineEntry } from "../types/api";

export type SlotIndex = 0 | 1 | 2 | 3;

interface StageChar {
  character: CharacterEntry;
  slotIndex: SlotIndex;
  entryOrder: number;
  otherClipsCount: number;
}

interface StageState {
  chars: StageChar[];
  isMultiSlot: boolean;
}

type StageAction =
  | {
      type: "CLIP_TRANSITION";
      completedCharKey: string | null;
      startedCharKey: string | null;
      startedCharacter: CharacterEntry | null;
      startedEntryOrder: number;
    }
  | { type: "ALL_EXIT" };

const SLOT_PRIORITY: SlotIndex[] = [0, 1, 2, 3];
const OTHER_CLIPS_LIMIT = 5;
export const QUEUE_EMPTY_MS = 15_000;

function charKey(speakerName: string, styleName: string): string {
  return `${speakerName}\0${styleName}`;
}

function availableSlot(chars: StageChar[]): SlotIndex {
  const used = new Set(chars.map((c) => c.slotIndex));
  return SLOT_PRIORITY.find((s) => !used.has(s)) ?? 0;
}

function stageReducer(state: StageState, action: StageAction): StageState {
  switch (action.type) {
    case "CLIP_TRANSITION": {
      let chars = state.chars;

      if (action.completedCharKey !== null) {
        chars = chars
          .map((c) =>
            charKey(c.character.speakerName, c.character.styleName) ===
            action.completedCharKey
              ? c
              : { ...c, otherClipsCount: c.otherClipsCount + 1 },
          )
          .filter((c) => c.otherClipsCount < OTHER_CLIPS_LIMIT);
      }

      if (action.startedCharKey !== null && action.startedCharacter !== null) {
        const existingIdx = chars.findIndex(
          (c) =>
            charKey(c.character.speakerName, c.character.styleName) ===
            action.startedCharKey,
        );

        if (existingIdx >= 0) {
          chars = chars.map((c, i) =>
            i === existingIdx ? { ...c, otherClipsCount: 0 } : c,
          );
        } else {
          let updated = [...chars];
          if (updated.length >= 4) {
            let oldestIdx = 0;
            for (let i = 1; i < updated.length; i++) {
              if (updated[i].entryOrder < updated[oldestIdx].entryOrder) {
                oldestIdx = i;
              }
            }
            updated = [
              ...updated.slice(0, oldestIdx),
              ...updated.slice(oldestIdx + 1),
            ];
          }
          const slot = availableSlot(updated);
          chars = [
            ...updated,
            {
              character: action.startedCharacter,
              slotIndex: slot,
              entryOrder: action.startedEntryOrder,
              otherClipsCount: 0,
            },
          ];
        }
      }

      const isMultiSlot =
        chars.length === 0 ? false : state.isMultiSlot || chars.length >= 2;

      return { chars, isMultiSlot };
    }

    case "ALL_EXIT":
      return { chars: [], isMultiSlot: false };

    default:
      return state;
  }
}

export interface CharacterStageSlot {
  character: CharacterEntry;
}

export interface CharacterStageResult {
  slots: (CharacterStageSlot | null)[];
  isMultiSlot: boolean;
}

export function useCharacterStage(
  playingClipId: number | null,
  entries: TimelineEntry[],
  characters: CharacterEntry[],
): CharacterStageResult {
  const [state, dispatch] = useReducer(stageReducer, {
    chars: [],
    isMultiSlot: false,
  });

  const prevPlayingClipIdRef = useRef<number | null>(null);
  const entryCounterRef = useRef(0);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    const prev = prevPlayingClipIdRef.current;
    const curr = playingClipId;
    prevPlayingClipIdRef.current = curr;

    if (prev === curr) return;

    if (curr !== null && timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }

    let completedCharKey: string | null = null;
    if (prev !== null) {
      const e = entries.find((e) => e.kind === "clip" && e.id === prev);
      if (e?.kind === "clip") {
        completedCharKey = charKey(e.speakerName, e.styleName);
      }
    }

    let startedCharKey: string | null = null;
    let startedCharacter: CharacterEntry | null = null;
    const startedEntryOrder = entryCounterRef.current;

    if (curr !== null) {
      const e = entries.find((e) => e.kind === "clip" && e.id === curr);
      if (e?.kind === "clip") {
        startedCharKey = charKey(e.speakerName, e.styleName);
        startedCharacter =
          characters.find(
            (c) =>
              c.speakerName === e.speakerName && c.styleName === e.styleName,
          ) ?? null;
        if (startedCharacter) {
          entryCounterRef.current++;
        }
      }
    }

    dispatch({
      type: "CLIP_TRANSITION",
      completedCharKey,
      startedCharKey,
      startedCharacter,
      startedEntryOrder,
    });

    if (curr === null && prev !== null) {
      timerRef.current = setTimeout(() => {
        dispatch({ type: "ALL_EXIT" });
        timerRef.current = null;
      }, QUEUE_EMPTY_MS);
    }
  }, [playingClipId, entries, characters]);

  useEffect(() => {
    return () => {
      if (timerRef.current !== null) {
        clearTimeout(timerRef.current);
      }
    };
  }, []);

  const slots: (CharacterStageSlot | null)[] = [null, null, null, null];
  for (const c of state.chars) {
    slots[c.slotIndex] = { character: c.character };
  }

  return { slots, isMultiSlot: state.isMultiSlot };
}
