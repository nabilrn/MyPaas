import type { Config } from 'tailwindcss';

export default {
	darkMode: 'class',
	content: ['./src/**/*.{html,js,svelte,ts}'],
	theme: {
		extend: {
			colors: {
				brand: {
					50:  '#ecfdf5',
					100: '#d1fae5',
					500: '#10b981',
					600: '#059669',
					700: '#047857',
					900: '#064e3b'
				}
			},
			fontFamily: {
				sans: ['ui-sans-serif', 'system-ui', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'sans-serif']
			}
		}
	},
	plugins: []
} satisfies Config;
