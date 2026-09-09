<script setup lang="ts">
import { computed } from 'vue'
import type { WeatherPayload } from '../types'
import { formatClock, formatTemperature } from '../lib/format'

const props = defineProps<{
	weather: WeatherPayload | undefined
	refreshedAt: string | undefined
	error: string | undefined
	location: string
}>()

const heading = computed(() => {
	const asOf = formatClock(props.refreshedAt)
	return asOf ? `OpenWeather as of ${asOf}` : 'OpenWeather'
})
</script>

<template>
	<section class="box" aria-label="Weather">
		<h2>{{ heading }}</h2>

		<div v-if="weather" class="rows">
			<div class="row">Location: {{ weather.location }}</div>
			<div class="row">Temperature: {{ formatTemperature(weather.temperature) }}</div>
			<div class="row">Humidity: {{ weather.humidity }}%</div>
			<div class="row">
				Expected high / low:
				{{ formatTemperature(weather.dailyHigh) }} /
				{{ formatTemperature(weather.dailyLow) }}
			</div>
		</div>
		<div v-else class="pending">Refreshing weather for {{ location }}...</div>

		<div v-if="error" class="error">Last error: {{ error }}</div>
	</section>
</template>