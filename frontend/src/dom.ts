// DOM 要素取得ヘルパー。id に対応する要素が無ければ例外を投げる。
export function getElement<T extends HTMLElement>(id: string): T {
  const el = document.getElementById(id);
  if (el === null) {
    throw new Error(`element not found: #${id}`);
  }
  return el as T;
}
