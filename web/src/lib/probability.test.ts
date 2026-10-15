import { describe, expect, it } from "vitest";
import { WheelMode } from "@/gen/wheel/v1/wheel_pb";
import { layoutSlices, probabilities, targetRotation } from "./probability";

describe("probabilities", () => {
  it("splits equally across available segments", () => {
    const p = probabilities(WheelMode.EQUAL, [{ weight: 5 }, { weight: 1, stock: 0 }, { weight: 1, stock: 3 }]);
    expect(p).toEqual([0.5, 0, 0.5]);
  });

  it("uses weights in weighted mode", () => {
    const p = probabilities(WheelMode.WEIGHTED, [{ weight: 3 }, { weight: 1 }]);
    expect(p).toEqual([0.75, 0.25]);
  });

  it("ignores sold-out segments in weighted mode", () => {
    const p = probabilities(WheelMode.WEIGHTED, [{ weight: 3, stock: 0 }, { weight: 1 }]);
    expect(p).toEqual([0, 1]);
  });

  it("uses remaining stock in inventory mode", () => {
    const p = probabilities(WheelMode.INVENTORY, [{ weight: 1, stock: 1 }, { weight: 1, stock: 3 }]);
    expect(p).toEqual([0.25, 0.75]);
  });

  it("returns zeros when exhausted", () => {
    expect(probabilities(WheelMode.INVENTORY, [{ weight: 1, stock: 0 }])).toEqual([0]);
  });
});

describe("layoutSlices", () => {
  it("gives equal slices in equal mode and omits unwinnable segments", () => {
    const slices = layoutSlices(WheelMode.EQUAL, [0.5, 0, 0.5]);
    expect(slices.map((s) => s.index)).toEqual([0, 2]);
    expect(slices[0].end).toBe(180);
    expect(slices[1].end).toBe(360);
  });

  it("sizes slices by probability otherwise", () => {
    const slices = layoutSlices(WheelMode.WEIGHTED, [0.75, 0.25]);
    expect(slices[0].end).toBe(270);
  });
});

describe("targetRotation", () => {
  it("lands the pointer inside the winning slice", () => {
    const slice = { index: 1, start: 90, end: 180, probability: 0.25 };
    for (const r of [0, 0.5, 0.999]) {
      const rot = targetRotation(1234, slice, 5, () => r);
      expect(rot).toBeGreaterThan(1234 + 5 * 360 - 1);
      // The point under the pointer (12 o'clock) in wheel coordinates:
      const pointer = (((360 - rot) % 360) + 360) % 360;
      expect(pointer).toBeGreaterThanOrEqual(90);
      expect(pointer).toBeLessThanOrEqual(180);
    }
  });
});
