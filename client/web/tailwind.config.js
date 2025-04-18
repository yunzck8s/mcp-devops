/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: "#3498db",
        secondary: "#2ecc71",
        accent: "#e74c3c",
        neutral: "#191D24",
        "base-100": "#ffffff",
        info: "#3ABFF8",
        success: "#36D399",
        warning: "#FBBD23",
        error: "#F87272",
      },
    },
  },
  plugins: [require("daisyui")],
  daisyui: {
    themes: [
      {
        light: {
          ...require("daisyui/src/theming/themes")["[data-theme=light]"],
          primary: "#3498db",
          secondary: "#2ecc71",
          accent: "#e74c3c",
          "primary-focus": "#2980b9",
        },
        dark: {
          ...require("daisyui/src/theming/themes")["[data-theme=dark]"],
          primary: "#3498db",
          secondary: "#2ecc71",
          accent: "#e74c3c",
          "primary-focus": "#2980b9",
        },
      },
    ],
  },
}
