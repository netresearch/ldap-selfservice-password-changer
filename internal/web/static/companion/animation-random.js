const sample = new Uint32Array(1);

// Uniform [0, 1) samples keep the existing cosmetic timing and motion ranges.
export function animationRandom() {
  globalThis.crypto.getRandomValues(sample);
  return sample[0] / 0x100000000;
}
