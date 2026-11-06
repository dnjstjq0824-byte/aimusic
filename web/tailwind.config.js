/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts}'],
  theme: {
    extend: {
      colors: {
        brand: { 500: '#FF6B3D', 600: '#E5522B', 700: '#B93F22' },
      },
    },
  },
  plugins: [],
};
