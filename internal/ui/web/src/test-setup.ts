import '@testing-library/jest-dom/vitest';

// jsdom has no Web Animations API, which svelte's animate:flip calls on every
// reordered element, and which its transitions call to play themselves. Without
// it any test that reorders a keyed list or renders a transition throws.
if (typeof Element !== 'undefined' && !Element.prototype.getAnimations) {
  Element.prototype.getAnimations = () => [];
}
if (typeof Element !== 'undefined' && !Element.prototype.animate) {
  Element.prototype.animate = () =>
    ({
      cancel() {},
      finish() {},
      pause() {},
      play() {},
      reverse() {},
      addEventListener() {},
      removeEventListener() {},
      currentTime: 0,
      playState: 'finished',
      startTime: 0,
      effect: null,
      finished: Promise.resolve(),
      onfinish: null,
      oncancel: null
    }) as unknown as Animation;
}
