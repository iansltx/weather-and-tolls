// Formatting helpers ported from the terminal UI's rendering in main.go so
// the web view matches the TUI's presentation.

export function formatDuration(seconds: number): string {
	if (seconds < 0) {
		seconds = 0
	}

	if (seconds === 1) {
		return '1 second'
	}
	if (seconds < 60) {
		return `${seconds} seconds`
	}

	const minutes = Math.floor(seconds / 60)
	const remainder = seconds % 60

	return `${minutes}:${String(remainder).padStart(2, '0')}`
}

export function countdownSeconds(now: number, target: string | undefined): number {
	if (!target) {
		return 0
	}

	const remaining = Math.floor((new Date(target).getTime() - now) / 1000)
	return Math.max(remaining, 0)
}

export function countdown(now: number, target: string | undefined): string {
	if (!target) {
		return '--'
	}
	return formatDuration(countdownSeconds(now, target))
}

export function formatClock(iso: string | undefined): string {
	if (!iso) {
		return ''
	}

	const parsed = new Date(iso)
	if (Number.isNaN(parsed.getTime())) {
		return ''
	}

	return parsed.toLocaleTimeString('en-US', {
		hour: 'numeric',
		minute: '2-digit',
		second: '2-digit',
	})
}

export function formatTemperature(value: number): string {
	return `${Math.round(value)}\u00B0F`
}

export function formatRate(value: number): string {
	return `$${value.toFixed(2)}`
}

// truncate mirrors the TUI's rune-based truncation for the toll rate name
// column; it is applied for narrow displays where CSS ellipsis is unavailable.
export function truncate(value: string, maxRunes: number): string {
	const runes = Array.from(value)
	if (runes.length <= maxRunes) {
		return value
	}

	if (maxRunes <= 1) {
		return '\u2026'
	}

	return runes.slice(0, maxRunes - 1).join('') + '\u2026'
}