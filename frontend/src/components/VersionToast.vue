<script setup lang="ts">
defineProps<{
	updateAvailable: boolean
}>()

const emit = defineEmits<{
	refresh: []
}>()
</script>

<template>
	<transition name="toast">
		<div v-if="updateAvailable" class="toast" role="alert">
			New version available,
			<button class="toast-button" type="button" @click="emit('refresh')">click to refresh</button>
		</div>
	</transition>
</template>