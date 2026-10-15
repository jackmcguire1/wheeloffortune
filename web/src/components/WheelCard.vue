<script setup lang="ts">
import { mdiPauseCircleOutline, mdiRotateRight, mdiTrophyOutline } from "@mdi/js";
import { computed } from "vue";
import type { Wheel } from "@/gen/wheel/v1/wheel_pb";
import { modeInfo } from "@/lib/modes";
import FortuneWheel from "./FortuneWheel.vue";

const props = defineProps<{ wheel: Wheel }>();

const mode = computed(() => modeInfo(props.wheel.mode));
const segments = computed(() => props.wheel.segments.map((s) => ({ label: s.spec?.label ?? "", color: s.spec?.color })));
const probs = computed(() => props.wheel.segments.map((s) => s.probability));
</script>

<template>
  <v-card :to="`/wheels/${wheel.id}`" class="wheel-card h-100 d-flex flex-column" variant="flat">
    <div class="preview pa-6">
      <FortuneWheel mini :segments="segments" :probabilities="probs" :mode="wheel.mode" :disabled="!wheel.enabled || wheel.exhausted" />
    </div>
    <v-card-item>
      <v-card-title class="font-weight-bold">{{ wheel.name }}</v-card-title>
      <v-card-subtitle>
        {{ wheel.segments.length }} segments<span v-if="wheel.ownerName"> · by {{ wheel.ownerName }}</span>
      </v-card-subtitle>
    </v-card-item>
    <v-card-text class="pt-0 flex-grow-1 text-medium-emphasis desc">
      {{ wheel.description || "No description." }}
    </v-card-text>
    <v-card-actions class="px-4 pb-4 flex-wrap ga-2">
      <v-chip size="small" :color="mode.color" variant="tonal" :prepend-icon="mode.icon">{{ mode.short }}</v-chip>
      <v-chip size="small" variant="tonal" :prepend-icon="mdiRotateRight">{{ wheel.totalSpins }} spins</v-chip>
      <v-chip v-if="wheel.exhausted" size="small" color="error" variant="tonal" :prepend-icon="mdiTrophyOutline">All won</v-chip>
      <v-chip v-else-if="!wheel.enabled" size="small" color="warning" variant="tonal" :prepend-icon="mdiPauseCircleOutline">Paused</v-chip>
    </v-card-actions>
  </v-card>
</template>

<style scoped>
.wheel-card {
  border: 1px solid rgba(var(--v-border-color), 0.08);
  transition:
    transform 0.25s ease,
    box-shadow 0.25s ease;
}

.wheel-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 18px 40px -18px rgba(108, 60, 240, 0.45);
}

.wheel-card:hover :deep(.rotor) {
  transition-duration: 1.4s !important;
  transform: rotate(120deg) !important;
}

.preview {
  background: linear-gradient(135deg, rgba(var(--v-theme-primary), 0.12), rgba(var(--v-theme-secondary), 0.1));
  display: flex;
  justify-content: center;
}

.preview > * {
  max-width: 180px;
}

.desc {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
