<?php
$app = require __DIR__ . '/app.php';
date_default_timezone_set($app['timezone'] ?? 'Europe/Moscow');

ini_set('session.cookie_httponly', '1');
ini_set('session.use_strict_mode', '1');
if (PHP_VERSION_ID >= 70300) {
    session_set_cookie_params([
        'lifetime' => 0,
        'path' => '/',
        'secure' => (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off'),
        'httponly' => true,
        'samesite' => 'Lax',
    ]);
}
if (session_status() !== PHP_SESSION_ACTIVE) {
    session_start();
}

require_once __DIR__ . '/db.php';
require_once __DIR__ . '/logger.php';
require_once __DIR__ . '/mail.php';

function app_config(string $key, $default = null) {
    static $cfg;
    if ($cfg === null) {
        $cfg = require __DIR__ . '/app.php';
    }
    return $cfg[$key] ?? $default;
}

function base_url(string $path = ''): string {
    $base = rtrim(app_config('base_path', '/reg_v3'), '/');
    $path = '/' . ltrim($path, '/');
    return $base . ($path === '/' ? '/' : $path);
}

function redirect_to(string $path): void {
    header('Location: ' . base_url($path));
    exit;
}

function e(string $value): string {
    return htmlspecialchars($value, ENT_QUOTES, 'UTF-8');
}

function post(string $key, $default = '') {
    return isset($_POST[$key]) ? trim((string)$_POST[$key]) : $default;
}

function json_response(array $payload, int $status = 200): void {
    http_response_code($status);
    header('Content-Type: application/json; charset=utf-8');
    echo json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

function current_ip(): string {
    $keys = ['HTTP_CF_CONNECTING_IP', 'HTTP_X_FORWARDED_FOR', 'REMOTE_ADDR'];
    foreach ($keys as $key) {
        if (!empty($_SERVER[$key])) {
            $raw = explode(',', (string)$_SERVER[$key])[0];
            return trim($raw);
        }
    }
    return '0.0.0.0';
}

function current_user_agent(): string {
    return $_SERVER['HTTP_USER_AGENT'] ?? 'Unknown';
}

function get_geo_by_ip(string $ip): array {
    $geo = ['country' => '', 'region' => '', 'city' => ''];
    $autoload = dirname(__DIR__) . '/vendor/autoload.php';
    $mmdb = dirname(__DIR__) . '/vendor/maxmind/GeoLite2-City.mmdb';

    if (!file_exists($autoload) || !file_exists($mmdb)) {
        return $geo;
    }

    try {
        require_once $autoload;
        if (class_exists('MaxMind\Db\Reader')) {
            $reader = new MaxMind\Db\Reader($mmdb);
            $record = $reader->get($ip);
            if ($record) {
                $geo['country'] = $record['country']['names']['ru'] ?? $record['country']['names']['en'] ?? '';
                $geo['region']  = $record['subdivisions'][0]['names']['ru'] ?? $record['subdivisions'][0]['names']['en'] ?? '';
                $geo['city']    = $record['city']['names']['ru'] ?? $record['city']['names']['en'] ?? '';
            }
            $reader->close();
        }
    } catch (Throwable $e) {
        error_log('Geo lookup error: ' . $e->getMessage());
    }

    return $geo;
}

function normalize_email(string $email): string {
    return mb_strtolower(trim($email), 'UTF-8');
}

function generate_user_password(int $length = 10): string {
    $alphabet = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789';
    $max = strlen($alphabet) - 1;
    $password = '';
    for ($i = 0; $i < $length; $i++) {
        $password .= $alphabet[random_int(0, $max)];
    }
    return $password;
}

function issue_user_session(int $userId, string $email): void {
    $_SESSION['user_id'] = $userId;
    $_SESSION['auth_email'] = normalize_email($email);
}

function clear_user_session(): void {
    unset($_SESSION['user_id'], $_SESSION['auth_email']);
}

function admin_session_key(): string {
    return 'selector_admin_auth';
}

function admin_pending_key(): string {
    return 'selector_admin_pending';
}

function is_admin_authenticated(): bool {
    $auth = $_SESSION[admin_session_key()] ?? null;
    if (!is_array($auth)) {
        return false;
    }
    $email = normalize_email((string)($auth['email'] ?? ''));
    return $email !== '' && hash_equals(normalize_email((string)app_config('admin_email', '')), $email);
}

function require_admin_auth(): void {
    if (!is_admin_authenticated()) {
        header('Location: ' . base_url('/admin/login.php'));
        exit;
    }
}

function issue_admin_otp(PDO $pdo, string $email): array {
    $otp = str_pad((string)random_int(0, 999999), 6, '0', STR_PAD_LEFT);
    $expiresAt = time() + ((int)app_config('admin_otp_ttl_minutes', 10) * 60);

    $_SESSION[admin_pending_key()] = [
        'email' => normalize_email($email),
        'otp_hash' => password_hash($otp, PASSWORD_DEFAULT),
        'expires_at' => $expiresAt,
        'attempts' => 0,
    ];

    $result = send_otp_email($email, (string)app_config('admin_name', 'Администратор'), $otp);
    if (!($result['ok'] ?? false)) {
        unset($_SESSION[admin_pending_key()]);
        write_log($pdo, 'admin_otp_error', $email, (string)($result['error'] ?? 'OTP send failed'));
        return ['ok' => false, 'message' => 'Не удалось отправить OTP на почту администратора.'];
    }

    write_log($pdo, 'admin_otp_sent', $email, 'Admin OTP sent via ' . ($result['driver'] ?? 'unknown'));
    return ['ok' => true, 'message' => 'Код подтверждения отправлен на почту администратора.'];
}

function complete_admin_login(string $email): void {
    $_SESSION[admin_session_key()] = [
        'email' => normalize_email($email),
        'name' => (string)app_config('admin_name', 'Администратор'),
        'at' => time(),
    ];
    unset($_SESSION[admin_pending_key()]);
}

function admin_logout(): void {
    unset($_SESSION[admin_session_key()], $_SESSION[admin_pending_key()]);
}
