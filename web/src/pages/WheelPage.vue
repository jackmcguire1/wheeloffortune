<script setup lang="ts">
import {
  mdiArrowLeft,
  mdiDeleteOutline,
  mdiInfinity,
  mdiPackageVariant,
  mdiPause,
  mdiPlay,
  mdiRefresh,
  mdiShareVariant,
  mdiTrophy,
} from "@mdi/js";
import confetti from "canvas-confetti";
import { computed, onBeforeUnmount, onMounted, ref, shallowRef, useTemplateRef, watch } from "vue";
import { useRouter } from "vue-router";
import FortuneWheel from "@/components/FortuneWheel.vue";
import { WheelMode, type Segment, type Spin, type Wheel } from "@/gen/wheel/v1/wheel_pb";
import { friendlyError, wheelClient } from "@/lib/api";
import { modeInfo } from "@/lib/modes";
import { colorFor } from "@/lib/palette";
import { formatPercent } from "@/lib/probability";
import { timeAgo } from "@/lib/time";
import { useAuthStore } from "@/stores/auth";
import { useNotify } from "@/stores/notify";

const props = defineProps<{ id: string }>();

const router = useRouter();
const authStore = useAuthStore();
const notify = useNotify();
const wheelRef = useTemplateRef<InstanceType<typeof FortuneWheel>>("wheelRef");

const wheel = shallowRef<Wheel | null>(null);
const spins = shallowRef<Spin[]>([]);
const loadError = ref("");
const spinning = ref(false);
const busy = ref(false);
const result = shallowRef<Segment | null>(null);
const resultOpen = ref(false);
const confirmDelete = ref(false);

const isOwner = computed(() => !!wheel.value && authStore.user?.uid === wheel.value.ownerUid);
const mode = computed(() => modeInfo(wheel.value?.mode ?? WheelMode.EQUAL));
const segments = computed(() => wheel.value?.segments.map((s) => ({ label: s.spec?.label ?? "", color: s.spec?.color })) ?? []);
const probs = computed(() => wheel.value?.segments.map((s) => s.probability) ?? []);
const canSpin = computed(() => !!wheel.value && wheel.value.enabled && !wheel.value.exhausted && !spinning.value);

const spinLabel = computed(() => {
  if (!wheel.value) return "Spin";
  if (wheel.value.exhausted) return "All prizes won";
  if (!wheel.value.enabled) return "Paused";
  if (spinning.value) return "Spinning…";
  return authStore.signedIn ? "Spin the wheel" : "Sign in to spin";
});

async function load() {
  try {
    const [w, s] = await Promise.all([wheelClient.getWheel({ id: props.id }), wheelClient.listSpins({ id: props.id, limit: 20 })]);
    // Don't swap the layout out from under a spin in progress.
    if (!spinning.value) wheel.value = w.wheel ?? null;
    spins.value = s.spins;
    loadError.value = "";
  } catch (err) {
    loadError.value = friendlyError(err);
  }
}

async function spin() {
  if (!canSpin.value || !wheel.value) return;
  if (!authStore.requireSignIn()) return;
  spinning.value = true;
  try {
    const res = await wheelClient.spinWheel({ id: props.id });
    await wheelRef.value?.spinTo(res.segmentIndex);
    result.value = res.segment ?? null;
    if (res.wheel) wheel.value = res.wheel;
    if (res.spin) spins.value = [res.spin, ...spins.value].slice(0, 20);
    resultOpen.value = true;
    celebrate(colorFor(res.segmentIndex, res.segment?.spec?.color));
  } catch (err) {
    notify.error(err);
    load();
  } finally {
    spinning.value = false;
  }
}

function celebrate(color: string) {
  if (window.matchMedia?.("(prefers-reduced-motion: reduce)").matches) return;
  const colors = [color, "#ffffff", "#ffb627", "#ff4f8b"];
  confetti({ particleCount: 140, spread: 90, origin: { y: 0.55 }, colors, zIndex: 3000 });
  window.setTimeout(() => {
    confetti({ particleCount: 60, angle: 60, spread: 60, origin: { x: 0 }, colors, zIndex: 3000 });
    confetti({ particleCount: 60, angle: 120, spread: 60, origin: { x: 1 }, colors, zIndex: 3000 });
  }, 250);
}

async function ownerAction(fn: () => Promise<{ wheel?: Wheel }>, message: string) {
  busy.value = true;
  try {
    const res = await fn();
    if (res.wheel) wheel.value = res.wheel;
    notify.success(message);
  } catch (err) {
    notify.error(err);
  } finally {
    busy.value = false;
  }
}

const toggleEnabled = () =>
  ownerAction(
    () => wheelClient.setWheelEnabled({ id: props.id, enabled: !wheel.value?.enabled }),
    wheel.value?.enabled ? "Wheel paused." : "Wheel is live again.",
  );

const restock = () => ownerAction(() => wheelClient.restockWheel({ id: props.id }), "Stock restored.");

async function remove() {
  busy.value = true;
  try {
    await wheelClient.deleteWheel({ id: props.id });
    notify.success("Wheel deleted.");
    router.push("/");
  } catch (err) {
    notify.error(err);
  } finally {
    busy.value = false;
    confirmDelete.value = false;
  }
}

async function share() {
  const url = window.location.href;
  try {
    if (navigator.share) await navigator.share({ title: wheel.value?.name, url });
    else {
      await navigator.clipboard.writeText(url);
      notify.info("Link copied.");
    }
  } catch {
    /* user cancelled */
  }
}

function onKey(e: KeyboardEvent) {
  const target = e.target as HTMLElement | null;
  if (e.code === "Space" && !resultOpen.value && !target?.closest("input, textarea, button, [role=dialog]")) {
    e.preventDefault();
    spin();
  }
}

let timer: number | undefined;
onMounted(() => {
  load();
  timer = window.setInterval(() => !spinning.value && !document.hidden && load(), 10_000);
  window.addEventListener("keydown", onKey);
});
onBeforeUnmount(() => {
  window.clearInterval(timer);
  window.removeEventListener("keydown", onKey);
});
watch(() => props.id, load);
</script>

<template>
  <v-container class="py-6" max-width="1280">
    <v-btn to="/" variant="text" :prepend-icon="mdiArrowLeft" class="mb-2 ml-n3">All wheels</v-btn>

    <v-alert v-if="loadError && !wheel" type="error" variant="tonal" rounded="xl" :text="loadError">
      <template #append><v-btn variant="text" @click="load">Retry</v-btn></template>
    </v-alert>

    <template v-else-if="!wheel">
      <v-row>
        <v-col cols="12" md="7"><v-skeleton-loader type="image" height="520" class="rounded-xl" /></v-col>
        <v-col cols="12" md="5"><v-skeleton-loader type="article, list-item-three-line@3" class="rounded-xl" /></v-col>
      </v-row>
    </template>

    <v-row v-else>
      <v-col cols="12" md="7">
        <div class="d-flex flex-wrap align-center ga-2 mb-1">
          <h1 class="text-h4 text-sm-h3 font-weight-black mr-2">{{ wheel.name }}</h1>
          <v-chip :color="mode.color" variant="tonal" :prepend-icon="mode.icon">{{ mode.short }}</v-chip>
          <v-chip v-if="!wheel.enabled" color="warning" variant="tonal" :prepend-icon="mdiPause">Paused</v-chip>
          <v-btn :icon="mdiShareVariant" variant="text" size="small" aria-label="Share" @click="share" />
        </div>
        <p v-if="wheel.description" class="text-medium-emphasis mb-2">{{ wheel.description }}</p>
        <p class="text-caption text-medium-emphasis mb-6">
          {{ wheel.totalSpins }} spins<span v-if="wheel.ownerName"> · created by {{ wheel.ownerName }}</span>
        </p>

        <div class="stage">
          <div class="stage-wheel" :class="{ clickable: canSpin }" @click="spin">
            <FortuneWheel ref="wheelRef" :segments="segments" :probabilities="probs" :mode="wheel.mode" :disabled="!wheel.enabled || wheel.exhausted" />
          </div>
          <v-btn
            size="x-large"
            color="primary"
            variant="flat"
            rounded="pill"
            class="spin-btn mt-6 px-10"
            :disabled="!canSpin"
            :loading="spinning"
            @click="spin"
          >
            {{ spinLabel }}
          </v-btn>
          <div class="text-caption text-medium-emphasis mt-2 d-none d-md-block">or press <kbd>Space</kbd></div>
        </div>
      </v-col>

      <v-col cols="12" md="5">
        <v-card v-if="isOwner" class="pa-4 mb-4 glass">
          <div class="text-subtitle-2 font-weight-bold mb-3">Owner controls</div>
          <div class="d-flex flex-wrap ga-2">
            <v-btn :prepend-icon="wheel.enabled ? mdiPause : mdiPlay" variant="tonal" :loading="busy" @click="toggleEnabled">
              {{ wheel.enabled ? "Pause" : "Resume" }}
            </v-btn>
            <v-btn :prepend-icon="mdiRefresh" variant="tonal" :disabled="busy" @click="restock">Restock</v-btn>
            <v-btn :prepend-icon="mdiDeleteOutline" variant="tonal" color="error" :disabled="busy" @click="confirmDelete = true">Delete</v-btn>
          </div>
        </v-card>

        <v-card class="pa-4 mb-4 glass">
          <div class="text-subtitle-2 font-weight-bold mb-3">Odds right now</div>
          <div v-for="(s, i) in wheel.segments" :key="i" class="mb-3" :class="{ 'text-disabled': s.probability === 0 }">
            <div class="d-flex align-center text-body-2 mb-1">
              <span class="dot mr-2" :style="{ background: colorFor(i, s.spec?.color) }" />
              <span class="font-weight-medium text-truncate mr-auto">{{ s.spec?.label }}</span>
              <span class="text-caption text-medium-emphasis mr-3 d-flex align-center">
                <template v-if="s.remaining !== undefined">
                  <v-icon :icon="mdiPackageVariant" size="14" class="mr-1" />{{ s.remaining }}<span v-if="s.spec?.stock !== undefined">/{{ s.spec.stock }}</span>
                </template>
                <v-icon v-else :icon="mdiInfinity" size="14" />
                <v-icon :icon="mdiTrophy" size="14" class="ml-3 mr-1" />{{ s.wins }}
              </span>
              <span class="mono text-body-2 pct">{{ formatPercent(s.probability) }}</span>
            </div>
            <v-progress-linear :model-value="s.probability * 100" :color="colorFor(i, s.spec?.color)" rounded height="8" bg-opacity="0.12" />
          </div>
        </v-card>

        <v-card class="pa-4 glass">
          <div class="text-subtitle-2 font-weight-bold mb-2">Recent spins</div>
          <div v-if="!spins.length" class="text-medium-emphasis text-body-2 py-4 text-center">No spins yet. Be the first!</div>
          <v-list v-else density="compact" class="bg-transparent pa-0 feed">
            <v-slide-y-transition group>
              <v-list-item v-for="s in spins" :key="`${s.spinTime?.seconds}-${s.spinTime?.nanos}-${s.userUid}`" class="px-0">
                <template #prepend>
                  <v-avatar size="28" :color="colorFor(s.segmentIndex, wheel.segments[s.segmentIndex]?.spec?.color)" class="mr-3">
                    <v-icon :icon="mdiTrophy" size="14" color="white" />
                  </v-avatar>
                </template>
                <v-list-item-title class="text-body-2">
                  <strong>{{ s.userName || "Someone" }}</strong> won <strong>{{ s.segmentLabel }}</strong>
                </v-list-item-title>
                <v-list-item-subtitle class="text-caption">{{ timeAgo(s.spinTime) }}</v-list-item-subtitle>
              </v-list-item>
            </v-slide-y-transition>
          </v-list>
        </v-card>
      </v-col>
    </v-row>

    <v-dialog v-model="resultOpen" max-width="420">
      <v-card v-if="result" class="text-center pa-6 result-card">
        <div class="text-overline text-medium-emphasis">You landed on</div>
        <div class="text-h3 font-weight-black my-3 gradient-text">{{ result.spec?.label }}</div>
        <div v-if="result.remaining !== undefined" class="text-body-2 text-medium-emphasis">{{ result.remaining }} left in the pool</div>
        <v-card-actions class="justify-center mt-4">
          <v-btn variant="text" @click="resultOpen = false">Close</v-btn>
          <v-btn
            color="primary"
            variant="flat"
            :disabled="!wheel?.enabled || wheel?.exhausted"
            @click="
              resultOpen = false;
              spin();
            "
          >
            Spin again
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="confirmDelete" max-width="400">
      <v-card class="pa-2">
        <v-card-title class="font-weight-bold">Delete this wheel?</v-card-title>
        <v-card-text>All segments, stock and spin history for “{{ wheel?.name }}” will be removed. This can't be undone.</v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="confirmDelete = false">Cancel</v-btn>
          <v-btn color="error" variant="flat" :loading="busy" @click="remove">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<style scoped>
.stage {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.stage-wheel {
  width: min(560px, 100%);
  padding-top: 3%;
}

.stage-wheel.clickable {
  cursor: pointer;
}

.spin-btn {
  font-weight: 800;
  font-size: 1.15rem;
  box-shadow: 0 12px 30px -10px rgba(108, 60, 240, 0.7);
}

.dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.pct {
  min-width: 48px;
  text-align: right;
}

.feed {
  max-height: 360px;
  overflow-y: auto;
}

.result-card {
  background: linear-gradient(160deg, rgba(var(--v-theme-primary), 0.1), rgba(var(--v-theme-secondary), 0.08)), rgb(var(--v-theme-surface));
}

kbd {
  font-family: "JetBrains Mono", monospace;
  padding: 1px 6px;
  border-radius: 6px;
  border: 1px solid rgba(var(--v-border-color), 0.2);
}
</style>
