import { useState } from "react";

export function usePersistedState<T>(
  key: string,
  defaultValue: T,
  parse: (raw: string) => T | null,
  serialize: (value: T) => string,
): [T, (value: T) => void] {
  const [value, setValue] = useState<T>(() => {
    const raw = localStorage.getItem(key);
    if (raw === null) return defaultValue;
    const parsed = parse(raw);
    return parsed ?? defaultValue;
  });

  const setPersisted = (next: T): void => {
    setValue(next);
    try {
      localStorage.setItem(key, serialize(next));
    } catch (err) {
      console.error("failed to persist state", key, err);
    }
  };

  return [value, setPersisted];
}
