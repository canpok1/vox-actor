import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { usePersistedState } from "./usePersistedState";

const KEY = "test.usePersistedState";

const numberParse = (raw: string): number | null => {
  const n = Number(raw);
  return Number.isFinite(n) ? n : null;
};
const numberSerialize = (n: number): string => String(n);

describe("usePersistedState", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("localStorage に値が無い場合は defaultValue を返す", () => {
    const { result } = renderHook(() =>
      usePersistedState(KEY, 42, numberParse, numberSerialize),
    );
    expect(result.current[0]).toBe(42);
  });

  it("localStorage の値を初期値として復元する", () => {
    localStorage.setItem(KEY, "7");
    const { result } = renderHook(() =>
      usePersistedState(KEY, 42, numberParse, numberSerialize),
    );
    expect(result.current[0]).toBe(7);
  });

  it("parse が null を返す破損値は defaultValue にフォールバックする", () => {
    localStorage.setItem(KEY, "not-a-number");
    const { result } = renderHook(() =>
      usePersistedState(KEY, 42, numberParse, numberSerialize),
    );
    expect(result.current[0]).toBe(42);
  });

  it("setter で値の更新と localStorage への保存を行う", () => {
    const { result } = renderHook(() =>
      usePersistedState(KEY, 0, numberParse, numberSerialize),
    );
    act(() => {
      result.current[1](123);
    });
    expect(result.current[0]).toBe(123);
    expect(localStorage.getItem(KEY)).toBe("123");
  });

  it("localStorage.setItem が例外を投げてもエラーログ出力のみで状態は更新する", () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const setItemSpy = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("quota exceeded");
      });
    const { result } = renderHook(() =>
      usePersistedState(KEY, 0, numberParse, numberSerialize),
    );
    act(() => {
      result.current[1](5);
    });
    expect(result.current[0]).toBe(5);
    expect(errorSpy).toHaveBeenCalled();
    setItemSpy.mockRestore();
  });
});
