<?php
function write_log(?PDO $pdo, string $type, string $email, string $message): void {
    if (!$pdo) {
        return;
    }
    try {
        $stmt = $pdo->prepare("INSERT INTO reg_logs (event_type, user_email, ip_address, message, user_agent) VALUES (?, ?, ?, ?, ?)");
        $stmt->execute([
            $type,
            $email,
            $_SERVER['REMOTE_ADDR'] ?? '0.0.0.0',
            $message,
            $_SERVER['HTTP_USER_AGENT'] ?? 'Unknown',
        ]);
    } catch (Throwable $e) {
        error_log('Logger error: ' . $e->getMessage());
    }
}
