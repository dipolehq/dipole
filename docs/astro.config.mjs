// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
	integrations: [
		starlight({
			title: 'Dipole Docs',
			logo: {
				src: './src/assets/logo.svg',
			},
			social: [],
			sidebar: [
				{
					label: 'Get Started',
					items: [{ autogenerate: { directory: 'get-started' } }],
				},
				{
					label: 'User Guide',
					items: [{ autogenerate: { directory: 'user-guide' } }],
				},
				{
					label: 'CLI Guide',
					items: [{ autogenerate: { directory: 'cli-guide' } }],
				},
				{
					label: 'API Guide',
					items: [{ autogenerate: { directory: 'api-guide' } }],
				},
				{
					label: 'AI Guide',
					items: [{ autogenerate: { directory: 'ai-guide' } }],
				},
				{
					label: 'Reference',
					items: [{ autogenerate: { directory: 'reference' } }],
				},
			],
		}),
	],
});
