<?php

declare(strict_types=1);

require __DIR__ . '/../vendor/autoload.php';

use Kiosk\KioskState;
use Kiosk\Poller;

(new Poller(new KioskState()))->run();