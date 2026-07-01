import type { Config } from 'tailwindcss';

export default {
	darkMode: 'class',
	content: ['./src/**/*.{html,js,svelte,ts}'],
	theme: {
		extend: {
			colors: {
				brand: {
					50:  '#eff6ff',
					100: '#dbeafe',
					500: '#3b82f6',
					600: '#2563eb',
					700: '#1d4ed8',
					900: '#1e3a8a'
				}
			},
			fontFamily: {
				sans: ['Inter', 'ui-sans-serif', 'system-ui', 'sans-serif']
			}
		}
	},
	plugins: []
} satisfies Config;
