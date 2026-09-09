<?php

declare(strict_types=1);

namespace Kiosk;

use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Slim\Psr7\Response as SlimResponse;

/**
 * Slim 4 controllers for the kiosk HTTP API. The routes proxy both upstream
 * services through the kiosk_* Go extension bridge, so API keys never leave
 * the Go layer.
 */
final class KioskController
{
	/**
	 * App shell: renders the built Vue entry from the Vite manifest so the
	 * hashed asset names always match the deployment.
	 */
	public function shell(Request $request, Response $response): Response
	{
		$webRoot = KioskState::webRoot();
		$manifestPath = $webRoot . '/.vite/manifest.json';

		$entryFile = '';
		$entryCss = '';
		$manifestRaw = @file_get_contents($manifestPath);
		if ($manifestRaw !== false) {
			$manifest = json_decode($manifestRaw, true);
			$entry = $manifest['index.html'] ?? null;
			if (is_array($entry)) {
				$entryFile = (string) ($entry['file'] ?? '');
				$css = $entry['css'][0] ?? '';
				$entryCss = is_string($css) ? $css : '';
			}
		}

		$script = $entryFile !== '' ? '<script type="module" src="/' . htmlspecialchars($entryFile, ENT_QUOTES) . '"></script>' : '';
		$style = $entryCss !== '' ? '<link rel="stylesheet" href="/' . htmlspecialchars($entryCss, ENT_QUOTES) . '">' : '';

		$html = <<<HTML
		<!doctype html>
		<html lang="en">
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<title>Austin Weather and Toll Rates</title>
			{$style}
		</head>
		<body>
			<div id="app"></div>
			{$script}
		</body>
		</html>
		HTML;

		$response->getBody()->write($html);

		return $response
			->withHeader('Content-Type', 'text/html; charset=utf-8')
			->withHeader('Cache-Control', 'no-cache, must-revalidate');
	}

	public function state(Request $request, Response $response): Response
	{
		$location = $this->location($request);
		$refreshSeconds = $this->refreshSeconds($request);

		$live = KioskState::fetchLive($location);

		return $this->json($response, 200, KioskState::snapshot(
			KioskState::stateTopic($location),
			$location,
			KioskState::version(),
			$refreshSeconds,
			$live['weather'],
			$live['weatherError'],
			$live['tolls'],
			$live['tollError'],
		));
	}

	public function weather(Request $request, Response $response): Response
	{
		$location = $this->location($request);

		try {
			$coords = kiosk_resolve_coords($location);
			$weather = kiosk_fetch_weather((float) $coords['lat'], (float) $coords['lon']);
		} catch (\Throwable $e) {
			return $this->json($response, 502, ['error' => $e->getMessage()]);
		}

		return $this->json($response, 200, [
			'kind' => 'weather',
			'version' => KioskState::version(),
			'weather' => $weather,
		]);
	}

	public function tolls(Request $request, Response $response): Response
	{
		try {
			$tolls = kiosk_fetch_tolls();
		} catch (\Throwable $e) {
			return $this->json($response, 502, ['error' => $e->getMessage()]);
		}

		return $this->json($response, 200, [
			'kind' => 'tolls',
			'version' => KioskState::version(),
			'tolls' => $tolls,
		]);
	}

	/**
	 * Manual refresh (Ctrl-R): drops a marker file that the poller picks up
	 * on its next tick; the refreshed state is pushed over Mercure.
	 */
	public function refresh(Request $request, Response $response): Response
	{
		if ($request->getMethod() !== 'POST') {
			return $this->json($response, 405, ['error' => 'method not allowed']);
		}

		$location = $this->location($request);
		$workDir = sys_get_temp_dir() . '/kiosk-poller';
		if (!is_dir($workDir)) {
			@mkdir($workDir, 0777, true);
		}

		$marker = $workDir . '/refresh-' . md5($location);
		if (@file_put_contents($marker, (string) time()) === false) {
			return $this->json($response, 500, ['error' => 'could not request refresh']);
		}

		@file_put_contents(
			$workDir . '/interval-' . md5($location),
			(string) $this->refreshSeconds($request),
		);

		return $this->json($response, 202, [
			'status' => 'refreshing',
			'topic' => KioskState::stateTopic($location),
			'location' => $location,
		]);
	}

	public function version(Request $request, Response $response): Response
	{
		return $this->json($response, 200, [
			'kind' => 'version',
			'version' => KioskState::version(),
		]);
	}

	public function health(Request $request, Response $response): Response
	{
		return $this->json($response, 200, ['status' => 'ok']);
	}

	private function location(Request $request): string
	{
		$params = (array) $request->getQueryParams();
		$location = trim((string) ($params['location'] ?? ''));
		return $location !== '' ? $location : KioskState::defaultLocation();
	}

	private function refreshSeconds(Request $request): int
	{
		$params = (array) $request->getQueryParams();
		$seconds = (int) trim((string) ($params['refresh'] ?? ''));
		if ($seconds <= 0) {
			return KioskState::defaultRefreshSeconds();
		}

		return KioskState::clampRefreshSeconds($seconds);
	}

	private function json(Response $response, int $status, array $payload): Response
	{
		$response->getBody()->write(
			json_encode($payload, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE) ?: '{}',
		);

		return $response
			->withStatus($status)
			->withHeader('Content-Type', 'application/json');
	}
}