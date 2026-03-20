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

if (!function_exists('absolute_url')) {
    function absolute_url(string $path): string {
        return 'https://mgoprof.ru' . $path;
    }
}

if (!function_exists('generate_user_password')) {
    function generate_user_password(int $length = 10): string {
        $chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789';
        $max = strlen($chars) - 1;
        $password = '';

        for ($i = 0; $i < $length; $i++) {
            $password .= $chars[random_int(0, $max)];
        }

        return $password;
    }
}

if (!function_exists('send_password_reset_mail')) {
    function send_password_reset_mail(string $to, string $subject, string $html): array {
        try {
            $autoload = __DIR__ . '/../vendor/autoload.php';
            if (!file_exists($autoload)) {
                return ['ok' => false, 'error' => 'vendor/autoload.php не найден'];
            }

            require_once $autoload;

            if (!function_exists('mail_config')) {
                $mailConfigFile = __DIR__ . '/../config/mail.php';
                if (file_exists($mailConfigFile)) {
                    require_once $mailConfigFile;
                }
            }

            if (!function_exists('mail_config')) {
                return ['ok' => false, 'error' => 'Функция mail_config() не найдена'];
            }

            $cfg = mail_config();

            $mail = new PHPMailer\PHPMailer\PHPMailer(true);
            $mail->CharSet = 'UTF-8';
            $mail->isSMTP();
            $mail->Host = $cfg['host'] ?? '';
            $mail->SMTPAuth = true;
            $mail->Username = $cfg['username'] ?? '';
            $mail->Password = $cfg['password'] ?? '';
            $mail->Port = (int)($cfg['port'] ?? 587);

            $encryption = (string)($cfg['encryption'] ?? 'tls');
            if ($encryption === 'ssl') {
                $mail->SMTPSecure = PHPMailer\PHPMailer\PHPMailer::ENCRYPTION_SMTPS;
            } else {
                $mail->SMTPSecure = PHPMailer\PHPMailer\PHPMailer::ENCRYPTION_STARTTLS;
            }

            $fromEmail = $cfg['from_email'] ?? ($cfg['username'] ?? '');
            $fromName  = $cfg['from_name'] ?? 'MGOPROF';

            $mail->setFrom($fromEmail, $fromName);
            $mail->addAddress($to);
            $mail->isHTML(true);
            $mail->Subject = $subject;
            $mail->Body = $html;
            $mail->AltBody = trim(strip_tags(str_replace(
                ['<br>', '<br/>', '<br />', '</p>', '</div>'],
                ["\n", "\n", "\n", "\n", "\n"],
                $html
            )));

            $mail->send();

            return ['ok' => true];
        } catch (Throwable $e) {
            return ['ok' => false, 'error' => $e->getMessage()];
        }
    }
}

$message = '';
$error = '';
$prefillEmail = trim((string)($_GET['email'] ?? ''));

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $email = trim((string)($_POST['email'] ?? ''));
    $prefillEmail = $email;

    if ($email === '') {
        $error = 'Укажите email.';
    } else {
        $stmt = $pdo->prepare("SELECT * FROM reg_users WHERE lower(email) = lower(?) LIMIT 1");
        $stmt->execute([$email]);
        $user = $stmt->fetch();

        if (!$user) {
            $error = 'Пользователь с таким email не найден.';
            if (function_exists('write_log')) {
                write_log(
                    $pdo,
                    'password_reset_not_found',
                    $email,
                    'Password reset requested for unknown email',
                    get_client_ip()
                );
            }
        } else {
            $newPassword = generate_user_password(10);
            $passwordHash = password_hash($newPassword, PASSWORD_DEFAULT);

            $upd = $pdo->prepare("
                UPDATE reg_users
                SET password_hash = ?, updated_at = CURRENT_TIMESTAMP
                WHERE id = ?
            ");
            $upd->execute([$passwordHash, (int)$user['id']]);

            $loginUrl = absolute_url(
                base_url('/cabinet/login.php?email=' . rawurlencode((string)$user['email']))
            );

            $subject = 'Новый пароль для личного кабинета';
            $body = '
                <p>Здравствуйте!</p>
                <p>Для вашего личного кабинета сформирован новый пароль.</p>
                <p><strong>Email:</strong> ' . e((string)$user['email']) . '</p>
                <p><strong>Пароль:</strong> ' . e($newPassword) . '</p>
                <p><strong>Ссылка для входа:</strong><br><a href="' . e($loginUrl) . '">' . e($loginUrl) . '</a></p>
                <p>Если вы не запрашивали пароль, просто проигнорируйте это письмо.</p>
            ';

            $mailResult = send_password_reset_mail((string)$user['email'], $subject, $body);

            if (!empty($mailResult['ok'])) {
                $message = 'Новый пароль отправлен на вашу почту.';
                if (function_exists('write_log')) {
                    write_log(
                        $pdo,
                        'password_reset_sent',
                        (string)$user['email'],
                        'Password reset email sent',
                        get_client_ip()
                    );
                }
            } else {
                $smtpError = (string)($mailResult['error'] ?? 'unknown error');
                $error = 'Не удалось отправить письмо: ' . $smtpError;
                if (function_exists('write_log')) {
                    write_log(
                        $pdo,
                        'password_reset_error',
                        (string)$user['email'],
                        'Password reset email failed: ' . $smtpError,
                        get_client_ip()
                    );
                }
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
    <title>Восстановление пароля</title>
    <style>
        :root{
            --o:#ff7c2c;
            --g:#009b35;
            --bg:#f4f7fb;
            --card:#fff;
            --line:#d8e0ea;
            --text:#213547;
            --muted:#6b7280;
            --ok-bg:#ecfdf5;
            --ok-text:#166534;
            --err-bg:#fff1f2;
            --err-text:#9f1239;
        }
        *{box-sizing:border-box}
        body{
            margin:0;
            padding:24px;
            background:var(--bg);
            font-family:Arial,sans-serif;
            color:var(--text);
        }
        .wrap{max-width:720px;margin:0 auto}
        .card{
            background:var(--card);
            border-radius:18px;
            padding:28px;
            box-shadow:0 12px 28px rgba(0,0,0,.06);
        }
        h1{margin:0 0 14px;font-size:28px}
        p{margin:0 0 18px;color:var(--muted)}
        label{
            display:block;
            font-weight:700;
            margin:14px 0 6px;
            font-size:14px;
        }
        input{
            width:100%;
            padding:14px 16px;
            border:1px solid var(--line);
            border-radius:14px;
            font-size:16px;
        }
        button{
            width:100%;
            margin-top:20px;
            padding:16px;
            border:0;
            border-radius:14px;
            color:#fff;
            font-size:16px;
            font-weight:700;
            cursor:pointer;
            background:linear-gradient(90deg,var(--o),var(--g));
        }
        .msg{
            margin-top:14px;
            padding:12px 14px;
            border-radius:12px;
            font-size:14px;
            line-height:1.45;
        }
        .ok{background:var(--ok-bg);color:var(--ok-text)}
        .err{background:var(--err-bg);color:var(--err-text)}
        .links{
            margin-top:16px;
            display:flex;
            gap:10px;
            flex-wrap:wrap;
        }
        .link-btn{
            display:inline-block;
            padding:10px 14px;
            border-radius:10px;
            text-decoration:none;
            font-weight:700;
            font-size:14px;
            line-height:1.2;
        }
        .link-btn.back{
            background:#e2e8f0;
            color:#0f172a;
        }
        .link-btn.login{
            background:#2563eb;
            color:#fff;
        }
    </style>
</head>
<body>
<div class="wrap">
    <div class="card">
        <h1>Восстановление пароля</h1>
        <p>Укажите email, который использовался при регистрации. Мы отправим новый пароль для входа в личный кабинет.</p>

        <?php if ($message): ?>
            <div class="msg ok"><?= e($message) ?></div>
        <?php endif; ?>

        <?php if ($error): ?>
            <div class="msg err"><?= e($error) ?></div>
        <?php endif; ?>

        <form method="post" autocomplete="on">
            <label for="email">Email</label>
            <input
                type="email"
                name="email"
                id="email"
                required
                autocomplete="email"
                value="<?= e($prefillEmail) ?>"
            >
            <button type="submit">Выслать новый пароль</button>
        </form>

        <div class="links">
            <a class="link-btn login" href="<?= e(base_url('/cabinet/login.php')) ?>" target="_top">← Вернуться ко входу</a>
            <a class="link-btn back" href="<?= e(base_url('/widget/')) ?>" target="_top">К регистрации</a>
        </div>
    </div>
</div>
</body>
</html>