import { defineStore } from "pinia";
import { ref } from "vue";
import { friendlyError } from "@/lib/api";

export const useNotify = defineStore("notify", () => {
  const open = ref(false);
  const text = ref("");
  const color = ref("primary");

  function show(message: string, tone = "primary") {
    text.value = message;
    color.value = tone;
    open.value = true;
  }

  return {
    open,
    text,
    color,
    success: (m: string) => show(m, "success"),
    info: (m: string) => show(m, "primary"),
    error: (err: unknown) => show(typeof err === "string" ? err : friendlyError(err), "error"),
  };
});
