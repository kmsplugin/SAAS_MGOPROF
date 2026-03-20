<?php
require_once __DIR__ . '/../config/bootstrap.php';

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    header('Location: ' . base_url('/widget/'));
    exit;
}

if (!function_exists('absolute_url')) {
    function absolute_url(string $path): string {
        return 'https://mgoprof.ru' . $path;
    }
}

$email = normalize_email((string)filter_var(post('email'), FILTER_SANITIZE_EMAIL));
$otp = preg_replace('/\D+/', '', post('otp'));

if (!filter_var($email, FILTER_VALIDATE_EMAIL) || $otp === '') {
    json_response(['status' => 'error', 'message' => 'Введите email и код.'], 422);
}

try {
    $stmt = $pdo->prepare("
        SELECT r.id, r.user_id, r.otp_expires_at
        FROM reg_registrations r
        INNER JOIN reg_users u ON u.id = r.user_id
        WHERE u.email = ? AND r.otp_code = ? AND r.status = 'pending'
        ORDER BY r.created_at DESC
        LIMIT 1
    ");
    $stmt->execute([$email, $otp]);
    $row = $stmt->fetch();

    if (!$row) {
        json_response(['status' => 'error', 'message' => 'Неверный код или email.'], 404);
    }

    if (strtotime((string)$row['otp_expires_at']) < time()) {
        json_response(['status' => 'error', 'message' => 'Код истёк. Запросите новый код повторной регистрацией.'], 410);
    }

    $update = $pdo->prepare("
        UPDATE reg_registrations
        SET status = 'verified',
            otp_verified_at = CURRENT_TIMESTAMP,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    ");
    $update->execute([$row['id']]);

    issue_user_session((int)$row['user_id'], $email);

    write_log($pdo, 'otp_verified', $email, 'OTP verified successfully');

    json_response([
        'status' => 'success',
        'redirect' => absolute_url(base_url('/cabinet/'))
    ]);
} catch (Throwable $e) {
    write_log($pdo, 'otp_verify_error', (string)$email, $e->getMessage());
    json_response(['status' => 'error', 'message' => 'Ошибка проверки кода.'], 500);
}