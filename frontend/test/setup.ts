import "@testing-library/jest-dom/vitest";

// jsdom では scrollIntoView が未実装のためグローバルにスタブを設定する
window.HTMLElement.prototype.scrollIntoView = function () {};
