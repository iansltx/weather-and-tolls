// Payload shapes mirroring the Go structs in webapp/kioskapi/state.go.

export interface WeatherPayload {
	location: string
	temperature: number
	humidity: number
	dailyHigh: number
	dailyLow: number
	units: string
	observedAt: string
}

export interface TollRatePayload {
	name: string
	tollTagRate: number
	payByMailRate: number
	closed: boolean
}

export interface TollsPayload {
	requestedAt: string
	rates: TollRatePayload[]
}

export interface StateSnapshot {
	kind: 'state'
	status: 'loading' | 'ready'
	version: string
	topic: string
	location: string
	refreshSeconds: number
	weather?: WeatherPayload
	weatherError?: string
	tolls?: TollsPayload
	tollError?: string
	weatherRefreshedAt?: string
	tollsRefreshedAt?: string
	nextWeatherRefreshAt?: string
	nextTollsRefreshAt?: string
}

export interface VersionEvent {
	kind: 'version'
	version: string
}