import {
  createUserWithEmailAndPassword,
  GoogleAuthProvider,
  onAuthStateChanged,
  signInAnonymously,
  signInWithEmailAndPassword,
  signInWithPopup,
  signOut as fbSignOut,
  updateProfile,
  type User,
} from "firebase/auth";
import { defineStore } from "pinia";
import { computed, ref, shallowRef } from "vue";
import { auth } from "@/lib/firebase";

export const useAuthStore = defineStore("auth", () => {
  const user = shallowRef<User | null>(null);
  const ready = ref(false);
  const dialogOpen = ref(false);

  onAuthStateChanged(auth, (u) => {
    user.value = u;
    ready.value = true;
  });

  const signedIn = computed(() => user.value !== null);
  const displayName = computed(() => {
    const u = user.value;
    if (!u) return "";
    if (u.isAnonymous) return "Guest";
    return u.displayName || u.email?.split("@")[0] || "Player";
  });
  const initials = computed(() =>
    displayName.value
      .split(/\s+/)
      .map((p) => p[0])
      .join("")
      .slice(0, 2)
      .toUpperCase(),
  );

  async function withGoogle() {
    await signInWithPopup(auth, new GoogleAuthProvider());
    dialogOpen.value = false;
  }

  async function withEmail(email: string, password: string, create: boolean, name?: string) {
    if (create) {
      const cred = await createUserWithEmailAndPassword(auth, email, password);
      if (name) {
        await updateProfile(cred.user, { displayName: name });
        // Force a fresh token so the server sees the new display name.
        await cred.user.getIdToken(true);
        user.value = auth.currentUser;
      }
    } else {
      await signInWithEmailAndPassword(auth, email, password);
    }
    dialogOpen.value = false;
  }

  async function asGuest() {
    await signInAnonymously(auth);
    dialogOpen.value = false;
  }

  async function signOut() {
    await fbSignOut(auth);
  }

  /** Opens the sign-in dialog and resolves true once signed in. */
  function requireSignIn(): boolean {
    if (user.value) return true;
    dialogOpen.value = true;
    return false;
  }

  return { user, ready, dialogOpen, signedIn, displayName, initials, withGoogle, withEmail, asGuest, signOut, requireSignIn };
});
