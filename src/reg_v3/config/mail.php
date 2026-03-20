<?php
function mail_config(): array {
    static $cfg = null;
    if ($cfg === null) {
        $cfg = [
            'host'       => 'smtp.mail.ru',
            'username'   => 'info@mgoprof.ru',
            'password'   => 'q6ecrKA7HuleYJgiFf5H',
            'port'       => 587,
            'encryption' => 'tls',
            'from_email' => 'info@mgoprof.ru',
            'from_name'  => 'МГО Профсоюза',
        ];
    }
    return $cfg;
}

function send_html_email(string $toEmail, string $subject, string $bodyHtml): array {
    $cfg = mail_config();
    $autoload = dirname(__DIR__) . '/vendor/autoload.php';
    if (file_exists($autoload)) {
        require_once $autoload;
    }

    try {
        if (class_exists('PHPMailer\PHPMailer\PHPMailer')) {
            $mail = new PHPMailer\PHPMailer\PHPMailer(true);
            $mail->isSMTP();
            $mail->Host = $cfg['host'];
            $mail->SMTPAuth = true;
            $mail->Username = $cfg['username'];
            $mail->Password = $cfg['password'];
            $mail->Port = (int)$cfg['port'];
            $mail->SMTPSecure = $cfg['encryption'];
            $mail->CharSet = 'UTF-8';
            $mail->setFrom($cfg['from_email'], $cfg['from_name']);
            $mail->addAddress($toEmail);
            $mail->isHTML(true);
            $mail->Subject = $subject;
            $mail->Body = $bodyHtml;
            $mail->send();
            return ['ok' => true, 'driver' => 'phpmailer'];
        }
    } catch (Throwable $e) {
        return ['ok' => false, 'driver' => 'phpmailer', 'error' => $e->getMessage()];
    }

    $headers = "MIME-Version: 1.0
";
    $headers .= "Content-type:text/html;charset=UTF-8
";
    $headers .= 'From: ' . $cfg['from_name'] . ' <' . $cfg['from_email'] . ">
";
    $sent = @mail($toEmail, '=?UTF-8?B?' . base64_encode($subject) . '?=', $bodyHtml, $headers);

    return ['ok' => $sent, 'driver' => 'mail'];
}

function send_otp_email(string $toEmail, string $firstName, string $otp, ?string $password = null): array {
    $loginLink = function_exists('base_url') ? base_url('/cabinet/login.php?email=' . urlencode($toEmail)) : '#';
    $bodyHtml = '<div style="font-family:Arial,sans-serif;font-size:16px">'
        . '<p>Здравствуйте' . ($firstName ? ', ' . htmlspecialchars($firstName, ENT_QUOTES, 'UTF-8') : '') . '.</p>'
        . '<p>Ваш код подтверждения: <strong style="font-size:22px">' . htmlspecialchars($otp, ENT_QUOTES, 'UTF-8') . '</strong></p>'
        . '<p>Код действует 10 минут.</p>';

    if ($password !== null && $password !== '') {
        $bodyHtml .= '<div style="margin:18px 0;padding:14px 16px;background:#f4f7fb;border-radius:12px">'
            . '<p style="margin:0 0 8px"><strong>Пароль для входа в личный кабинет:</strong></p>'
            . '<p style="margin:0;font-size:20px"><strong>' . htmlspecialchars($password, ENT_QUOTES, 'UTF-8') . '</strong></p>'
            . '</div>'
            . '<p><a href="' . htmlspecialchars($loginLink, ENT_QUOTES, 'UTF-8') . '" style="display:inline-block;padding:10px 16px;background:#0f172a;color:#fff;text-decoration:none;border-radius:10px">Войти в кабинет</a></p>';
    }

    $bodyHtml .= '<p>Если это были не вы, просто проигнорируйте письмо.</p></div>';

    return send_html_email($toEmail, 'Код подтверждения регистрации', $bodyHtml);
}
