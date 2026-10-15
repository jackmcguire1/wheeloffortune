<script setup lang="ts">
import { mdiGithub, mdiLogout, mdiPlus, mdiWeatherNight, mdiWhiteBalanceSunny } from "@mdi/js";
import { computed } from "vue";
import { useDisplay, useTheme } from "vuetify";
import SignInDialog from "@/components/SignInDialog.vue";
import { useAuthStore } from "@/stores/auth";
import { useNotify } from "@/stores/notify";

const theme = useTheme();
const { smAndDown } = useDisplay();
const authStore = useAuthStore();
const notify = useNotify();

const isDark = computed(() => theme.current.value.dark);

function toggleTheme() {
  theme.change(isDark.value ? "light" : "dark");
}
</script>

<template>
  <v-app>
    <v-app-bar flat class="glass" height="68">
      <template #prepend>
        <router-link to="/" class="brand d-flex align-center ml-2">
          <img src="/favicon.svg" alt="" width="36" height="36" class="brand-logo" />
          <span class="text-h6 font-weight-black ml-3 gradient-text">Wheel of Fortune</span>
        </router-link>
      </template>

      <template #append>
        <v-btn v-if="!smAndDown" to="/wheels/new" color="primary" variant="flat" :prepend-icon="mdiPlus" class="mr-2">New wheel</v-btn>
        <v-btn v-else to="/wheels/new" :icon="mdiPlus" color="primary" variant="flat" size="small" class="mr-1" aria-label="New wheel" />

        <v-btn :icon="isDark ? mdiWhiteBalanceSunny : mdiWeatherNight" variant="text" aria-label="Toggle theme" @click="toggleTheme" />

        <v-btn
          v-if="!smAndDown"
          :icon="mdiGithub"
          variant="text"
          href="https://github.com/jackmcguire1/wheeloffortune"
          target="_blank"
          aria-label="GitHub"
        />

        <template v-if="authStore.ready">
          <v-menu v-if="authStore.user" location="bottom end">
            <template #activator="{ props }">
              <v-btn v-bind="props" icon class="mr-2" aria-label="Account">
                <v-avatar size="38" color="secondary">
                  <v-img v-if="authStore.user.photoURL" :src="authStore.user.photoURL" />
                  <span v-else class="font-weight-bold">{{ authStore.initials }}</span>
                </v-avatar>
              </v-btn>
            </template>
            <v-list rounded="lg" min-width="220">
              <v-list-item :title="authStore.displayName" :subtitle="authStore.user.email || 'Anonymous guest'" />
              <v-divider class="my-1" />
              <v-list-item :prepend-icon="mdiLogout" title="Sign out" @click="authStore.signOut()" />
            </v-list>
          </v-menu>
          <v-btn v-else variant="tonal" class="mr-2" @click="authStore.dialogOpen = true">Sign in</v-btn>
        </template>
      </template>
    </v-app-bar>

    <v-main>
      <router-view v-slot="{ Component }">
        <v-fade-transition mode="out-in">
          <component :is="Component" />
        </v-fade-transition>
      </router-view>
    </v-main>

    <SignInDialog />

    <v-snackbar v-model="notify.open" :color="notify.color" rounded="lg" location="bottom" timeout="4000">
      {{ notify.text }}
    </v-snackbar>
  </v-app>
</template>

<style scoped>
.brand {
  text-decoration: none;
  color: inherit;
}

.brand-logo {
  transition: transform 0.6s cubic-bezier(0.2, 0.9, 0.3, 1.2);
}

.brand:hover .brand-logo {
  transform: rotate(240deg);
}
</style>
