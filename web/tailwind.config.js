/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts}'],
  theme: {
    extend: {
      colors: {
        brand: { 500: '#6d5cff', 600: '#5748e8', 700: '#4538bd' },
      },
    },
  },
  plugins: [],
};
