import adapter from '@sveltejs/adapter-node';

export default {
	kit: {
		adapter: adapter(),
		version: {
			pollInterval: 60_000
		},
		alias: {
			$lib: 'src/lib',
			$components: 'src/lib/components',
			$stores: 'src/lib/stores',
			$types: 'src/lib/types',
			$utils: 'src/lib/utils',
			$api: 'src/lib/api'
		}
	}
};
