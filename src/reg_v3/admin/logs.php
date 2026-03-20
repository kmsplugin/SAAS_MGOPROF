<?php
require_once __DIR__ . '/../config/bootstrap.php';
require_admin_auth();
$logs = $pdo->query("SELECT * FROM reg_logs ORDER BY created_at DESC LIMIT 200")->fetchAll();
?>
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <title>Логи системы</title>
    <style>
        body{font-family:Arial,sans-serif;background:#f4f7fb;padding:24px}.wrap{max-width:1280px;margin:0 auto}.card{background:#fff;border-radius:16px;padding:20px;box-shadow:0 10px 26px rgba(0,0,0,.05)}table{width:100%;border-collapse:collapse}th,td{padding:10px 12px;border:1px solid #eef2f7;text-align:left;vertical-align:top}th{background:#0f172a;color:#fff}
    </style>
</head>
<body>
<div class="wrap">
    <div class="card">
        <h1>Логи системы</h1>
        <p><a href="index.php">← В админку</a> · <a href="login.php?logout=1">Выйти</a></p>
        <table>
            <thead>
                <tr>
                    <th>Дата</th>
                    <th>Событие</th>
                    <th>Email</th>
                    <th>IP</th>
                    <th>Сообщение</th>
                </tr>
            </thead>
            <tbody>
                <?php foreach ($logs as $log): ?>
                    <tr>
                        <td><?= e($log['created_at']) ?></td>
                        <td><?= e((string)$log['event_type']) ?></td>
                        <td><?= e((string)$log['user_email']) ?></td>
                        <td><?= e((string)$log['ip_address']) ?></td>
                        <td><?= e((string)$log['message']) ?></td>
                    </tr>
                <?php endforeach; ?>
            </tbody>
        </table>
    </div>
</div>
</body>
</html>
