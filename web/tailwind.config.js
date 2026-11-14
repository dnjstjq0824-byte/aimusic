/** @type {import('tailwindcss').Config} */
export default {
  content: ['./app/**/*.{vue,ts}'],
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#ecfdff',
          100: '#cffafe',
          200: '#a5f3fc',
          300: '#67e8f9',
          400: '#22d3ee',
          500: '#00d4ff', // Core signal cyan
          600: '#00b8db',
          700: '#0b7a94',
          800: '#155e75',
          900: '#164e63',
          950: '#083344',
        },
        surface: {
          0: '#08080d',       // Near-black studio canvas
          1: '#0e0e15',       // Card / Sidebar background
          2: '#16161e',       // Elevated surface / Dropdown
          3: '#202029',       // Hover surface
          border: '#23232e',  // Subtle hairline border
          'border-hover': '#3a3a46',
        },
        brandteal: {
          DEFAULT: '#0a141b',
          raised: '#10232e',
        },
        accent: {
          cyan: '#00d4ff',
          emerald: '#10b981',
          amber: '#f59e0b',
          crimson: '#ef4444',
        },
        text: {
          primary: 'var(--color-text-primary)',
          secondary: 'var(--color-text-secondary)',
          muted: 'var(--color-text-muted)',
        },
        verified: 'var(--color-verified)',
        success: 'var(--color-success)',
        'on-brand': 'var(--color-on-brand)',
      },
      fontSize: {
        caption: ['0.75rem', { lineHeight: '1rem' }],
        'body-sm': ['0.875rem', { lineHeight: '1.25rem' }],
        body: ['1rem', { lineHeight: '1.5rem' }],
        h3: ['1.25rem', { lineHeight: '1.75rem' }],
        h2: ['1.75rem', { lineHeight: '2.25rem' }],
        h1: ['2.5rem', { lineHeight: '3rem' }],
        display: ['3.5rem', { lineHeight: '4rem' }],
      },
      fontFamily: {
        sans: ['var(--font-sans)', '-apple-system', 'BlinkMacSystemFont', 'system-ui', 'sans-serif'],
        display: ['var(--font-display)', '-apple-system', 'BlinkMacSystemFont', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'Consolas', 'monospace'],
      },
      boxShadow: {
        'glow-brand': '0 0 25px -5px rgba(0, 212, 255, 0.35)',
        'glow-brand-lg': '0 0 40px -8px rgba(0, 212, 255, 0.45)',
        'glow-emerald': '0 0 25px -5px rgba(16, 185, 129, 0.35)',
        'subtle-elevated': '0 10px 30px -10px rgba(0, 0, 0, 0.8), 0 0 1px 1px rgba(255, 255, 255, 0.05)',
      },
      animation: {
        shimmer: 'shimmer 2s linear infinite',
        'pulse-subtle': 'pulseSubtle 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
      keyframes: {
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
        pulseSubtle: {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.6' },
        },
      },
    },
  },
  plugins: [],
};
