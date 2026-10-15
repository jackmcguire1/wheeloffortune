<script setup lang="ts">
import { computed, ref } from "vue";
import { WheelMode } from "@/gen/wheel/v1/wheel_pb";
import { colorFor, contrastText } from "@/lib/palette";
import { layoutSlices, targetRotation } from "@/lib/probability";

export interface WheelSlice {
  label: string;
  color?: string;
}

const props = withDefaults(
  defineProps<{
    segments: WheelSlice[];
    probabilities: number[];
    mode: WheelMode;
    /** Hides labels, pegs and the hub, for thumbnails. */
    mini?: boolean;
    disabled?: boolean;
  }>(),
  { mini: false, disabled: false },
);

const emit = defineEmits<{ done: [index: number] }>();

const VIEW = 400;
const C = VIEW / 2;
const R = C - 14;

const rotation = ref(0);
const duration = ref(0);
const spinning = ref(false);
const landed = ref<number | null>(null);

const slices = computed(() => layoutSlices(props.mode, props.probabilities));

function point(angle: number, radius: number) {
  const rad = (angle * Math.PI) / 180;
  return { x: C + radius * Math.sin(rad), y: C - radius * Math.cos(rad) };
}

function arcPath(start: number, end: number) {
  const a = point(start, R);
  const b = point(end, R);
  const large = end - start > 180 ? 1 : 0;
  return `M ${C} ${C} L ${a.x} ${a.y} A ${R} ${R} 0 ${large} 1 ${b.x} ${b.y} Z`;
}

const drawn = computed(() =>
  slices.value.map((s) => {
    const seg = props.segments[s.index];
    const fill = colorFor(s.index, seg?.color);
    const width = s.end - s.start;
    const mid = s.start + width / 2;
    const maxChars = Math.max(3, Math.min(14, Math.floor(width / 2.2)));
    const raw = seg?.label ?? "";
    const label = raw.length > maxChars ? `${raw.slice(0, maxChars - 1)}…` : raw;
    return {
      ...s,
      full: width >= 359.99,
      path: arcPath(s.start, s.end),
      fill,
      text: contrastText(fill),
      mid,
      // Keep text upright: right-hand labels read outwards, left-hand inwards.
      flip: mid < 180,
      label: width >= 7 ? label : "",
      fontSize: width < 20 ? 12 : width < 40 ? 15 : 18,
      peg: point(s.start, R - 2),
    };
  }),
);

const reducedMotion =
  typeof window !== "undefined" && window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;

/** Spins until the pointer rests inside the given segment's slice. */
function spinTo(index: number): Promise<void> {
  const slice = slices.value.find((s) => s.index === index);
  if (!slice) return Promise.resolve();
  landed.value = null;
  spinning.value = true;
  duration.value = reducedMotion ? 600 : 5200;
  rotation.value = targetRotation(rotation.value, slice, reducedMotion ? 1 : 7);
  return new Promise((resolve) => {
    window.setTimeout(() => {
      spinning.value = false;
      landed.value = index;
      emit("done", index);
      resolve();
    }, duration.value + 50);
  });
}

defineExpose({ spinTo, spinning });
</script>

<template>
  <div class="wheel" :class="{ mini, disabled, spinning }">
    <svg :viewBox="`0 0 ${VIEW} ${VIEW}`" role="img" aria-label="Wheel of fortune">
      <defs>
        <radialGradient id="wheel-shine" cx="50%" cy="35%" r="70%">
          <stop offset="0%" stop-color="#fff" stop-opacity="0.35" />
          <stop offset="60%" stop-color="#fff" stop-opacity="0" />
        </radialGradient>
        <linearGradient id="wheel-rim" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stop-color="#ffd166" />
          <stop offset="50%" stop-color="#ff4f8b" />
          <stop offset="100%" stop-color="#7c4dff" />
        </linearGradient>
      </defs>

      <circle :cx="C" :cy="C" :r="C - 2" fill="url(#wheel-rim)" />
      <circle :cx="C" :cy="C" :r="R + 2" fill="#1a1033" />

      <g
        class="rotor"
        :style="{
          transform: `rotate(${rotation}deg)`,
          transitionDuration: `${duration}ms`,
        }"
      >
        <template v-for="s in drawn" :key="s.index">
          <circle v-if="s.full" :cx="C" :cy="C" :r="R" :fill="s.fill" />
          <path
            v-else
            :d="s.path"
            :fill="s.fill"
            stroke="rgba(0,0,0,0.18)"
            stroke-width="1"
            :class="{ win: landed === s.index }"
          />
        </template>
        <!-- Labels drawn radially, reading from the rim towards the hub. -->
        <template v-if="!mini">
          <g v-for="s in drawn" :key="`l${s.index}`" :transform="`rotate(${s.mid} ${C} ${C})`">
            <text
              v-if="s.label"
              :x="C"
              :y="C - R + 18"
              :fill="s.text"
              :font-size="s.fontSize"
              font-weight="700"
              :text-anchor="s.flip ? 'end' : 'start'"
              dominant-baseline="middle"
              :transform="`rotate(${s.flip ? -90 : 90} ${C} ${C - R + 18})`"
              class="label"
            >
              {{ s.label }}
            </text>
          </g>
          <circle
            v-for="s in drawn.length > 1 ? drawn : []"
            :key="`p${s.index}`"
            :cx="s.peg.x"
            :cy="s.peg.y"
            r="4"
            fill="#fff"
            stroke="#1a1033"
            stroke-width="1.5"
          />
        </template>
        <circle :cx="C" :cy="C" :r="R" fill="url(#wheel-shine)" pointer-events="none" />
      </g>

      <g v-if="!mini">
        <circle :cx="C" :cy="C" r="34" fill="#1a1033" />
        <circle :cx="C" :cy="C" r="28" fill="url(#wheel-rim)" />
        <circle :cx="C" :cy="C" r="12" fill="#fff" opacity="0.9" />
      </g>
    </svg>

    <div v-if="!mini" class="pointer" :class="{ bounce: landed !== null }" aria-hidden="true">
      <svg viewBox="0 0 40 52">
        <path d="M20 50 L4 12 A16 16 0 1 1 36 12 Z" fill="#fff" stroke="#1a1033" stroke-width="3" />
        <circle cx="20" cy="16" r="6" fill="#ff4f8b" />
      </svg>
    </div>
  </div>
</template>

<style scoped>
.wheel {
  position: relative;
  width: 100%;
  aspect-ratio: 1;
  user-select: none;
}

.wheel svg {
  width: 100%;
  height: 100%;
  display: block;
  filter: drop-shadow(0 18px 40px rgba(26, 16, 51, 0.35));
}

.wheel.mini svg {
  filter: drop-shadow(0 6px 14px rgba(26, 16, 51, 0.25));
}

.wheel.disabled svg {
  filter: grayscale(0.85) opacity(0.7);
}

.rotor {
  transform-origin: 200px 200px;
  transform-box: view-box;
  transition-property: transform;
  transition-timing-function: cubic-bezier(0.12, 0.72, 0.12, 1);
}

.label {
  font-family: var(--wof-font);
  paint-order: stroke;
  stroke: rgba(0, 0, 0, 0.12);
  stroke-width: 2px;
}

.win {
  animation: win-flash 0.6s ease-in-out 3;
}

@keyframes win-flash {
  50% {
    filter: brightness(1.35);
  }
}

.pointer {
  position: absolute;
  top: -3%;
  left: 50%;
  width: 11%;
  transform: translateX(-50%);
  transform-origin: 50% 20%;
  filter: drop-shadow(0 4px 6px rgba(0, 0, 0, 0.3));
}

.spinning .pointer {
  animation: wobble 0.12s ease-in-out infinite alternate;
}

.pointer.bounce {
  animation: bounce 0.5s cubic-bezier(0.3, 1.6, 0.5, 1);
}

@keyframes wobble {
  from {
    transform: translateX(-50%) rotate(-7deg);
  }
  to {
    transform: translateX(-50%) rotate(4deg);
  }
}

@keyframes bounce {
  0% {
    transform: translateX(-50%) rotate(-12deg);
  }
  100% {
    transform: translateX(-50%) rotate(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .spinning .pointer,
  .pointer.bounce,
  .win {
    animation: none;
  }
}
</style>
