import { mdiChartPie, mdiEqualBox, mdiPackageVariantClosed } from "@mdi/js";
import { WheelMode } from "@/gen/wheel/v1/wheel_pb";

export interface ModeInfo {
  mode: WheelMode;
  title: string;
  short: string;
  description: string;
  icon: string;
  color: string;
}

export const MODES: ModeInfo[] = [
  {
    mode: WheelMode.EQUAL,
    title: "Equal",
    short: "Equal odds",
    description: "Every segment that still has prizes has the same chance.",
    icon: mdiEqualBox,
    color: "info",
  },
  {
    mode: WheelMode.WEIGHTED,
    title: "Weighted",
    short: "Weighted",
    description: "You set a weight per segment. A weight of 3 is three times as likely as 1.",
    icon: mdiChartPie,
    color: "secondary",
  },
  {
    mode: WheelMode.INVENTORY,
    title: "Inventory",
    short: "Prize pool",
    description: "Odds follow the prizes left, so the pool drains evenly. Every segment needs a stock.",
    icon: mdiPackageVariantClosed,
    color: "accent",
  },
];

export function modeInfo(mode: WheelMode): ModeInfo {
  return MODES.find((m) => m.mode === mode) ?? MODES[0];
}
