import "vuetify/styles";
import { createVuetify } from "vuetify";
import { aliases, mdi } from "vuetify/iconsets/mdi-svg";

export default createVuetify({
  icons: { defaultSet: "mdi", aliases, sets: { mdi } },
  theme: {
    defaultTheme: "system",
    themes: {
      light: {
        dark: false,
        colors: {
          background: "#f6f3ff",
          surface: "#ffffff",
          "surface-variant": "#ece6ff",
          primary: "#6c3cf0",
          secondary: "#ff4f8b",
          accent: "#ffb627",
          info: "#3a86ff",
          success: "#06b48a",
          warning: "#ff9f1c",
          error: "#e63946",
        },
      },
      dark: {
        dark: true,
        colors: {
          background: "#0f0a1f",
          surface: "#1a1233",
          "surface-variant": "#2a1f4d",
          primary: "#a283ff",
          secondary: "#ff6fa3",
          accent: "#ffc94d",
          info: "#5aa0ff",
          success: "#2ed3a6",
          warning: "#ffb347",
          error: "#ff5c6c",
        },
      },
    },
  },
  defaults: {
    global: { ripple: true },
    VCard: { rounded: "xl", elevation: 0 },
    VBtn: { rounded: "lg", style: "text-transform: none; letter-spacing: 0;" },
    VTextField: { variant: "outlined", density: "comfortable", rounded: "lg" },
    VTextarea: { variant: "outlined", density: "comfortable", rounded: "lg" },
    VChip: { rounded: "lg" },
  },
});
