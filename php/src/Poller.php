<?php

declare(strict_types=1);

namespace Kiosk;

/**
 * Server-side poller: publishes weather and toll updates over Mercure while
 * clients are subscribed, using the injected KioskState wrapper (which
 * extends the Kiosk\Bridge Go extension) for both the upstream lookups and
 * the hub interactions.
 *
 * Scheduling semantics are ported from the original Go implementation (and
 * the terminal UI before it): a refresh is never attempted more than once
 * per minPostRefreshDelay, weather and tolls are debounced to a shared
 * cadence, and failures back off exponentially up to maxBackoff. Every
 * completed cycle is published, so clients always see a live countdown and
 * current "as of" times; actual data changes are thus also pushed within
 * seconds of each poll.
 */
final class Poller
{
	private const MinPostRefreshDelay = 5;
	private const SharedDelay = 10;
	private const MaxBackoff = 300;

	private const TickSeconds = 1;

	/** @var array<string, array<string, mixed>> keyed by Mercure topic */
	private array $topics = [];

	private int $versionCheckAt = 0;
	private string $version = '';
	private int $seq = 0;

	private string $webRoot;
	private int $refreshSeconds;
	private float $scanInterval;
	private int $versionInterval;
	private string $workDir;

	public function __construct(private KioskState $state)
	{
		$this->webRoot = $state->webRoot();
		$this->refreshSeconds = $state->defaultRefreshSeconds();
		$this->scanInterval = max(1.0, (float) (getenv('KIOSK_SCAN_SECONDS') ?: 5));
		$this->versionInterval = max(5, (int) (getenv('KIOSK_VERSION_SECONDS') ?: 30));
		$this->workDir = sys_get_temp_dir() . '/kiosk-poller';
		$this->version = $this->readVersion();

		if (!is_dir($this->workDir)) {
			@mkdir($this->workDir, 0777, true);
		}
	}

	public function run(): void
	{
		$this->log('kiosk poller started');

		$lastScan = 0.0;

		while (true) {
			$now = time();

			if (($now - $lastScan) >= $this->scanInterval) {
				$lastScan = $now;
				$this->scan();
				$this->checkVersion($now);
			}

			$this->pollTopics($now);

			sleep(1);
		}
	}

	/**
	 * Reconciles per-topic pollers with the hub's active subscriptions: a
	 * topic is polled only while at least one subscriber is connected.
	 */
	private function scan(): void
	{
		$active = $this->activeTopics();

		foreach (array_keys($this->topics) as $topic) {
			if (!in_array($topic, $active, true)) {
				$this->log('no subscribers; stopping poller', ['topic' => $topic]);
				unset($this->topics[$topic]);
			}
		}

		foreach ($active as $topic) {
			if (!isset($this->topics[$topic])) {
				$location = $this->locationFromTopic($topic);
				if ($location === null) {
					continue;
				}

				$this->log('subscriber connected; starting poller', ['topic' => $topic, 'location' => $location]);
				$this->topics[$topic] = $this->newTopicState($location);
			}
		}
	}

	/** @return string[] topics that currently have subscribers */
	private function activeTopics(): array
	{
		try {
			$result = $this->state->callExtension($this->state->mercureSubscriptions());
		} catch (\Throwable $e) {
			$this->log('scan mercure subscriptions failed', ['error' => $e->getMessage()]);
			return [];
		}

		$topics = [];
		foreach ($result['topics'] ?? [] as $topic) {
			if (is_string($topic) && $this->locationFromTopic($topic) !== null) {
				$topics[] = $topic;
			}
		}

		return $topics;
	}

	private function locationFromTopic(string $topic): ?string
	{
		$prefix = 'https://kiosk.local/state/';
		if (!str_starts_with($topic, $prefix)) {
			return null;
		}

		$decoded = base64_decode(strtr(substr($topic, strlen($prefix)), '-_', '+/'), true);
		if ($decoded === false) {
			return null;
		}

		return $decoded;
	}

	/** @return array<string, mixed> */
	private function newTopicState(string $location): array
	{
		$now = time();

		return [
			'location' => $location,
			'interval' => $this->topicInterval($location),
			'coords' => null,
			'weather' => null,
			'weatherError' => '',
			'tolls' => null,
			'tollError' => '',
			'weatherRefreshedAt' => null,
			'tollsRefreshedAt' => null,
			'nextWeather' => $now,
			'nextTolls' => $now,
			'weatherBackoff' => self::MinPostRefreshDelay,
			'tollsBackoff' => self::MinPostRefreshDelay,
			'lastAttempt' => 0,
			'fingerprint' => null,
			'seq' => 0,
			'manual' => false,
		];
	}

	private function pollTopics(int $now): void
	{
		foreach (array_keys($this->topics) as $topic) {
			$this->handleManualRequests($topic, $this->topics[$topic]);
			$this->tick($topic, $now);
		}
	}

	/**
	 * /api/refresh (Ctrl-R) drops a marker file per location; processing it
	 * triggers an immediate fetch and unconditional publish.
	 */
	private function handleManualRequests(string $topic, array &$state): void
	{
		clearstatcache(true, $this->refreshMarker((string) $state['location']));

		$marker = $this->refreshMarker($state['location']);
		if (!file_exists($marker)) {
			return;
		}

		$mtime = (int) filemtime($marker);
		if ($mtime <= (int) ($state['lastManualHandled'] ?? 0)) {
			return;
		}

		$state['lastManualHandled'] = $mtime;
		$state['manual'] = true;
		$state['interval'] = $this->topicInterval($state['location']);
	}

	private function tick(string $topic, int $now): void
	{
		$state = &$this->topics[$topic];

		$dueWeather = $state['manual'] || ($state['nextWeather'] > 0 && $now >= $state['nextWeather']);
		$dueTolls = $state['manual'] || ($state['nextTolls'] > 0 && $now >= $state['nextTolls']);

		if (!$dueWeather && !$dueTolls) {
			return;
		}

		// Debounce: a refresh attempt within the post-refresh window pushes
		// both due sources out to the shared cadence.
		if (!$state['manual'] && ($now - (int) $state['lastAttempt']) < self::MinPostRefreshDelay) {
			$delayedUntil = $now + self::SharedDelay;
			if ($dueWeather) {
				$state['nextWeather'] = $delayedUntil;
			}
			if ($dueTolls) {
				$state['nextTolls'] = $delayedUntil;
			}
			return;
		}

		$state['lastAttempt'] = $now;

		if ($dueWeather) {
			if ($state['coords'] === null) {
				$this->resolveCoords($topic, $now);
			}
			if ($state['coords'] !== null) {
				$this->fetchWeather($topic, $now);
			}
		}

		if ($dueTolls) {
			$this->fetchTolls($topic, $now);
		}

		$this->alignSharedCadence($topic, $now);
		$this->publish($topic, (bool) $state['manual']);

		$state['manual'] = false;
	}

	private function resolveCoords(string $topic, int $now): void
	{
		$state = &$this->topics[$topic];

		try {
			$coords = $this->state->callExtension($this->state->resolveCoords((string) $state['location']));
			$state['coords'] = ['lat' => (float) $coords['lat'], 'lon' => (float) $coords['lon']];
			$state['weatherError'] = '';
		} catch (\Throwable $e) {
			$state['weatherError'] = $e->getMessage();
			$state['nextWeather'] = $now + (int) $state['weatherBackoff'];
			$state['weatherBackoff'] = self::nextBackoff((int) $state['weatherBackoff']);
			$this->log('weather geo lookup failed', ['location' => $state['location'], 'error' => $e->getMessage()]);
		}
	}

	private function fetchWeather(string $topic, int $now): void
	{
		$state = &$this->topics[$topic];

		try {
			$data = $this->state->callExtension($this->state->fetchWeather(
				(float) $state['coords']['lat'],
				(float) $state['coords']['lon'],
			));

			$state['weather'] = $data;
			$state['weatherError'] = '';
			$state['weatherRefreshedAt'] = $now;
			$state['weatherBackoff'] = self::MinPostRefreshDelay;

			$candidateNext = $now + (int) $state['interval'];
			$state['nextWeather'] = self::soonerFuture($candidateNext, (int) $state['nextTolls'], $now);
		} catch (\Throwable $e) {
			$state['weatherError'] = $e->getMessage();
			$state['nextWeather'] = $now + (int) $state['weatherBackoff'];
			$state['weatherBackoff'] = self::nextBackoff((int) $state['weatherBackoff']);
			$this->log('weather refresh failed', ['location' => $state['location'], 'error' => $e->getMessage()]);
		}
	}

	private function fetchTolls(string $topic, int $now): void
	{
		$state = &$this->topics[$topic];

		try {
			$data = $this->state->callExtension($this->state->fetchTolls());

			$state['tolls'] = $data;
			$state['tollError'] = '';
			$state['tollsRefreshedAt'] = $now;
			$state['tollsBackoff'] = self::MinPostRefreshDelay;

			$candidateNext = $now + (int) $state['interval'];
			$state['nextTolls'] = self::soonerFuture($candidateNext, (int) $state['nextWeather'], $now);
		} catch (\Throwable $e) {
			$state['tollError'] = $e->getMessage();
			$state['nextTolls'] = $now + (int) $state['tollsBackoff'];
			$state['tollsBackoff'] = self::nextBackoff((int) $state['tollsBackoff']);
			$this->log('toll refresh failed', ['error' => $e->getMessage()]);
		}
	}

	private function alignSharedCadence(string $topic, int $now): void
	{
		$state = &$this->topics[$topic];

		$delayedUntil = $now + self::SharedDelay;
		if (self::shouldDelayForSharedCadence($now, (int) $state['nextWeather'])
			|| self::shouldDelayForSharedCadence($now, (int) $state['nextTolls'])) {
			if ($state['nextWeather'] > 0 && $state['nextWeather'] < $delayedUntil) {
				$state['nextWeather'] = $delayedUntil;
			}
			if ($state['nextTolls'] > 0 && $state['nextTolls'] < $delayedUntil) {
				$state['nextTolls'] = $delayedUntil;
			}
		}
	}

	/**
	 * Stores the current state and publishes it over Mercure after every
	 * completed refresh cycle. Identical data still produces an event so the
	 * clients' "as of" times and refresh countdowns stay live (matching the
	 * terminal UI's display behavior); the fingerprint records whether the
	 * observed values actually changed.
	 */
	private function publish(string $topic, bool $manual): void
	{
		$state = &$this->topics[$topic];

		$snapshot = $this->state->snapshot(
			$topic,
			(string) $state['location'],
			$this->version,
			(int) $state['interval'],
			$state['weather'],
			(string) $state['weatherError'],
			$state['tolls'],
			(string) $state['tollError'],
			$state['weatherRefreshedAt'] !== null ? (int) $state['weatherRefreshedAt'] : null,
			$state['tollsRefreshedAt'] !== null ? (int) $state['tollsRefreshedAt'] : null,
			$state['weather'] !== null || $state['weatherError'] !== '' ? (int) $state['nextWeather'] : null,
			$state['tolls'] !== null || $state['tollError'] !== '' ? (int) $state['nextTolls'] : null,
		);

		$fingerprint = $this->state->fingerprint($snapshot);
		$changed = $fingerprint !== $state['fingerprint'];
		$state['fingerprint'] = $fingerprint;

		$data = json_encode($snapshot, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE);
		if ($data === false) {
			$this->log('marshal state snapshot failed');
			return;
		}

		$this->seq++;
		$id = 'k-' . $this->seq;

		try {
			$result = $this->state->callExtension($this->state->mercurePublish(
				$topic,
				$data,
				'state',
				$id,
			));
			$this->log('published state update', ['topic' => $topic, 'id' => $result['id'] ?? '', 'changed' => $changed, 'manual' => $manual]);
		} catch (\Throwable $e) {
			$this->log('publish state update failed', ['topic' => $topic, 'error' => $e->getMessage()]);
		}
	}

	/**
	 * Watches the deployed frontend version and pushes a version event
	 * whenever it changes.
	 */
	private function checkVersion(int $now): void
	{
		if ($now < $this->versionCheckAt) {
			return;
		}
		$this->versionCheckAt = $now + $this->versionInterval;

		$version = $this->readVersion();
		if ($version === $this->version) {
			return;
		}

		$this->version = $version;

		$data = json_encode(['kind' => 'version', 'version' => $version], JSON_UNESCAPED_SLASHES);
		if ($data === false) {
			return;
		}

		try {
			$this->state->callExtension($this->state->mercurePublish(
				KioskState::VersionTopic,
				$data,
				'version',
				'version-' . $version,
			));
			$this->log('published frontend version update', ['version' => $version]);
		} catch (\Throwable $e) {
			$this->log('publish version event failed', ['error' => $e->getMessage()]);
		}
	}

	private function readVersion(): string
	{
		$raw = @file_get_contents($this->webRoot . '/version.txt');
		if ($raw === false) {
			return 'dev';
		}

		$version = trim($raw);
		return $version !== '' ? $version : 'dev';
	}

	/**
	 * The refresh interval is a query parameter on /api/state and
	 * /api/refresh; those routes drop it into a per-location file.
	 */
	private function topicInterval(string $location): int
	{
		$file = $this->workDir . '/interval-' . md5($location);
		$raw = @file_get_contents($file);
		if ($raw === false) {
			return $this->refreshSeconds;
		}

		$seconds = (int) trim($raw);
		if ($seconds < KioskState::MinRefreshSeconds || $seconds > KioskState::MaxRefreshSeconds) {
			return $this->refreshSeconds;
		}

		return $seconds;
	}

	private function refreshMarker(string $location): string
	{
		return $this->workDir . '/refresh-' . md5($location);
	}

	private function log(string $message, array $context = []): void
	{
		$line = 'kiosk-poller: ' . $message;
		if ($context !== []) {
			$line .= ' ' . json_encode($context, JSON_UNESCAPED_SLASHES);
		}

		error_log($line);
	}

	private static function nextBackoff(int $current): int
	{
		if ($current <= 0) {
			return self::MinPostRefreshDelay;
		}

		return min($current * 2, self::MaxBackoff);
	}

	private static function shouldDelayForSharedCadence(int $now, int $next): bool
	{
		if ($next <= 0) {
			return false;
		}

		$until = $next - $now;
		return $until > 0 && $until < self::MinPostRefreshDelay;
	}

	/** Ported from the terminal UI: only considers $b when strictly after $floor. */
	private static function soonerFuture(int $a, int $b, int $floor): int
	{
		if ($b <= 0 || $b <= $floor) {
			return $a;
		}

		return min($a, $b);
	}
}