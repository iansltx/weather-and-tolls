<?php

declare(strict_types=1);

/**
 * Front controller. FrankenPHP's php_server executes this for any path that
 * does not match a static file; static assets from the built frontend are
 * served directly.
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

$app->run();