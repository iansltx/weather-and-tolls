<script setup lang="ts">
import { computed } from 'vue'
import type { StateSnapshot } from '../types'
import { countdown } from '../lib/format'

// Mirrors the terminal UI's nextRefreshLine and statusView logic: a unified
// countdown when both sources agree, split countdowns when they differ, and
// per-source retry lines after failures.
const props = defineProps<{
	snapshot: StateSnapshot | null
	refreshing: boolean
	connected: boolean
	now: number
}>()

const weatherError = computed(() => props.snapshot?.weatherError ?? '')
const tollError = computed(() => props.snapshot?.tollError ?? '')

const weatherCountdown = computed(() => countdown(props.now, props.snapshot?.nextWeatherRefreshAt))
const tollsCountdown = computed(() => countdown(props.now, props.snapshot?.nextTollsRefreshAt))

const nextRefreshLine = computed(() => {
	if (props.refreshing || (weatherError.value && tollError.value)) {
		return ''
	}

	if (!weatherError.value && !tollError.value) {
		if (weatherCountdown.value === tollsCountdown.value) {
			return `Next refresh in ${weatherCountdown.value}`
		}
		return `Next refresh in ${weatherCountdown.value} (weather), ${tollsCountdown.value} (tolls)`
	}

	if (!weatherError.value) {
		return `Next refresh in ${weatherCountdown.value}`
	}
	return `Next refresh in ${tollsCountdown.value}`
})

const lines = computed(() => {
	const result: { text: string; error: boolean }[] = []

	if (weatherError.value) {
		result.push({
			text: `Weather: Failed to refresh; retrying in ${weatherCountdown.value}`,
			error: true,
		})
	}
	if (tollError.value) {
		result.push({
			text: `Tolls: Failed to refresh; retrying in ${tollsCountdown.value}`,
			error: true,
		})
	}
	if (!props.connected) {
		result.push({ text: 'Live updates: reconnecting...', error: false })
	}
	if (nextRefreshLine.value) {
		result.push({ text: nextRefreshLine.value, error: false })
	}

	return result
})
</script>

<template>
	<div class="status" aria-live="polite">
		<div v-if="refreshing" class="status-line">Refreshing...</div>
		<div v-for="line in lines" :key="line.text" class="status-line" :class="{ error: line.error }">
			{{ line.text }}
		</div>
	</div>
</template>