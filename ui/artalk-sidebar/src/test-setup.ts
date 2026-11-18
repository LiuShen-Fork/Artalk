// jsdom 29 exposes `localStorage` as a prototype accessor, and vitest copies
// only own properties of the jsdom window onto the test global, so the module
// under test sees `localStorage` as undefined. Provide the same synchronous
// storage contract on the global for the state tests.
if (typeof globalThis.localStorage === 'undefined') {
  const store = new Map<string, string>()
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: {
      get length() {
        return store.size
      },
      clear: () => store.clear(),
      getItem: (key: string) => store.get(key) ?? null,
      key: (index: number) => [...store.keys()][index] ?? null,
      removeItem: (key: string) => void store.delete(key),
      setItem: (key: string, value: string) => void store.set(key, String(value)),
    },
  })
}
