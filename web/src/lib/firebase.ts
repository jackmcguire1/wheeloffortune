import { initializeApp } from "firebase/app";
import { connectAuthEmulator, getAuth } from "firebase/auth";

const projectId = import.meta.env.VITE_FIREBASE_PROJECT_ID || "demo-wheeloffortune";

export const firebaseApp = initializeApp({
  apiKey: import.meta.env.VITE_FIREBASE_API_KEY || "demo-api-key",
  authDomain: `${projectId}.firebaseapp.com`,
  projectId,
});

export const auth = getAuth(firebaseApp);

// An explicit empty value opts out of the emulator and talks to real Firebase.
const emulatorUrl = import.meta.env.VITE_FIREBASE_AUTH_EMULATOR_URL ?? "http://127.0.0.1:9099";
export const usingEmulator = emulatorUrl !== "";
if (usingEmulator) {
  connectAuthEmulator(auth, emulatorUrl, { disableWarnings: true });
}
