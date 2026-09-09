<script setup lang="ts">
import { computed } from 'vue'
import { useKiosk } from './lib/useKiosk'
import WeatherCard from './components/WeatherCard.vue'
import TollRatesCard from './components/TollRatesCard.vue'
import StatusLine from './components/StatusLine.vue'
import VersionToast from './components/VersionToast.vue'

const {
	snapshot,
	showPayByMail,
	refreshing,
	updateAvailable,
	connected,
	now,
	appVersion,
	refreshPage,
} = useKiosk()

const helpLine = computed(
	() => `${appVersion} \u2022 Ctrl-R refresh now \u2022 Ctrl-T toggle toll tag/pay-by-mail`,
)
</script>

<template>
	<main class="kiosk">
		<WeatherCard
			:weather="snapshot?.weather"
			:refreshed-at="snapshot?.weatherRefreshedAt"
			:error="snapshot?.weatherError"
			:location="snapshot?.location ?? '...'"
		/>

		<TollRatesCard
			:tolls="snapshot?.tolls"
			:refreshed-at="snapshot?.tollsRefreshedAt"
			:error="snapshot?.tollError"
			:show-pay-by-mail="showPayByMail"
		/>

		<StatusLine
			:snapshot="snapshot"
			:refreshing="refreshing"
			:connected="connected"
			:now="now"
		/>

		<footer class="help">{{ helpLine }}</footer>

		<VersionToast :update-available="updateAvailable" @refresh="refreshPage" />
	</main>
</template>