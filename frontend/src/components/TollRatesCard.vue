<script setup lang="ts">
import { computed } from 'vue'
import type { TollsPayload } from '../types'
import { formatClock, formatRate, truncate } from '../lib/format'

const props = defineProps<{
	tolls: TollsPayload | undefined
	refreshedAt: string | undefined
	error: string | undefined
	showPayByMail: boolean
}>()

const rateLabel = computed(() => (props.showPayByMail ? 'Pay by mail' : 'Toll tag'))

const heading = computed(() => {
	const asOf = formatClock(props.refreshedAt)
	const suffix = asOf ? ` as of ${asOf}` : ''
	return `Austin Toll Rates${suffix} (${rateLabel.value})`
})

const rows = computed(() => {
	if (!props.tolls) {
		return []
	}

	return props.tolls.rates.map((rate) => ({
		name: truncate(rate.name, 50),
		value: props.showPayByMail ? rate.payByMailRate : rate.tollTagRate,
		closed: rate.closed,
	}))
})
</script>

<template>
	<section class="box" aria-label="Austin toll rates">
		<h2>{{ heading }}</h2>

		<div v-if="rows.length > 0" class="rows toll-rows">
			<template v-for="row in rows" :key="row.name">
				<div v-if="row.closed" class="row toll-row">
					<span class="toll-name">{{ row.name }}</span>
					<span class="closed">CLOSED</span>
				</div>
				<div v-else class="row toll-row">
					<span class="toll-name">{{ row.name }}</span>
					<span class="rate">{{ formatRate(row.value) }}</span>
				</div>
			</template>
		</div>
		<div v-else class="pending">Refreshing toll rates...</div>

		<div v-if="error" class="error">Last error: {{ error }}</div>
	</section>
</template>