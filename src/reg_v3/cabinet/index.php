<?php
require_once __DIR__ . '/../config/bootstrap.php';

if (isset($_GET['logout'])) {
    clear_user_session();
    session_regenerate_id(true);
    redirect_to('/cabinet/login.php');
}

if (empty($_SESSION['user_id'])) {
    redirect_to('/cabinet/login.php');
}

$userId = (int)$_SESSION['user_id'];

$stmtUser = $pdo->prepare("SELECT last_name, first_name, patronymic, email, organization, district FROM reg_users WHERE id = ? LIMIT 1");
$stmtUser->execute([$userId]);
$user = $stmtUser->fetch();

if (!$user) {
    if (function_exists('clear_user_session')) {
        clear_user_session();
    } else {
        unset($_SESSION['user_id'], $_SESSION['user_email'], $_SESSION['cabinet_user_id'], $_SESSION['cabinet_user_email']);
    }
    session_regenerate_id(true);
    redirect_to('/cabinet/login.php');
}

$stmtEvents = $pdo->prepare("SELECT e.title, e.description, e.event_date, e.event_time, e.cabinet_link, r.created_at
    FROM reg_events e
    INNER JOIN reg_registrations r ON r.event_id = e.id
    WHERE r.user_id = ? AND r.status = 'verified'
    ORDER BY e.event_date ASC, e.event_time ASC");
$stmtEvents->execute([$userId]);
$events = $stmtEvents->fetchAll();
?>
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Личный кабинет</title>
    <style>
        :root{--o:#ff7c2c;--g:#009b35;--bg:#f4f7fb;--card:#fff;--text:#1f2937;--muted:#6b7280}
        body{font-family:Arial,sans-serif;margin:0;background:var(--bg);padding:24px;color:var(--text)}
        .wrap{max-width:900px;margin:0 auto}.card{background:var(--card);border-radius:18px;padding:26px;box-shadow:0 12px 28px rgba(0,0,0,.06);margin-bottom:18px}
        .event{border-left:6px solid var(--g);padding:18px;border-radius:12px;background:#f8fff9;margin-top:14px}.title{font-size:20px;font-weight:700;margin-bottom:6px}
        .meta{color:var(--muted);font-size:14px;margin-bottom:10px}.btn{display:inline-block;padding:12px 18px;border-radius:12px;background:linear-gradient(135deg,var(--o),var(--g));color:#fff;text-decoration:none;font-weight:700}
        .logout{color:#dc2626;text-decoration:none}
    </style>
</head>
<body>
<div class="wrap">
    <div class="card">
        <h1>Личный кабинет</h1>
        <p><strong><?= e(trim(($user['last_name'] ?? '') . ' ' . ($user['first_name'] ?? '') . ' ' . ($user['patronymic'] ?? ''))) ?></strong></p>
        <p><?= e($user['organization'] ?? '') ?> · <?= e($user['district'] ?? '') ?></p>
        <p><?= e($user['email'] ?? '') ?></p>
        <p><a class="logout" href="?logout=1">Выйти</a></p>
    </div>

    <div class="card">
        <h2>Мои мероприятия</h2>
        <?php if (!$events): ?>
            <p>Подтверждённых регистраций пока нет.</p>
        <?php else: ?>
            <?php foreach ($events as $event): ?>
                <div class="event">
                    <div class="title"><?= e($event['title']) ?></div>
                    <div class="meta">
                        Дата: <?= e(date('d.m.Y', strtotime($event['event_date']))) ?> · Время: <?= e(substr($event['event_time'], 0, 5)) ?>
                    </div>
                    <?php if (!empty($event['description'])): ?>
                        <p><?= nl2br(e($event['description'])) ?></p>
                    <?php endif; ?>
                    <?php if (!empty($event['cabinet_link'])): ?>
                        <a class="btn" href="<?= e($event['cabinet_link']) ?>" target="_blank" rel="noopener noreferrer">Перейти к трансляции</a>
                    <?php endif; ?>
                </div>
            <?php endforeach; ?>
        <?php endif; ?>
    </div>
</div>
</body>
</html>
