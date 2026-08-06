import type { Config } from "tailwindcss";

const config: Config = {
  darkMode: ["class"],
  content: [
    "./pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        border:     "hsl(var(--border))",
        input:      "hsl(var(--input))",
        ring:       "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT:    "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT:    "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        destructive: {
          DEFAULT:    "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        muted: {
          DEFAULT:    "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        accent: {
          DEFAULT:    "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        card: {
          DEFAULT:    "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
        // «Вольт» — фирменный неоново-салатовый (Stadium Night + Volt).
        // Шкала derive-ится из --brand-* (globals.css): один источник цвета,
        // opacity-модификаторы (yellow-400/10) работают через <alpha-value>.
        yellow: {
          DEFAULT: "hsl(var(--brand-500) / <alpha-value>)",
          50:  "#f8ffe1",
          400: "hsl(var(--brand-400) / <alpha-value>)",
          500: "hsl(var(--brand-500) / <alpha-value>)",
          600: "hsl(var(--brand-600) / <alpha-value>)",
        },
        brand: "hsl(var(--brand-500) / <alpha-value>)",
        // Семантические поверхности (архитектура тем): bg-surface-1, border-edge…
        // Значения из globals.css (--surface-*, --edge*) — светлая тема их переопределит.
        "surface-1": "hsl(var(--surface-1))",
        "surface-2": "hsl(var(--surface-2))",
        edge:        "hsl(var(--edge))",
        "edge-top":  "hsl(var(--edge-top))",
        // Золото — только для чемпионов и трофеев.
        gold: {
          DEFAULT: "#f59e0b",
          300: "#fde047",
          400: "#fbbf24",
          500: "#f59e0b",
        },
        // zinc-500 чуть светлее стандартного (#71717a): на zinc-900
        // даёт контраст ≥4.5:1 (WCAG AA для мелкого текста).
        zinc: {
          500: "#8b8b96",
        },
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      fontFamily: {
        sans: ["'Inter Variable'", "'Inter Fallback'", "Inter", "ui-sans-serif", "system-ui", "-apple-system", "sans-serif"],
        display: ["'Unbounded Variable'", "'Unbounded Fallback'", "'Inter Variable'", "ui-sans-serif", "system-ui", "sans-serif"],
      },
      keyframes: {
        "accordion-down": { from: { height: "0" }, to: { height: "var(--radix-accordion-content-height)" } },
        "accordion-up":   { from: { height: "var(--radix-accordion-content-height)" }, to: { height: "0" } },
        "fade-in":    { from: { opacity: "0", transform: "translateY(4px)" }, to: { opacity: "1", transform: "translateY(0)" } },
        "slide-in":   { from: { opacity: "0", transform: "translateX(-8px)" }, to: { opacity: "1", transform: "translateX(0)" } },
        shimmer: { from: { backgroundPosition: "-200% 0" }, to: { backgroundPosition: "200% 0" } },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up":   "accordion-up 0.2s ease-out",
        "fade-in":  "fade-in 0.25s ease-out",
        "slide-in": "slide-in 0.2s ease-out",
        shimmer: "shimmer 2s infinite linear",
      },
    },
  },
  plugins: [],
};

export default config;
