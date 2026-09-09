<?php

declare(strict_types=1);

/**
 * Front controller, executed as a FrankenPHP worker: the Slim app below is
 * booted once and kept resident, and frankenphp_handle_request() dispatches
 * each HTTP request to it. php_server still serves static assets from disk
 * directly and rewrites every other path to this script, which the worker
 * pool handles.
 */

require __DIR__ . '/../php/vendor/autoload.php';

use Kiosk\KioskController;
use Slim\Factory\AppFactory;

$app = AppFactory::create();
$controller = new KioskController();

$app->get('/api/state', [$controller, 'state']);
$app->get('/api/weather', [$controller, 'weather']);
$app->get('/api/tolls', [$controller, 'tolls']);
$app->post('/api/refresh', [$controller, 'refresh']);
$app->get('/api/version', [$controller, 'version']);
$app->get('/api/health', [$controller, 'health']);

// Everything that is not an API route or a static asset renders the app
// shell, so deep links keep working. The optional path matches "/" too.
$app->get('/[{path:.*}]', [$controller, 'shell']);

$handler = static function () use ($app): void {
	try {
		$app->run();
	} catch (\Throwable $e) {
		// The default exception handler only runs when the worker script
		// ends, so recover here: anything that slips past Slim's own error
		// handling becomes a 500 and the worker stays alive.
		if (!headers_sent()) {
			header('HTTP/1.1 500 Internal Server Error', true, 500);
			header('Content-Type: application/json');
		}

		echo json_encode(['error' => $e->getMessage()], JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE) ?: '{}';
	}
};

// Restart the worker after MAX_REQUESTS requests to bound memory growth
// (0 = unlimited), matching the reference worker script in the docs.
$maxRequests = (int) ($_SERVER['MAX_REQUESTS'] ?? 0);

for ($nbRequests = 0; !$maxRequests || $nbRequests < $maxRequests; ++$nbRequests) {
	$keepRunning = \frankenphp_handle_request($handler);

	// Collect garbage between requests so it cannot fire mid-response.
	\gc_collect_cycles();

	if (!$keepRunning) {
		break;
	}
}