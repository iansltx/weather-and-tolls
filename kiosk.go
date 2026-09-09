// Package kiosk links the kiosk PHP extension into custom FrankenPHP
// builds. The extension itself lives in the ext package; importing it here
// makes xcaddy's blank import of this module root register the extension.
package kiosk

import _ "ian.im/weather-and-tolls/ext"
