<script setup lang="ts">
import { mdiPlus, mdiRefresh } from "@mdi/js";
import { computed, onMounted, ref, shallowRef } from "vue";
import FortuneWheel from "@/components/FortuneWheel.vue";
import WheelCard from "@/components/WheelCard.vue";
import { WheelMode, type Wheel } from "@/gen/wheel/v1/wheel_pb";
import { friendlyError, wheelClient } from "@/lib/api";
import { MODES } from "@/lib/modes";

const wheels = shallowRef<Wheel[]>([]);
const loading = ref(true);
const error = ref("");
const filter = ref<WheelMode | "all">("all");

const shown = computed(() => (filter.value === "all" ? wheels.value : wheels.value.filter((w) => w.mode === filter.value)));

const heroSegments = ["Jackpot", "Spin again", "Free coffee", "High five", "Mystery box", "Pizza"].map((label) => ({ label }));
const heroProbs = [0.1, 0.25, 0.2, 0.2, 0.1, 0.15];

async function load() {
  loading.value = true;
  error.value = "";
  try {
    wheels.value = (await wheelClient.listWheels({})).wheels;
  } catch (err) {
    error.value = friendlyError(err);
  } finally {
    loading.value = false;
  }
}

onMounted(load);
</script>

<template>
  <v-container class="py-8" max-width="1280">
    <section class="hero mb-12">
      <v-row align="center">
        <v-col cols="12" md="7">
          <div class="text-overline text-primary font-weight-bold">Redis-backed · real-time odds</div>
          <h1 class="text-h3 text-md-h2 font-weight-black mb-4">
            Spin it. <span class="gradient-text">Win it.</span>
          </h1>
          <p class="text-body-1 text-medium-emphasis mb-6 hero-copy">
            Build wheels with equal odds, custom weights or a limited prize pool. Every spin is settled atomically on
            the server, so the odds you see are the odds you get.
          </p>
          <div class="d-flex flex-wrap ga-3">
            <v-btn to="/wheels/new" color="primary" size="x-large" variant="flat" :prepend-icon="mdiPlus">Create a wheel</v-btn>
            <v-btn href="#gallery" size="x-large" variant="tonal">Browse wheels</v-btn>
          </div>
        </v-col>
        <v-col cols="12" md="5" class="d-flex justify-center">
          <div class="hero-wheel">
            <FortuneWheel :segments="heroSegments" :probabilities="heroProbs" :mode="WheelMode.WEIGHTED" />
          </div>
        </v-col>
      </v-row>
    </section>

    <section id="gallery">
      <div class="d-flex flex-wrap align-center ga-3 mb-6">
        <h2 class="text-h4 font-weight-bold mr-auto">Wheels</h2>
        <v-chip-group v-model="filter" mandatory selected-class="text-primary" color="primary">
          <v-chip value="all" variant="tonal">All</v-chip>
          <v-chip v-for="m in MODES" :key="m.mode" :value="m.mode" :prepend-icon="m.icon" variant="tonal">{{ m.title }}</v-chip>
        </v-chip-group>
        <v-btn :icon="mdiRefresh" variant="text" :loading="loading" aria-label="Refresh" @click="load" />
      </div>

      <v-alert v-if="error" type="warning" variant="tonal" rounded="xl" class="mb-6" :text="error">
        <template #append>
          <v-btn variant="text" @click="load">Retry</v-btn>
        </template>
      </v-alert>

      <v-row v-if="loading && !wheels.length">
        <v-col v-for="n in 3" :key="n" cols="12" sm="6" lg="4">
          <v-skeleton-loader type="image, article" class="rounded-xl" />
        </v-col>
      </v-row>

      <v-sheet v-else-if="!shown.length && !error" rounded="xl" class="pa-12 text-center glass">
        <div class="text-h5 font-weight-bold mb-2">No wheels yet</div>
        <p class="text-medium-emphasis mb-6">Be the first to build one. It takes under a minute.</p>
        <v-btn to="/wheels/new" color="primary" variant="flat" size="large" :prepend-icon="mdiPlus">Create a wheel</v-btn>
      </v-sheet>

      <v-row v-else>
        <v-col v-for="w in shown" :key="w.id" cols="12" sm="6" lg="4">
          <WheelCard :wheel="w" />
        </v-col>
      </v-row>
    </section>
  </v-container>
</template>

<style scoped>
.hero-copy {
  max-width: 560px;
  font-size: 1.15rem !important;
}

.hero-wheel {
  width: min(380px, 80vw);
}

.hero-wheel :deep(.rotor) {
  animation: idle-spin 40s linear infinite;
}

@keyframes idle-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .hero-wheel :deep(.rotor) {
    animation: none;
  }
}
</style>
