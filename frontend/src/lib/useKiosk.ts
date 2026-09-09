import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { StateSnapshot, VersionEvent } from '../types'

// Must match webapp/kioskapi/state.go.
const VERSION_TOPIC = 'https://kiosk.local/version'

// Baked in at build time (see vite.config.ts).
const APP_VERSION = __APP_VERSION__

const HUB_URL = '/.well-known/mercure'

function locationFromPageQuery(): string | undefined {
	const params = new URLSearchParams(window.location.search)
	return params.get('location') ?? undefined
}

export function useKiosk() {
	const snapshot = ref<StateSnapshot | null>(null)
	const showPayByMail = ref(false)
	const refreshing = ref(false)
	const updateAvailable = ref(false)
	const connected = ref(false)
	const now = ref(Date.now())

	let eventSource: EventSource | null = null

	function acceptVersion(version: string | undefined) {
		if (version && version !== APP_VERSION) {
			updateAvailable.value = true
		}
	}

	function handleStateEvent(data: string) {
		const parsed = JSON.parse(data) as StateSnapshot
		if (parsed.kind !== 'state') {
			return
		}

		snapshot.value = parsed
		refreshing.value = false
		acceptVersion(parsed.version)
	}

	function handleVersionEvent(data: string) {
		const parsed = JSON.parse(data) as VersionEvent
		if (parsed.kind !== 'version') {
			return
		}

		acceptVersion(parsed.version)
	}

	async function loadState() {
		const location = locationFromPageQuery()
		const query = location ? `?location=${encodeURIComponent(location)}` : ''

		const response = await fetch(`/api/state${query}`)
		if (!response.ok) {
			throw new Error(`state request failed: ${response.status}`)
		}

		handleStateEvent(await response.text())
	}

	function subscribe(topic: string) {
		const url = `${HUB_URL}?topic=${encodeURIComponent(topic)}&topic=${encodeURIComponent(VERSION_TOPIC)}`

		eventSource?.close()
		eventSource = new EventSource(url)

		eventSource.addEventListener('state', (event) => {
			connected.value = true
			handleStateEvent((event as MessageEvent<string>).data)
		})

		eventSource.addEventListener('version', (event) => {
			handleVersionEvent((event as MessageEvent<string>).data)
		})

		eventSource.onopen = () => {
			connected.value = true
		}

		eventSource.onerror = () => {
			// EventSource reconnects automatically; reflect the interruption
			// in the status line.
			connected.value = false
		}
	}

	// Ctrl-R refreshes both sources immediately (the server pushes results
	// over Mercure); Ctrl-T toggles toll tag vs. pay-by-mail display, exactly
	// like the terminal UI.
	async function manualRefresh() {
		const location = snapshot.value?.location
		if (!location) {
			return
		}

		refreshing.value = true

		try {
			await fetch(`/api/refresh?location=${encodeURIComponent(location)}`, { method: 'POST' })
		} catch {
			refreshing.value = false
		}
	}

	function togglePayByMail() {
		showPayByMail.value = !showPayByMail.value
	}

	function onKeydown(event: KeyboardEvent) {
		if (!event.ctrlKey) {
			return
		}

		const key = event.key.toLowerCase()
		if (key === 'r') {
			event.preventDefault()
			void manualRefresh()
		} else if (key === 't') {
			event.preventDefault()
			togglePayByMail()
		}
	}

	let clockInterval: number | undefined

	onMounted(() => {
		window.addEventListener('keydown', onKeydown)

		clockInterval = window.setInterval(() => {
			now.value = Date.now()
		}, 1000)

		void loadState()
			.then(() => {
				if (snapshot.value?.topic) {
					subscribe(snapshot.value.topic)
				}
			})
			.catch((error) => {
				console.error('failed to load kiosk state', error)
			})
	})

	onBeforeUnmount(() => {
		window.removeEventListener('keydown', onKeydown)
		if (clockInterval !== undefined) {
			window.clearInterval(clockInterval)
		}
		eventSource?.close()
		eventSource = null
	})

	return {
		snapshot,
		showPayByMail,
		refreshing,
		updateAvailable,
		connected,
		now,
		appVersion: APP_VERSION,
		manualRefresh,
		togglePayByMail,
		refreshPage: () => window.location.reload(),
	}
}