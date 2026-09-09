# kiosk Extension

Auto-generated PHP extension from Go code.

## Classes

### Kiosk\Bridge

```php
namespace Kiosk;

class Bridge
{
    public function __construct() {}
    public function resolveCoords(string $location): array {}
    public function fetchWeather(float $lat, float $lon): array {}
    public function fetchTolls(): array {}
    public function mercureSubscriptions(): array {}
    public function mercurePublish(string $topic, string $data, string $updateType, string $id): array {}
}
```

Functions are exposed as instance methods: construct the class with
`new Kiosk\Bridge()` (the generated `__construct` binds the PHP object to its
Go-side counterpart) and call the methods on the instance.

#### Kiosk\Bridge::resolveCoords()

```php
Kiosk\Bridge::resolveCoords(string $location): array
```

**Parameters:**

- `location` (string)

**Returns:** array

#### Kiosk\Bridge::fetchWeather()

```php
Kiosk\Bridge::fetchWeather(float $lat, float $lon): array
```

**Parameters:**

- `lat` (float)
- `lon` (float)

**Returns:** array

#### Kiosk\Bridge::fetchTolls()

```php
Kiosk\Bridge::fetchTolls(): array
```

**Returns:** array

#### Kiosk\Bridge::mercureSubscriptions()

```php
Kiosk\Bridge::mercureSubscriptions(): array
```

**Returns:** array

#### Kiosk\Bridge::mercurePublish()

```php
Kiosk\Bridge::mercurePublish(string $topic, string $data, string $updateType, string $id): array
```

**Parameters:**

- `topic` (string)
- `data` (string)
- `updateType` (string)
- `id` (string)

**Returns:** array