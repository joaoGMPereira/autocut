import '@testing-library/jest-dom'

// Polyfill ResizeObserver for jsdom (used by radix-ui Slider, etc.)
global.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};
