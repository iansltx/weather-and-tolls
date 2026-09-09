import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// The frontend version is baked in at build time (Docker passes VERSION);
// the server publishes its own copy over Mercure so open pages can detect
// new deployments.
const version = process.env.VITE_APP_VERSION ?? 'dev'

export default defineConfig({
	plugins: [vue()],
	define: {
		__APP_VERSION__: JSON.stringify(version),
	},
	build: {
		// Emit a manifest so the PHP front controller can reference the hashed
		// asset names.
		manifest: true,
	},
	server: {
		// Local development: proxy API calls and the Mercure hub to the
		// FrankenPHP server (docker compose up, or a locally run binary).
		proxy: {
			'/api': 'http://localhost:8080',
			'/.well-known/mercure': {
				target: 'http://localhost:8080',
				ws: false,
			},
		},
	},
})