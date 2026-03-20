<?php
require_once __DIR__ . '/../config/bootstrap.php';
require_admin_auth();

$rows = $pdo->query("SELECT
    r.created_at,
    r.status,
    e.title AS event_title,
    u.last_name,
    u.first_name,
    u.patronymic,
    u.organization,
    u.district,
    u.email,
    u.is_union_member,
    u.union_ticket,
    u.extra_info,
    r.ip_address,
    r.geo_country,
    r.geo_region,
    r.geo_city
FROM reg_registrations r
INNER JOIN reg_events e ON e.id = r.event_id
INNER JOIN reg_users u ON u.id = r.user_id
ORDER BY r.created_at DESC")->fetchAll();
?>
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <title>Все регистрации</title>
    <style>
        body{font-family:Arial,sans-serif;background:#f4f7fb;padding:24px}.wrap{max-width:1480px;margin:0 auto}.top{display:flex;justify-content:space-between;align-items:center;margin-bottom:16px}a{color:#0b66ff;text-decoration:none}
        table{width:100%;border-collapse:collapse;background:#fff;border-radius:16px;overflow:hidden;box-shadow:0 10px 26px rgba(0,0,0,.05)}th,td{padding:10px 12px;border-bottom:1px solid #eef2f7;text-align:left;vertical-align:top;font-size:14px}th{background:#0f172a;color:#fff;position:sticky;top:0}.pill{display:inline-block;padding:4px 10px;border-radius:999px;font-size:12px;font-weight:700}.ok{background:#dcfce7;color:#166534}.wait{background:#fef3c7;color:#92400e}.table-wrap{overflow:auto;max-height:78vh}
    </style>
</head>
<body>
<div class="wrap">
    <div class="top">
        <h1>Все регистрации</h1>
        <div><a href="index.php">← В админку</a> · <a href="login.php?logout=1">Выйти</a></div>
    </div>
    <div class="table-wrap">
        <table>
            <thead>
                <tr>
                    <th>Дата/время</th>
                    <th>Мероприятие</th>
                    <th>ФИО</th>
                    <th>Организация</th>
                    <th>Округ</th>
                    <th>Почта</th>
                    <th>Профсоюз</th>
                    <th>Билет</th>
                    <th>Другое</th>
                    <th>IP</th>
                    <th>Гео</th>
                    <th>Статус</th>
                </tr>
            </thead>
            <tbody>
                <?php foreach ($rows as $row): ?>
                    <tr>
                        <td><?= e(date('d.m.Y H:i', strtotime($row['created_at']))) ?></td>
                        <td><?= e($row['event_title']) ?></td>
                        <td><?= e(trim($row['last_name'] . ' ' . $row['first_name'] . ' ' . $row['patronymic'])) ?></td>
                        <td><?= e($row['organization']) ?></td>
                        <td><?= e($row['district']) ?></td>
                        <td><?= e($row['email']) ?></td>
                        <td><?= (int)$row['is_union_member'] === 1 ? 'Да' : 'Нет' ?></td>
                        <td><?= e((string)$row['union_ticket']) ?></td>
                        <td><?= e((string)$row['extra_info']) ?></td>
                        <td><?= e((string)$row['ip_address']) ?></td>
                        <td><?= e(trim(implode(', ', array_filter([$row['geo_country'], $row['geo_region'], $row['geo_city']])))) ?></td>
                        <td><?= $row['status'] === 'verified' ? '<span class="pill ok">Подтверждено</span>' : '<span class="pill wait">Ожидает OTP</span>' ?></td>
                    </tr>
                <?php endforeach; ?>
            </tbody>
        </table>
    </div>
</div>
</body>
</html>
