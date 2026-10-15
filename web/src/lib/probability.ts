import { WheelMode } from "@/gen/wheel/v1/wheel_pb";

/** The subset of a segment the probability rules depend on. */
export interface ChanceInput {
  weight: number;
  /** Remaining stock. Undefined means unlimited. */
  stock?: number;
}

/** True while a segment can still be won. */
export function isAvailable(s: ChanceInput): boolean {
  return s.stock === undefined || s.stock > 0;
}

/**
 * Mirrors the server's spin rules:
 * - EQUAL: 1/n across available segments.
 * - WEIGHTED: weight / sum(weight) across available segments.
 * - INVENTORY: stock / sum(stock).
 * Returns all zeros when nothing can be won.
 */
export function probabilities(mode: WheelMode, segments: ChanceInput[]): number[] {
  const raw = segments.map((s) => {
    if (!isAvailable(s)) return 0;
    switch (mode) {
      case WheelMode.WEIGHTED:
        return Math.max(0, s.weight);
      case WheelMode.INVENTORY:
        return s.stock ?? 0;
      default:
        return 1;
    }
  });
  const total = raw.reduce((a, b) => a + b, 0);
  return total > 0 ? raw.map((v) => v / total) : raw.map(() => 0);
}

export interface Slice {
  index: number;
  start: number; // degrees, clockwise from 12 o'clock
  end: number;
  probability: number;
}

/**
 * Lays segments out around the wheel. EQUAL wheels get equal slices; the other
 * modes size slices by probability. Segments that can't win are omitted, so
 * landing inside a slice always matches the server result.
 */
export function layoutSlices(mode: WheelMode, probs: number[]): Slice[] {
  const live = probs.map((p, index) => ({ p, index })).filter((x) => x.p > 0);
  if (live.length === 0) {
    // Nothing winnable: draw everything equally so the wheel still renders.
    const step = 360 / Math.max(1, probs.length);
    return probs.map((_, i) => ({ index: i, start: i * step, end: (i + 1) * step, probability: 0 }));
  }
  const sizes = mode === WheelMode.EQUAL ? live.map(() => 1 / live.length) : live.map((x) => x.p);
  let cursor = 0;
  return live.map((x, i) => {
    const start = cursor;
    cursor += sizes[i] * 360;
    return { index: x.index, start, end: i === live.length - 1 ? 360 : cursor, probability: x.p };
  });
}

/**
 * Picks the rotation (degrees, always increasing) that lands the pointer at
 * 12 o'clock on a random point inside the winning slice.
 */
export function targetRotation(current: number, slice: Slice, turns = 6, rand = Math.random): number {
  const width = slice.end - slice.start;
  const margin = Math.min(width * 0.15, 4);
  const point = slice.start + margin + rand() * Math.max(0, width - margin * 2);
  // Rotating the wheel by R moves the point at angle a to a + R, we want 360k.
  const base = ((360 - point) % 360 + 360) % 360;
  const currentMod = ((current % 360) + 360) % 360;
  let delta = base - currentMod;
  if (delta < 0) delta += 360;
  return current + turns * 360 + delta;
}

export function formatPercent(p: number): string {
  if (p === 0) return "0%";
  if (p < 0.001) return "<0.1%";
  return `${(p * 100).toFixed(p < 0.1 ? 1 : 0)}%`;
}
