<?php
require_once __DIR__ . '/../config/bootstrap.php';
require_admin_auth();

$stats = $pdo->query("SELECT
    (SELECT COUNT(*) FROM reg_users) AS total_users,
    (SELECT COUNT(*) FROM reg_registrations WHERE status = 'verified') AS verified_regs,
    (SELECT COUNT(*) FROM reg_registrations WHERE status = 'pending') AS pending_regs,
    (SELECT COUNT(*) FROM reg_events WHERE is_active = 1) AS active_events
")->fetch();

$events = $pdo->query("SELECT e.*, (SELECT COUNT(*) FROM reg_registrations r WHERE r.event_id = e.id) AS total_regs,
    (SELECT COUNT(*) FROM reg_registrations r WHERE r.event_id = e.id AND r.status='verified') AS verified_count
    FROM reg_events e ORDER BY e.event_date DESC, e.event_time DESC")->fetchAll();

$recent = $pdo->query("SELECT r.created_at, r.status, e.title, u.last_name, u.first_name, u.email
    FROM reg_registrations r
    INNER JOIN reg_events e ON e.id = r.event_id
    INNER JOIN reg_users u ON u.id = r.user_id
    ORDER BY r.created_at DESC LIMIT 20")->fetchAll();
?>
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <title>Админка / reg_v3</title>
    <style>
        body{font-family:Arial,sans-serif;background:#f4f7fb;margin:0;padding:24px;color:#213547}
        .wrap{max-width:1180px;margin:0 auto}.cards{display:grid;grid-template-columns:repeat(4,1fr);gap:16px;margin-bottom:24px}.card{background:#fff;border-radius:16px;padding:20px;box-shadow:0 10px 26px rgba(0,0,0,.05)}
        .num{font-size:28px;font-weight:700;margin-top:8px}.toolbar{display:flex;gap:12px;flex-wrap:wrap;margin-bottom:18px}.btn{display:inline-block;background:#007bff;color:#fff;text-decoration:none;padding:12px 16px;border-radius:12px;font-weight:700}
        .btn.secondary{background:#0ea5e9}.btn.gray{background:#64748b} table{width:100%;border-collapse:collapse;background:#fff;border-radius:16px;overflow:hidden;box-shadow:0 10px 26px rgba(0,0,0,.05);margin-bottom:24px}
        th,td{padding:12px 14px;border-bottom:1px solid #eef2f7;text-align:left;vertical-align:top} th{background:#0f172a;color:#fff}.pill{display:inline-block;padding:4px 10px;border-radius:999px;font-size:12px;font-weight:700}
        .ok{background:#dcfce7;color:#166534}.wait{background:#fef3c7;color:#92400e}.muted{color:#64748b}
        @media (max-width:900px){.cards{grid-template-columns:repeat(2,1fr)}} @media (max-width:620px){.cards{grid-template-columns:1fr}}
    </style>
</head>
<body>
<div class="wrap">
    <h1>Администратор / reg_v3</h1>
    <div class="cards">
        <div class="card"><div>Всего пользователей</div><div class="num"><?= (int)$stats['total_users'] ?></div></div>
        <div class="card"><div>Подтверждённых регистраций</div><div class="num"><?= (int)$stats['verified_regs'] ?></div></div>
        <div class="card"><div>Ожидают OTP</div><div class="num"><?= (int)$stats['pending_regs'] ?></div></div>
        <div class="card"><div>Активных мероприятий</div><div class="num"><?= (int)$stats['active_events'] ?></div></div>
    </div>

    <div style="display:flex;justify-content:space-between;align-items:center;gap:12px;flex-wrap:wrap;margin-bottom:18px"><div class="toolbar">
        <a class="btn" href="edit_event.php">+ Создать мероприятие</a>
        <a class="btn secondary" href="registrations.php">Все регистрации</a>
        <a class="btn gray" href="export_xls.php">Скачать XLS</a>
        <a class="btn gray" href="logs.php">Логи</a>
    </div><div><a class="btn gray" href="login.php?logout=1">Выйти</a></div></div>

    <table>
        <thead>
            <tr>
                <th>Дата</th>
                <th>Мероприятие</th>
                <th>Статус</th>
                <th>Всего</th>
                <th>Подтверждено</th>
                <th>Действие</th>
            </tr>
        </thead>
        <tbody>
            <?php if (!$events): ?>
                <tr><td colspan="6">Пока нет мероприятий.</td></tr>
            <?php else: ?>
                <?php foreach ($events as $event): ?>
                    <tr>
                        <td><?= e(date('d.m.Y', strtotime($event['event_date']))) ?> <span class="muted"><?= e(substr($event['event_time'], 0, 5)) ?></span></td>
                        <td><strong><?= e($event['title']) ?></strong></td>
                        <td><?= (int)$event['is_active'] === 1 ? '<span class="pill ok">Активно</span>' : '<span class="pill wait">Скрыто</span>' ?></td>
                        <td><?= (int)$event['total_regs'] ?></td>
                        <td><?= (int)$event['verified_count'] ?></td>
                        <td><a href="edit_event.php?id=<?= (int)$event['id'] ?>">Изменить</a></td>
                    </tr>
                <?php endforeach; ?>
            <?php endif; ?>
        </tbody>
    </table>

    <table>
        <thead>
            <tr>
                <th>Дата регистрации</th>
                <th>Мероприятие</th>
                <th>Участник</th>
                <th>Email</th>
                <th>Статус</th>
            </tr>
        </thead>
        <tbody>
            <?php if (!$recent): ?>
                <tr><td colspan="5">Регистраций пока нет.</td></tr>
            <?php else: ?>
                <?php foreach ($recent as $row): ?>
                    <tr>
                        <td><?= e(date('d.m.Y H:i', strtotime($row['created_at']))) ?></td>
                        <td><?= e($row['title']) ?></td>
                        <td><?= e(trim($row['last_name'] . ' ' . $row['first_name'])) ?></td>
                        <td><?= e($row['email']) ?></td>
                        <td><?= $row['status'] === 'verified' ? '<span class="pill ok">Подтверждено</span>' : '<span class="pill wait">Ожидает OTP</span>' ?></td>
                    </tr>
                <?php endforeach; ?>
            <?php endif; ?>
        </tbody>
    </table>
</div>
</body>
</html>
