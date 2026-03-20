<?php
require_once __DIR__ . '/../config/bootstrap.php';

if (session_status() !== PHP_SESSION_ACTIVE) {
    session_start();
}

if (!function_exists('get_client_ip')) {
    function get_client_ip(): string {
        $keys = [
            'HTTP_CF_CONNECTING_IP',
            'HTTP_X_REAL_IP',
            'HTTP_X_FORWARDED_FOR',
            'REMOTE_ADDR',
        ];

        foreach ($keys as $key) {
            if (!empty($_SERVER[$key])) {
                $value = trim((string)$_SERVER[$key]);

                if ($key === 'HTTP_X_FORWARDED_FOR') {
                    $parts = explode(',', $value);
                    return trim($parts[0]);
                }

                return $value;
            }
        }

        return '0.0.0.0';
    }
}

if (!function_exists('clear_cabinet_session')) {
    function clear_cabinet_session(): void {
        unset($_SESSION['user_id'], $_SESSION['user_email'], $_SESSION['cabinet_user_id'], $_SESSION['cabinet_user_email']);
    }
}

$error = '';
$prefillEmail = trim((string)($_GET['email'] ?? ''));

// logout
if (isset($_GET['logout'])) {
    clear_cabinet_session();
    session_regenerate_id(true);
    header('Location: ' . base_url('/cabinet/login.php'));
    exit;
}

// Проверяем уже существующую сессию ТОЛЬКО по user_id
if (!empty($_SESSION['user_id'])) {
    $checkStmt = $pdo->prepare("SELECT id FROM reg_users WHERE id = ? LIMIT 1");
    $checkStmt->execute([(int)$_SESSION['user_id']]);
    $existingUser = $checkStmt->fetch();

    if ($existingUser) {
        header('Location: ' . base_url('/cabinet/'));
        exit;
    } else {
        clear_cabinet_session();
        session_regenerate_id(true);
    }
}

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $email = trim((string)($_POST['email'] ?? ''));
    $password = (string)($_POST['password'] ?? '');
    $prefillEmail = $email;

    if ($email === '' || $password === '') {
        $error = 'Укажите email и пароль.';
    } else {
        $stmt = $pdo->prepare("SELECT * FROM reg_users WHERE lower(email) = lower(?) LIMIT 1");
        $stmt->execute([$email]);
        $user = $stmt->fetch();

        $ok = false;
        if ($user && !empty($user['password_hash'])) {
            $ok = password_verify($password, $user['password_hash']);
        }

        if ($ok) {
            session_regenerate_id(true);

            $_SESSION['user_id'] = (int)$user['id'];
            $_SESSION['user_email'] = (string)$user['email'];
            $_SESSION['cabinet_user_id'] = (int)$user['id'];
            $_SESSION['cabinet_user_email'] = (string)$user['email'];

            if (function_exists('write_log')) {
                write_log($pdo, 'cabinet_login_success', (string)$user['email'], 'User logged in to cabinet', get_client_ip());
            }

            header('Location: ' . base_url('/cabinet/'));
            exit;
        } else {
            $error = 'Неверный email или пароль. Если пароль не пришёл, запросите новый.';
            if (function_exists('write_log')) {
                write_log($pdo, 'cabinet_login_failed', $email, 'Failed cabinet login attempt', get_client_ip());
            }
        }
    }
}
?>
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Вход в личный кабинет</title>
    <style>
        :root{
            --o:#ff7c2c;
            --g:#009b35;
            --bg:#f4f7fb;
            --card:#fff;
            --line:#d8e0ea;
            --text:#213547;
            --muted:#6b7280;
            --err-bg:#fff1f2;
            --err-text:#9f1239;
        }
        *{box-sizing:border-box}
        body{margin:0;padding:24px;background:var(--bg);font-family:Arial,sans-serif;color:var(--text)}
        .wrap{max-width:720px;margin:0 auto}
        .card{background:var(--card);border-radius:18px;padding:28px;box-shadow:0 12px 28px rgba(0,0,0,.06)}
        h1{margin:0 0 14px;font-size:28px}
        p{margin:0 0 18px;color:var(--muted)}
        label{display:block;font-weight:700;margin:14px 0 6px;font-size:14px}
        input{width:100%;padding:14px 16px;border:1px solid var(--line);border-radius:14px;font-size:16px}
        button{width:100%;margin-top:20px;padding:16px;border:0;border-radius:14px;color:#fff;font-size:16px;font-weight:700;cursor:pointer;background:linear-gradient(90deg,var(--o),var(--g))}
        .error{margin-top:14px;padding:12px 14px;border-radius:12px;background:var(--err-bg);color:var(--err-text);font-size:14px;line-height:1.45}
        .links{margin-top:16px;display:flex;gap:10px;flex-wrap:wrap}
        .link-btn{display:inline-block;padding:10px 14px;border-radius:10px;text-decoration:none;font-weight:700;font-size:14px;line-height:1.2}
        .link-btn.reset{background:#f97316;color:#fff}
        .link-btn.back{background:#e2e8f0;color:#0f172a}
        @media (max-width:640px){.links{flex-direction:column}.link-btn{text-align:center}}
    </style>
</head>
<body>
<div class="wrap">
    <div class="card">
        <h1>Вход в личный кабинет</h1>
        <p>Используйте email и пароль из письма. Если пароль не пришёл или потерялся, запросите новый.</p>

        <?php if ($error): ?>
            <div class="error"><?= e($error) ?></div>
        <?php endif; ?>

        <form method="post" autocomplete="on">
            <label for="email">Email</label>
            <input type="email" name="email" id="email" required autocomplete="email" value="<?= e($prefillEmail) ?>">

            <label for="password">Пароль</label>
            <input type="password" name="password" id="password" required autocomplete="current-password">

            <button type="submit">Войти</button>
        </form>

        <div class="links">
            <a class="link-btn reset" href="<?= e(base_url('/cabinet/forgot_password.php')) ?>" target="_top">Не пришёл пароль? Выслать новый</a>
            <a class="link-btn back" href="<?= e(base_url('/widget/')) ?>" target="_top">← Вернуться к регистрации</a>
        </div>
    </div>
</div>
</body>
</html>