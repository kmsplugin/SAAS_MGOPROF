<?php
require_once __DIR__ . '/../config/bootstrap.php';

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    header('Location: ' . base_url('/widget/'));
    exit;
}

const RESEND_COOLDOWN_SECONDS = 10;

if (!function_exists('absolute_url')) {
    function absolute_url(string $path): string {
        return 'https://mgoprof.ru' . $path;
    }
}

if (!function_exists('send_registration_mail')) {
    function send_registration_mail(
        string $to,
        string $firstName,
        string $otp,
        ?string $plainPassword,
        string $loginUrl,
        string $eventTitle
    ): array {
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

            $safeName = trim($firstName) !== '' ? htmlspecialchars($firstName, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8') : 'участник';
            $safeEvent = htmlspecialchars($eventTitle, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
            $safeOtp = htmlspecialchars($otp, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
            $safeLoginUrl = htmlspecialchars($loginUrl, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');

            $passwordBlock = '';
            if ($plainPassword !== null && $plainPassword !== '') {
                $safePassword = htmlspecialchars($plainPassword, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
                $passwordBlock = '
                    <p><strong>Пароль для личного кабинета:</strong> ' . $safePassword . '</p>
                    <p>Ссылка для входа: <a href="' . $safeLoginUrl . '">' . $safeLoginUrl . '</a></p>
                ';
            }

            $subject = 'Подтверждение регистрации';
            $html = '
                <p>Здравствуйте, ' . $safeName . '!</p>
                <p>Вы начали регистрацию на мероприятие:</p>
                <p><strong>' . $safeEvent . '</strong></p>
                <p><strong>Код подтверждения (OTP):</strong> ' . $safeOtp . '</p>
                ' . $passwordBlock . '
                <p>Если вы не запрашивали регистрацию, просто проигнорируйте это письмо.</p>
            ';

            $alt = "Здравствуйте, {$firstName}!\n"
                 . "Вы начали регистрацию на мероприятие: {$eventTitle}\n"
                 . "Код подтверждения (OTP): {$otp}\n";
            if ($plainPassword !== null && $plainPassword !== '') {
                $alt .= "Пароль для личного кабинета: {$plainPassword}\n";
                $alt .= "Ссылка для входа: {$loginUrl}\n";
            }
            $alt .= "Если вы не запрашивали регистрацию, просто проигнорируйте это письмо.\n";

            $mail->setFrom($fromEmail, $fromName);
            $mail->addAddress($to);
            $mail->isHTML(true);
            $mail->Subject = $subject;
            $mail->Body = $html;
            $mail->AltBody = $alt;

            $mail->send();

            return ['ok' => true, 'driver' => 'phpmailer'];
        } catch (Throwable $e) {
            return ['ok' => false, 'error' => $e->getMessage()];
        }
    }
}

$email = normalize_email((string)filter_var(post('email'), FILTER_SANITIZE_EMAIL));
$eventId = (int) post('event_id');
$firstName = post('first_name');
$lastName = post('last_name');
$patronymic = post('patronymic');
$organization = post('organization');
$district = post('district');
$isUnionMember = post('is_union_member', '0') === '1' ? 1 : 0;
$unionTicket = post('union_ticket');
$extraInfo = post('extra_info');

if (!filter_var($email, FILTER_VALIDATE_EMAIL) || !$eventId || $firstName === '' || $lastName === '' || $organization === '' || $district === '') {
    json_response(['status' => 'error', 'message' => 'Заполните обязательные поля.'], 422);
}

$eventStmt = $pdo->prepare("SELECT id, title, is_active FROM reg_events WHERE id = ? LIMIT 1");
$eventStmt->execute([$eventId]);
$event = $eventStmt->fetch();
if (!$event || (int)$event['is_active'] !== 1) {
    json_response(['status' => 'error', 'message' => 'Мероприятие недоступно для регистрации.'], 404);
}

$ip = current_ip();
$geo = get_geo_by_ip($ip);
$userAgent = current_user_agent();
$otp = str_pad((string)random_int(0, 999999), 6, '0', STR_PAD_LEFT);
$otpExpiresAt = date('Y-m-d H:i:s', strtotime('+' . (int)app_config('otp_ttl_minutes', 10) . ' minutes'));
$userPassword = null;
$needSendOtp = true;
$message = 'Код отправлен на почту.';
$passwordIssued = false;

$cabinetLoginUrl = absolute_url(base_url('/cabinet/login.php?email=' . rawurlencode($email)));

try {
    $pdo->beginTransaction();

    $userStmt = $pdo->prepare("SELECT * FROM reg_users WHERE email = ? LIMIT 1");
    $userStmt->execute([$email]);
    $user = $userStmt->fetch();

    if ($user) {
        $userId = (int)$user['id'];

        $updateUser = $pdo->prepare("UPDATE reg_users
            SET last_name = ?, first_name = ?, patronymic = ?, organization = ?, district = ?,
                is_union_member = ?, union_ticket = ?, extra_info = ?, last_ip = ?,
                geo_country = ?, geo_region = ?, geo_city = ?, user_agent = ?, updated_at = CURRENT_TIMESTAMP
            WHERE id = ?");
        $updateUser->execute([
            $lastName, $firstName, $patronymic, $organization, $district,
            $isUnionMember, $unionTicket, $extraInfo, $ip,
            $geo['country'], $geo['region'], $geo['city'], $userAgent, $userId
        ]);

        if (empty($user['password_hash'])) {
            $userPassword = generate_user_password();
            $passwordIssued = true;

            $setPassword = $pdo->prepare("
                UPDATE reg_users
                SET password_hash = ?, password_updated_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
                WHERE id = ?
            ");
            $setPassword->execute([password_hash($userPassword, PASSWORD_DEFAULT), $userId]);
        }
    } else {
        $userPassword = generate_user_password();
        $passwordIssued = true;

        $insertUser = $pdo->prepare("INSERT INTO reg_users
            (email, last_name, first_name, patronymic, organization, district, is_union_member, union_ticket, extra_info,
             last_ip, geo_country, geo_region, geo_city, user_agent, password_hash, password_updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)");
        $insertUser->execute([
            $email, $lastName, $firstName, $patronymic, $organization, $district,
            $isUnionMember, $unionTicket, $extraInfo, $ip,
            $geo['country'], $geo['region'], $geo['city'], $userAgent,
            password_hash($userPassword, PASSWORD_DEFAULT)
        ]);
        $userId = (int)$pdo->lastInsertId();
    }

    $regStmt = $pdo->prepare("SELECT * FROM reg_registrations WHERE event_id = ? AND user_id = ? LIMIT 1");
    $regStmt->execute([$eventId, $userId]);
    $existingReg = $regStmt->fetch();

    if ($existingReg) {
        if (($existingReg['status'] ?? '') === 'verified') {
            $pdo->commit();

            write_log($pdo, 'registration_duplicate_verified', $email, 'Verified registration already exists for event #' . $eventId);

            json_response([
                'status' => 'already_registered',
                'message' => 'Вы уже зарегистрированы на это мероприятие. Войдите в кабинет по email и паролю.',
                'redirect' => $cabinetLoginUrl
            ]);
        }

        $lastActivity = strtotime((string)($existingReg['updated_at'] ?: $existingReg['created_at']));
        $secondsSince = $lastActivity ? (time() - $lastActivity) : 999999;

        if ($secondsSince < RESEND_COOLDOWN_SECONDS) {
            $needSendOtp = false;
            $message = 'Регистрация уже начата. Проверьте письмо и введите код подтверждения.';
        } else {
            $updateReg = $pdo->prepare("UPDATE reg_registrations
                SET otp_code = ?, otp_expires_at = ?, otp_verified_at = NULL, status = 'pending',
                    ip_address = ?, geo_country = ?, geo_region = ?, geo_city = ?, updated_at = CURRENT_TIMESTAMP
                WHERE id = ?");
            $updateReg->execute([
                $otp, $otpExpiresAt, $ip, $geo['country'], $geo['region'], $geo['city'], $existingReg['id']
            ]);
            $message = 'Новый код отправлен на почту.';
        }
    } else {
        $insertReg = $pdo->prepare("INSERT INTO reg_registrations
            (event_id, user_id, otp_code, otp_expires_at, status, ip_address, geo_country, geo_region, geo_city)
            VALUES (?, ?, ?, ?, 'pending', ?, ?, ?, ?)");
        $insertReg->execute([
            $eventId, $userId, $otp, $otpExpiresAt, $ip, $geo['country'], $geo['region'], $geo['city']
        ]);
    }

    $pdo->commit();

    if ($needSendOtp) {
        $mailResult = send_registration_mail(
            $email,
            $firstName,
            $otp,
            $userPassword,
            $cabinetLoginUrl,
            (string)$event['title']
        );

        if (!$mailResult['ok']) {
            write_log($pdo, 'registration_mail_error', $email, 'Registration saved, but email send failed: ' . ($mailResult['error'] ?? 'unknown'));
            json_response([
                'status' => 'saved_no_mail',
                'message' => 'Регистрация сохранена, но письмо с кодом не отправлено. Попробуйте повторить позже.'
            ], 500);
        }

        write_log($pdo, 'registration_created', $email, 'OTP sent for event #' . $eventId . ' via ' . ($mailResult['driver'] ?? 'unknown'));
    } else {
        write_log($pdo, 'registration_pending_reused', $email, 'Pending registration reused for event #' . $eventId);
    }

    $responseMessage = $message;
    if ($passwordIssued) {
        $responseMessage .= ' Пароль для кабинета также отправлен на почту.';
    }

    json_response([
        'status' => $needSendOtp ? 'success' : 'pending',
        'message' => $responseMessage,
        'show_otp' => true,
        'cabinet_login' => $cabinetLoginUrl,
        'password_issued' => $passwordIssued
    ]);
} catch (Throwable $e) {
    if ($pdo->inTransaction()) {
        $pdo->rollBack();
    }

    write_log($pdo, 'registration_error', $email, $e->getMessage());

    json_response([
        'status' => 'error',
        'message' => 'Ошибка регистрации. Попробуйте ещё раз.'
    ], 500);
}