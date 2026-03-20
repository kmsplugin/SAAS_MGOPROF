<?php
require_once __DIR__ . '/../config/bootstrap.php';

if (isset($_GET['logout'])) {
    admin_logout();
    header('Location: login.php');
    exit;
}

if (is_admin_authenticated()) {
    header('Location: index.php');
    exit;
}

$error = '';
$notice = '';
$pending = $_SESSION[admin_pending_key()] ?? null;

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $step = post('step');

    if ($step === 'credentials') {
        $email = normalize_email(post('email'));
        $password = post('password');

        $emailOk = hash_equals(normalize_email((string)app_config('admin_email', '')), $email);
        $passwordHash = (string)app_config('admin_password_hash', '');
        $passwordOk = $passwordHash !== '' && password_verify($password, $passwordHash);

        if (!$emailOk || !$passwordOk) {
            $error = 'Неверный логин или пароль.';
            write_log($pdo, 'admin_login_failed', $email, 'Invalid admin credentials');
        } else {
            $otpResult = issue_admin_otp($pdo, $email);
            if ($otpResult['ok']) {
                $notice = $otpResult['message'];
                $pending = $_SESSION[admin_pending_key()] ?? null;
            } else {
                $error = $otpResult['message'];
            }
        }
    }

    if ($step === 'otp') {
        $code = preg_replace('/\D+/', '', post('otp'));
        $pending = $_SESSION[admin_pending_key()] ?? null;

        if (!$pending || !is_array($pending)) {
            $error = 'Сессия входа истекла. Введите логин и пароль заново.';
        } elseif ($code === '' || strlen($code) !== 6) {
            $error = 'Введите 6-значный OTP-код.';
        } elseif ((int)($pending['expires_at'] ?? 0) < time()) {
            unset($_SESSION[admin_pending_key()]);
            $error = 'Срок действия OTP истёк. Войдите заново.';
        } elseif (!password_verify($code, (string)($pending['otp_hash'] ?? ''))) {
            $_SESSION[admin_pending_key()]['attempts'] = (int)($pending['attempts'] ?? 0) + 1;
            write_log($pdo, 'admin_otp_failed', (string)($pending['email'] ?? ''), 'Invalid admin OTP');
            $error = 'Неверный OTP-код.';
            $pending = $_SESSION[admin_pending_key()] ?? null;
        } else {
            $email = (string)($pending['email'] ?? '');
            complete_admin_login($email);
            write_log($pdo, 'admin_login_success', $email, 'Admin logged in successfully');
            header('Location: index.php');
            exit;
        }
    }
}
?>
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Вход администратора</title>
    <style>
        body{font-family:Arial,sans-serif;background:#f4f7fb;margin:0;padding:24px;color:#213547}
        .card{max-width:520px;margin:40px auto;background:#fff;padding:28px;border-radius:18px;box-shadow:0 12px 28px rgba(0,0,0,.06)}
        label{display:block;font-weight:700;margin:0 0 6px}.row{margin-bottom:16px}
        input{width:100%;padding:12px 14px;border:1px solid #d8e0ea;border-radius:12px;box-sizing:border-box;font-size:15px}
        button{width:100%;border:0;border-radius:12px;padding:14px 18px;background:linear-gradient(135deg,#ff7c2c,#009b35);color:#fff;font-weight:700;font-size:16px;cursor:pointer}
        .msg{margin-bottom:16px;padding:12px 14px;border-radius:12px;font-size:14px}.err{background:#fff1f2;color:#9f1239}.ok{background:#ecfdf5;color:#166534}.muted{color:#64748b;font-size:14px}
        code{background:#f2f4f7;padding:2px 6px;border-radius:6px}
    </style>
</head>
<body>
<div class="card">
    <h1>Вход администратора</h1>
    <p class="muted">Доступ только для администратора: <strong><?= e((string)app_config('admin_name')) ?></strong>.</p>

    <?php if ($error): ?><div class="msg err"><?= e($error) ?></div><?php endif; ?>
    <?php if ($notice): ?><div class="msg ok"><?= e($notice) ?></div><?php endif; ?>

    <?php if (!empty($pending)): ?>
        <p class="muted">На адрес <code><?= e((string)$pending['email']) ?></code> отправлен OTP-код с ящика <code>info@mgoprof.ru</code>.</p>
        <form method="post">
            <input type="hidden" name="step" value="otp">
            <div class="row">
                <label for="otp">OTP-код</label>
                <input id="otp" name="otp" inputmode="numeric" maxlength="6" placeholder="000000" required>
            </div>
            <button type="submit">Подтвердить вход</button>
        </form>
        <p class="muted" style="margin-top:14px"><a href="login.php?logout=1">Начать заново</a></p>
    <?php else: ?>
        <form method="post" autocomplete="off">
            <input type="hidden" name="step" value="credentials">
            <div class="row">
                <label for="email">Логин</label>
                <input id="email" type="email" name="email" value="<?= e((string)app_config('admin_email')) ?>" required>
            </div>
            <div class="row">
                <label for="password">Пароль</label>
                <input id="password" type="password" name="password" required>
            </div>
            <button type="submit">Продолжить</button>
        </form>
    <?php endif; ?>
</div>
</body>
</html>
