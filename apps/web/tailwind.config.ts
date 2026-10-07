import type { Config } from "tailwindcss";
export default {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        brand: {
          deep: "#0B3B60",
          blue: "#1D6FA5",
          orange: "#FF6B00",
          light: "#FFF2EB",
        },
      },
    },
  },
  plugins: [],
} satisfies Config;
