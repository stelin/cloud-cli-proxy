import "@testing-library/jest-dom/vitest";
import { afterEach } from "vitest";
import { cleanup } from "@testing-library/react";

// 每个测试后清理 React DOM 残留，避免 act / event listener 串扰
afterEach(() => {
  cleanup();
});

// jsdom 默认未实现 matchMedia / ResizeObserver，Radix UI / shadcn primitives 在挂载时会读这两个 API
if (typeof window !== "undefined") {
  // jsdom 29 + vitest 4 组合下 about:blank 时 localStorage 不可用（getItem is not a function）。
  // 引入 CR-07 后多个 hook 在挂载时调用 getToken() → localStorage.getItem，
  // 所以这里补一份基于 Map 的 in-memory polyfill。
  const ensureStorage = (key: "localStorage" | "sessionStorage") => {
    const existing = (window as unknown as Record<string, unknown>)[key];
    const ok =
      existing && typeof (existing as Storage).getItem === "function";
    if (ok) return;
    const store = new Map<string, string>();
    const impl: Storage = {
      get length() {
        return store.size;
      },
      clear: () => store.clear(),
      getItem: (k) => (store.has(k) ? store.get(k)! : null),
      key: (i) => Array.from(store.keys())[i] ?? null,
      removeItem: (k) => {
        store.delete(k);
      },
      setItem: (k, v) => {
        store.set(k, String(v));
      },
    };
    Object.defineProperty(window, key, { value: impl, writable: true });
  };
  ensureStorage("localStorage");
  ensureStorage("sessionStorage");

  if (!window.matchMedia) {
    Object.defineProperty(window, "matchMedia", {
      writable: true,
      value: (query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => false,
      }),
    });
  }
  if (!(globalThis as { ResizeObserver?: unknown }).ResizeObserver) {
    class ResizeObserverPolyfill {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
    (globalThis as { ResizeObserver?: unknown }).ResizeObserver =
      ResizeObserverPolyfill;
  }

  // Radix UI Select / DropdownMenu 等内部使用 PointerEvent API + scrollIntoView，
  // jsdom 都没有实现，补齐以保证组件测试可用
  if (!HTMLElement.prototype.hasPointerCapture) {
    HTMLElement.prototype.hasPointerCapture = () => false;
  }
  if (!HTMLElement.prototype.setPointerCapture) {
    HTMLElement.prototype.setPointerCapture = () => {};
  }
  if (!HTMLElement.prototype.releasePointerCapture) {
    HTMLElement.prototype.releasePointerCapture = () => {};
  }
  if (!HTMLElement.prototype.scrollIntoView) {
    HTMLElement.prototype.scrollIntoView = () => {};
  }
}
