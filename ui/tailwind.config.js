/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['"Google Sans"', '"Google Sans Text"', '"Product Sans"', 'Roboto', 'Inter', 'system-ui', 'sans-serif'],
      },
      borderRadius: { DEFAULT: '6px', md: '6px', lg: '8px', xl: '8px' },
    },
  },
  plugins: [],
}
