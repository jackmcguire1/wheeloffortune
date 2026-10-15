<script setup lang="ts">
import { mdiArrowDown, mdiArrowUp, mdiCheck, mdiDeleteOutline, mdiInfinity, mdiPlus, mdiShuffleVariant } from "@mdi/js";
import { computed, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import FortuneWheel from "@/components/FortuneWheel.vue";
import { WheelMode } from "@/gen/wheel/v1/wheel_pb";
import { wheelClient } from "@/lib/api";
import { MODES, modeInfo } from "@/lib/modes";
import { colorFor, PALETTE } from "@/lib/palette";
import { formatPercent, probabilities } from "@/lib/probability";
import { useAuthStore } from "@/stores/auth";
import { useNotify } from "@/stores/notify";

interface Draft {
  key: number;
  label: string;
  color: string;
  weight: number;
  limited: boolean;
  stock: number;
}

const LIMITS = { name: 60, description: 280, label: 40, minSegments: 2, maxSegments: 24, maxWeight: 10_000, maxStock: 1_000_000 };

const router = useRouter();
const authStore = useAuthStore();
const notify = useNotify();

let nextKey = 0;
function seg(label: string, opts: Partial<Omit<Draft, "key" | "label">> = {}): Draft {
  const key = nextKey++;
  return { key, label, color: opts.color ?? PALETTE[key % PALETTE.length], weight: opts.weight ?? 1, limited: opts.limited ?? false, stock: opts.stock ?? 10 };
}

const form = reactive({
  name: "",
  description: "",
  mode: WheelMode.EQUAL as WheelMode,
  segments: [seg("Option A"), seg("Option B"), seg("Option C"), seg("Option D")] as Draft[],
});

const templates: { title: string; apply: () => void }[] = [
  {
    title: "Yes / No",
    apply: () => {
      form.name ||= "Yes or no?";
      form.mode = WheelMode.EQUAL;
      form.segments = [seg("Yes", { color: "#06d6a0" }), seg("No", { color: "#e63946" })];
    },
  },
  {
    title: "Lunch picker",
    apply: () => {
      form.name ||= "What's for lunch?";
      form.mode = WheelMode.WEIGHTED;
      form.segments = [seg("Pizza", { weight: 3 }), seg("Sushi", { weight: 2 }), seg("Tacos", { weight: 2 }), seg("Salad", { weight: 1 }), seg("Burgers", { weight: 2 }), seg("Ramen", { weight: 2 })];
    },
  },
  {
    title: "Prize draw",
    apply: () => {
      form.name ||= "Launch day prize draw";
      form.mode = WheelMode.INVENTORY;
      form.segments = [
        seg("Grand prize", { limited: true, stock: 1, color: "#ffb627" }),
        seg("T-shirt", { limited: true, stock: 20 }),
        seg("Sticker pack", { limited: true, stock: 50 }),
        seg("Coffee voucher", { limited: true, stock: 30 }),
        seg("Better luck next time", { limited: true, stock: 100, color: "#6c757d" }),
      ];
    },
  },
  {
    title: "Rare loot",
    apply: () => {
      form.name ||= "Loot box";
      form.mode = WheelMode.WEIGHTED;
      form.segments = [
        seg("Common", { weight: 70, color: "#adb5bd" }),
        seg("Rare", { weight: 22, color: "#3a86ff" }),
        seg("Epic", { weight: 7, color: "#8338ec" }),
        seg("Legendary", { weight: 1, limited: true, stock: 3, color: "#ffb627" }),
      ];
    },
  },
];

const isWeighted = computed(() => form.mode === WheelMode.WEIGHTED);
const isInventory = computed(() => form.mode === WheelMode.INVENTORY);

const chances = computed(() =>
  probabilities(
    form.mode,
    form.segments.map((s) => ({ weight: s.weight, stock: isInventory.value || s.limited ? s.stock : undefined })),
  ),
);

const preview = computed(() => form.segments.map((s, i) => ({ label: s.label || `#${i + 1}`, color: colorFor(i, s.color) })));

const errors = computed(() => {
  const out: string[] = [];
  const name = form.name.trim();
  if (!name) out.push("Give your wheel a name.");
  if (name.length > LIMITS.name) out.push(`Name must be at most ${LIMITS.name} characters.`);
  if (form.segments.length < LIMITS.minSegments) out.push(`Add at least ${LIMITS.minSegments} segments.`);
  if (form.segments.length > LIMITS.maxSegments) out.push(`At most ${LIMITS.maxSegments} segments.`);
  form.segments.forEach((s, i) => {
    const n = `Segment ${i + 1}`;
    if (!s.label.trim()) out.push(`${n} needs a label.`);
    else if (s.label.trim().length > LIMITS.label) out.push(`${n}'s label is too long.`);
    if (isWeighted.value && (!Number.isInteger(s.weight) || s.weight < 1 || s.weight > LIMITS.maxWeight))
      out.push(`${n} needs a whole-number weight between 1 and ${LIMITS.maxWeight}.`);
    if ((isInventory.value || s.limited) && (!Number.isInteger(s.stock) || s.stock < 1 || s.stock > LIMITS.maxStock))
      out.push(`${n} needs a stock between 1 and ${LIMITS.maxStock.toLocaleString()}.`);
  });
  return out;
});

const rules = {
  label: (v: string) => (!!v.trim() && v.trim().length <= LIMITS.label) || `1–${LIMITS.label} characters`,
  weight: (v: number) => (Number.isInteger(+v) && +v >= 1 && +v <= LIMITS.maxWeight) || "1 or more",
  stock: (v: number) => (Number.isInteger(+v) && +v >= 1 && +v <= LIMITS.maxStock) || "1 or more",
};

function add() {
  if (form.segments.length >= LIMITS.maxSegments) return;
  form.segments.push(seg(`Option ${form.segments.length + 1}`, { limited: isInventory.value }));
}

function remove(i: number) {
  form.segments.splice(i, 1);
}

function move(i: number, dir: -1 | 1) {
  const j = i + dir;
  if (j < 0 || j >= form.segments.length) return;
  const [s] = form.segments.splice(i, 1);
  form.segments.splice(j, 0, s);
}

function shuffleColors() {
  const offset = Math.floor(Math.random() * PALETTE.length);
  form.segments.forEach((s, i) => (s.color = PALETTE[(i + offset) % PALETTE.length]));
}

const saving = ref(false);
const triedSubmit = ref(false);

async function save() {
  triedSubmit.value = true;
  if (errors.value.length) return;
  if (!authStore.requireSignIn()) return;
  saving.value = true;
  try {
    const { wheel } = await wheelClient.createWheel({
      name: form.name.trim(),
      description: form.description.trim(),
      mode: form.mode,
      segments: form.segments.map((s) => ({
        label: s.label.trim(),
        color: s.color,
        weight: isWeighted.value ? Number(s.weight) : 1,
        stock: isInventory.value || s.limited ? Number(s.stock) : undefined,
      })),
    });
    notify.success("Wheel created. Give it a spin!");
    router.push(`/wheels/${wheel?.id}`);
  } catch (err) {
    notify.error(err);
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <v-container class="py-8" max-width="1280">
    <div class="d-flex flex-wrap align-end ga-4 mb-6">
      <div class="mr-auto">
        <div class="text-overline text-primary font-weight-bold">Wheel builder</div>
        <h1 class="text-h4 font-weight-black">Design your wheel</h1>
      </div>
      <div class="d-flex flex-wrap ga-2">
        <span class="text-caption text-medium-emphasis align-self-center mr-1">Start from:</span>
        <v-chip v-for="t in templates" :key="t.title" variant="outlined" @click="t.apply">{{ t.title }}</v-chip>
      </div>
    </div>

    <v-row>
      <v-col cols="12" md="7">
        <v-card class="pa-5 mb-6 glass">
          <v-text-field v-model="form.name" label="Wheel name" :counter="LIMITS.name" :maxlength="LIMITS.name + 10" />
          <v-textarea v-model="form.description" label="Description (optional)" rows="2" auto-grow :counter="LIMITS.description" :maxlength="LIMITS.description" />

          <div class="text-subtitle-1 font-weight-bold mb-2">How should it pick a winner?</div>
          <v-item-group v-model="form.mode" mandatory>
            <v-row dense>
              <v-col v-for="m in MODES" :key="m.mode" cols="12" sm="4">
                <v-item v-slot="{ isSelected, toggle }" :value="m.mode">
                  <v-card
                    :variant="isSelected ? 'tonal' : 'outlined'"
                    :color="isSelected ? m.color : undefined"
                    class="mode-card pa-4 h-100"
                    @click="toggle"
                  >
                    <div class="d-flex align-center mb-2">
                      <v-icon :icon="m.icon" class="mr-2" />
                      <span class="font-weight-bold">{{ m.title }}</span>
                      <v-spacer />
                      <v-icon v-if="isSelected" :icon="mdiCheck" size="small" />
                    </div>
                    <div class="text-caption text-medium-emphasis">{{ m.description }}</div>
                  </v-card>
                </v-item>
              </v-col>
            </v-row>
          </v-item-group>
        </v-card>

        <v-card class="pa-5 glass">
          <div class="d-flex align-center mb-4">
            <div class="text-subtitle-1 font-weight-bold mr-auto">
              Segments <span class="text-medium-emphasis font-weight-regular">({{ form.segments.length }}/{{ LIMITS.maxSegments }})</span>
            </div>
            <v-btn variant="text" size="small" :prepend-icon="mdiShuffleVariant" @click="shuffleColors">Colours</v-btn>
          </div>

          <v-slide-y-transition group tag="div">
            <div v-for="(s, i) in form.segments" :key="s.key" class="segment-row d-flex flex-wrap align-center ga-2 mb-3">
              <v-menu :close-on-content-click="false">
                <template #activator="{ props }">
                  <v-btn v-bind="props" icon size="small" variant="flat" :style="{ background: s.color }" :aria-label="`Colour for segment ${i + 1}`" />
                </template>
                <v-color-picker v-model="s.color" mode="hex" :swatches="[PALETTE.slice(0, 6), PALETTE.slice(6)]" show-swatches />
              </v-menu>

              <v-text-field v-model="s.label" :label="`Segment ${i + 1}`" :rules="[rules.label]" hide-details="auto" class="flex-1-1 label-field" density="compact" />

              <v-text-field
                v-if="isWeighted"
                v-model.number="s.weight"
                label="Weight"
                type="number"
                min="1"
                :rules="[rules.weight]"
                hide-details="auto"
                density="compact"
                class="num-field"
              />

              <template v-if="isInventory || s.limited">
                <v-text-field
                  v-model.number="s.stock"
                  label="Stock"
                  type="number"
                  min="1"
                  :rules="[rules.stock]"
                  hide-details="auto"
                  density="compact"
                  class="num-field"
                />
              </template>
              <v-btn
                v-if="!isInventory"
                :icon="mdiInfinity"
                size="small"
                :variant="s.limited ? 'text' : 'tonal'"
                :color="s.limited ? undefined : 'primary'"
                :aria-label="s.limited ? 'Make unlimited' : 'Limit stock'"
                v-tooltip="s.limited ? 'Limited stock, click for unlimited' : 'Unlimited, click to limit stock'"
                @click="s.limited = !s.limited"
              />

              <div class="d-flex">
                <v-btn :icon="mdiArrowUp" size="small" variant="text" :disabled="i === 0" aria-label="Move up" @click="move(i, -1)" />
                <v-btn :icon="mdiArrowDown" size="small" variant="text" :disabled="i === form.segments.length - 1" aria-label="Move down" @click="move(i, 1)" />
                <v-btn :icon="mdiDeleteOutline" size="small" variant="text" color="error" :disabled="form.segments.length <= LIMITS.minSegments" aria-label="Remove" @click="remove(i)" />
              </div>
            </div>
          </v-slide-y-transition>

          <v-btn block variant="tonal" :prepend-icon="mdiPlus" :disabled="form.segments.length >= LIMITS.maxSegments" @click="add">Add segment</v-btn>
        </v-card>
      </v-col>

      <v-col cols="12" md="5">
        <div class="sticky">
          <v-card class="pa-5 glass">
            <div class="d-flex align-center mb-4">
              <div class="text-subtitle-1 font-weight-bold mr-auto">Live preview</div>
              <v-chip size="small" :color="modeInfo(form.mode).color" variant="tonal" :prepend-icon="modeInfo(form.mode).icon">
                {{ modeInfo(form.mode).short }}
              </v-chip>
            </div>
            <div class="preview-wheel mx-auto mb-5">
              <FortuneWheel :segments="preview" :probabilities="chances" :mode="form.mode" />
            </div>

            <v-table density="compact" class="bg-transparent">
              <thead>
                <tr>
                  <th>Segment</th>
                  <th class="text-right">Chance</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(s, i) in form.segments" :key="s.key">
                  <td>
                    <span class="dot mr-2" :style="{ background: s.color }" />{{ s.label || `#${i + 1}` }}
                  </td>
                  <td class="text-right mono">{{ formatPercent(chances[i] ?? 0) }}</td>
                </tr>
              </tbody>
            </v-table>

            <v-expand-transition>
              <v-alert v-if="triedSubmit && errors.length" type="error" variant="tonal" rounded="lg" class="mt-4" density="compact">
                <div v-for="e in errors.slice(0, 4)" :key="e">{{ e }}</div>
                <div v-if="errors.length > 4">…and {{ errors.length - 4 }} more.</div>
              </v-alert>
            </v-expand-transition>

            <v-btn block size="x-large" color="primary" variant="flat" class="mt-4" :loading="saving" @click="save">
              {{ authStore.signedIn ? "Create wheel" : "Sign in & create" }}
            </v-btn>
          </v-card>
        </div>
      </v-col>
    </v-row>
  </v-container>
</template>

<style scoped>
.sticky {
  position: sticky;
  top: 84px;
}

.preview-wheel {
  max-width: 320px;
}

.mode-card {
  cursor: pointer;
  transition: transform 0.15s ease;
}

.mode-card.v-card--variant-outlined {
  border-color: rgba(var(--v-border-color), 0.16);
}

.mode-card:hover {
  transform: translateY(-2px);
}

.label-field {
  min-width: 160px;
}

.num-field {
  max-width: 110px;
  min-width: 90px;
}

.dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  vertical-align: middle;
}

.segment-row {
  padding-bottom: 12px;
  border-bottom: 1px dashed rgba(var(--v-border-color), 0.12);
}
</style>
