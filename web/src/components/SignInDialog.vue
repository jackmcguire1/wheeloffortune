<script setup lang="ts">
import { mdiAccountOutline, mdiEmailOutline, mdiGoogle, mdiIncognito, mdiLockOutline } from "@mdi/js";
import { FirebaseError } from "firebase/app";
import { ref } from "vue";
import { usingEmulator } from "@/lib/firebase";
import { useAuthStore } from "@/stores/auth";

const authStore = useAuthStore();

const tab = ref<"signin" | "signup">("signin");
const email = ref("");
const password = ref("");
const name = ref("");
const busy = ref(false);
const error = ref("");

const rules = {
  email: (v: string) => /.+@.+\..+/.test(v) || "Enter a valid email",
  password: (v: string) => v.length >= 6 || "At least 6 characters",
};

function message(err: unknown): string {
  if (err instanceof FirebaseError) {
    switch (err.code) {
      case "auth/invalid-credential":
      case "auth/wrong-password":
      case "auth/user-not-found":
        return "Wrong email or password.";
      case "auth/email-already-in-use":
        return "That email already has an account. Sign in instead.";
      case "auth/popup-closed-by-user":
        return "";
      case "auth/network-request-failed":
        return usingEmulator ? "Can't reach the auth emulator on :9099. Is it running?" : "Network error.";
    }
    return err.message;
  }
  return String(err);
}

async function run(fn: () => Promise<void>) {
  busy.value = true;
  error.value = "";
  try {
    await fn();
  } catch (err) {
    error.value = message(err);
  } finally {
    busy.value = false;
  }
}

function submit() {
  if (rules.email(email.value) !== true || rules.password(password.value) !== true) return;
  run(() => authStore.withEmail(email.value, password.value, tab.value === "signup", name.value.trim()));
}
</script>

<template>
  <v-dialog v-model="authStore.dialogOpen" max-width="440">
    <v-card class="pa-2">
      <v-card-item>
        <v-card-title class="text-h5 font-weight-bold">Join the game</v-card-title>
        <v-card-subtitle>Sign in to spin wheels and build your own.</v-card-subtitle>
      </v-card-item>

      <v-card-text>
        <v-alert v-if="usingEmulator" type="info" variant="tonal" density="compact" class="mb-4" rounded="lg">
          Using the Firebase Auth emulator. Any email works.
        </v-alert>

        <v-btn block size="large" variant="flat" color="primary" :prepend-icon="mdiGoogle" :loading="busy" @click="run(authStore.withGoogle)">
          Continue with Google
        </v-btn>
        <v-btn block size="large" variant="tonal" class="mt-3" :prepend-icon="mdiIncognito" :disabled="busy" @click="run(authStore.asGuest)">
          Play as a guest
        </v-btn>

        <div class="d-flex align-center my-5 text-medium-emphasis text-caption">
          <v-divider /><span class="mx-3">or with email</span><v-divider />
        </div>

        <v-tabs v-model="tab" density="compact" grow class="mb-4">
          <v-tab value="signin">Sign in</v-tab>
          <v-tab value="signup">Create account</v-tab>
        </v-tabs>

        <v-form @submit.prevent="submit">
          <v-text-field v-if="tab === 'signup'" v-model="name" label="Display name" :prepend-inner-icon="mdiAccountOutline" maxlength="40" />
          <v-text-field v-model="email" label="Email" type="email" autocomplete="email" :prepend-inner-icon="mdiEmailOutline" :rules="[rules.email]" />
          <v-text-field
            v-model="password"
            label="Password"
            type="password"
            :autocomplete="tab === 'signup' ? 'new-password' : 'current-password'"
            :prepend-inner-icon="mdiLockOutline"
            :rules="[rules.password]"
          />
          <v-alert v-if="error" type="error" variant="tonal" density="compact" class="mb-3" rounded="lg">{{ error }}</v-alert>
          <v-btn type="submit" block size="large" variant="outlined" :loading="busy">
            {{ tab === "signup" ? "Create account" : "Sign in" }}
          </v-btn>
        </v-form>
      </v-card-text>
    </v-card>
  </v-dialog>
</template>
