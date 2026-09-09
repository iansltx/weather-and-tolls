<?php

declare(strict_types=1);

namespace Kiosk;

/**
 * Shared state/payload helpers, and the app's wrapper around the extension:
 * it extends the Go extension's Kiosk\Bridge so its methods are reachable
 * right here. The JSON shapes produced here match the original Go
 * implementation byte-for-byte so the Vue frontend is unchanged.
 */
final class KioskState extends Bridge
{
	// Must match webapp/ext/kiosk.go.
	public const VersionTopic = 'https://kiosk.local/version';
	private const StateTopicPrefix = 'https://kiosk.local/state/';

	public const MinRefreshSeconds = 30;
	public const MaxRefreshSeconds = 3600;

	public static function stateTopic(string $location): string
	{
		return self::StateTopicPrefix . self::base64UrlEncode($location);
	}

	public static function base64UrlEncode(string $value): string
	{
		return rtrim(strtr(base64_encode($value), '+/', '-_'), '=');
	}

	public static function defaultLocation(): string
	{
		$location = trim(getenv('DEFAULT_WEATHER_LOCATION') ?: '');
		return $location !== '' ? $location : 'Austin, TX, US';
	}

	public static function defaultRefreshSeconds(): int
	{
		$seconds = (int) trim(getenv('KIOSK_REFRESH_SECONDS') ?: '');
		return $seconds > 0 ? $seconds : 300;
	}

	public static function clampRefreshSeconds(int $seconds): int
	{
		return max(self::MinRefreshSeconds, min(self::MaxRefreshSeconds, $seconds));
	}

	public static function webRoot(): string
	{
		$webRoot = trim(getenv('KIOSK_WEBROOT') ?: '');
		return $webRoot !== '' ? $webRoot : '/app/public';
	}

	public static function version(): string
	{
		$raw = @file_get_contents(self::webRoot() . '/version.txt');
		if ($raw === false) {
			return 'dev';
		}

		return trim($raw) ?: 'dev';
	}

	/**
	 * Calls a Kiosk\Bridge method and turns the uniform {"error": ...}
	 * payload into a proper exception.
	 *
	 * @param array<string, mixed> $result
	 * @return array<string, mixed>
	 */
	public static function callExtension(array $result): array
	{
		$error = $result['error'] ?? '';
		if (is_string($error) && $error === '') {
			unset($result['error']);
			return $result;
		}

		throw new \RuntimeException(is_string($error) ? $error : 'kiosk extension call failed');
	}

	/**
	 * Fetches live weather and toll data through the Go extension bridge.
	 *
	 * @return array{weather: ?array<string, mixed>, weatherError: string, tolls: ?array<string, mixed>, tollError: string}
	 */
	public static function fetchLive(string $location): array
	{
		$weather = null;
		$weatherError = '';
		$tolls = null;
		$tollError = '';

		$bridge = new self();

		try {
			$coords = self::callExtension($bridge->resolveCoords($location));
			$weather = self::callExtension($bridge->fetchWeather((float) $coords['lat'], (float) $coords['lon']));
		} catch (\Throwable $e) {
			$weatherError = $e->getMessage();
		}

		try {
			$tolls = self::callExtension($bridge->fetchTolls());
		} catch (\Throwable $e) {
			$tollError = $e->getMessage();
		}

		return [
			'weather' => $weather,
			'weatherError' => $weatherError,
			'tolls' => $tolls,
			'tollError' => $tollError,
		];
	}

	/**
	 * Builds the state payload shared by the /api/state route and the
	 * poller's Mercure pushes. Per-source scheduling timestamps default to a
	 * freshly computed schedule; the poller passes its own state.
	 *
	 * @param ?array<string, mixed> $weather
	 * @param ?array<string, mixed> $tolls
	 * @return array<string, mixed>
	 */
	public static function snapshot(
		string $topic,
		string $location,
		string $version,
		int $refreshSeconds,
		?array $weather,
		string $weatherError,
		?array $tolls,
		string $tollError,
		?int $weatherRefreshedAt = null,
		?int $tollsRefreshedAt = null,
		?int $nextWeatherRefreshAt = null,
		?int $nextTollsRefreshAt = null,
		?int $now = null,
	): array {
		$now = $now ?? time();
		$weatherRefreshedAt = $weatherRefreshedAt ?? $now;
		$tollsRefreshedAt = $tollsRefreshedAt ?? $now;

		$nextWeatherRefreshAt = $nextWeatherRefreshAt
			?? ($weatherError !== '' ? $now + 5 : $now + $refreshSeconds);
		$nextTollsRefreshAt = $nextTollsRefreshAt
			?? ($tollError !== '' ? $now + 5 : $now + $refreshSeconds);

		$snapshot = [
			'kind' => 'state',
			'status' => 'ready',
			'version' => $version,
			'topic' => $topic,
			'location' => $location,
			'refreshSeconds' => $refreshSeconds,
		];

		if ($weather !== null) {
			$snapshot['weather'] = $weather;
			$snapshot['weatherRefreshedAt'] = self::iso($weatherRefreshedAt);
			$snapshot['nextWeatherRefreshAt'] = self::iso($nextWeatherRefreshAt);
		}
		if ($weatherError !== '') {
			$snapshot['weatherError'] = $weatherError;
		}

		if ($tolls !== null) {
			$snapshot['tolls'] = $tolls;
			$snapshot['tollsRefreshedAt'] = self::iso($tollsRefreshedAt);
			$snapshot['nextTollsRefreshAt'] = self::iso($nextTollsRefreshAt);
		}
		if ($tollError !== '') {
			$snapshot['tollError'] = $tollError;
		}

		return $snapshot;
	}

	/**
	 * Reduces a snapshot to the observed values, prices, and errors, so callers
	 * can tell whether the data actually changed between poll cycles (the
	 * fetch/scheduling timestamps are excluded on purpose).
	 *
	 * @param array<string, mixed> $snapshot
	 */
	public static function fingerprint(array $snapshot): string
	{
		// The bridge's map values have nondeterministic key order, so the
		// fingerprint picks explicit fields rather than re-encoding arrays.
		$weather = $snapshot['weather'] ?? null;
		if (is_array($weather)) {
			$weather = [
				$weather['location'] ?? '',
				$weather['temperature'] ?? 0,
				$weather['humidity'] ?? 0,
				$weather['dailyHigh'] ?? 0,
				$weather['dailyLow'] ?? 0,
				$weather['units'] ?? '',
				$weather['observedAt'] ?? '',
			];
		}

		$tolls = $snapshot['tolls'] ?? null;
		if (is_array($tolls)) {
			// requestedAt is our own fetch timestamp, not an observed value.
			$tolls = array_map(
				static fn (array $rate): array => [
					$rate['name'] ?? '',
					$rate['tollTagRate'] ?? 0,
					$rate['payByMailRate'] ?? 0,
					$rate['closed'] ?? false,
				],
				$tolls['rates'] ?? [],
			);
		}

		return json_encode([
			'weather' => $weather,
			'weatherError' => $snapshot['weatherError'] ?? '',
			'tolls' => $tolls,
			'tollError' => $snapshot['tollError'] ?? '',
		]) ?: '';
	}

	private static function iso(int $unixSeconds): string
	{
		return gmdate('Y-m-d\TH:i:s\Z', $unixSeconds);
	}
}