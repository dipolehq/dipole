import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#f0f4ff',
          100: '#dbe4ff',
          200: '#bac8ff',
          300: '#91a7ff',
          400: '#748ffc',
          500: '#5c7cfa',
          600: '#4c6ef5',
          700: '#4263eb',
          800: '#3b5bdb',
          900: '#364fc7',
          950: '#2b3f9e',
        },
        surface: {
          DEFAULT: '#ffffff',
          secondary: '#f8f9fa',
          tertiary: '#f1f3f5',
        },
        border: {
          DEFAULT: '#e9ecef',
          strong: '#dee2e6',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      spacing: {
        18: '4.5rem',
        88: '22rem',
      },
    },
  },
  plugins: [],
} satisfies Config
