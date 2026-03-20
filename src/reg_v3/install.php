<?php
require_once __DIR__ . '/config/bootstrap.php';
ensure_schema($pdo);
?>
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <title>Установка reg_v3</title>
    <style>
        body{font-family:Arial,sans-serif;background:#f5f7fb;padding:40px;color:#222}
        .card{max-width:780px;margin:0 auto;background:#fff;padding:30px;border-radius:16px;box-shadow:0 8px 30px rgba(0,0,0,.06)}
        code{background:#f2f4f7;padding:2px 6px;border-radius:6px}
        a{color:#0b66ff;text-decoration:none}
    </style>
</head>
<body>
<div class="card">
    <h1>База данных подготовлена</h1>
    <p>SQLite-файл создан или обновлён: <code>reg_v3/config/database.sqlite</code></p>
    <p>Теперь можно открыть <a href="<?= htmlspecialchars(base_url('/admin/login.php')) ?>">вход администратора</a>, пройти логин + OTP, создать мероприятие и проверить публичную форму.</p>
    <p><strong>Дальше:</strong> заполните <code>config/mail.php</code>. Вход администратора разрешён только для <code>kms-oleg@mail.ru</code>; после пароля система отправит OTP через <code>info@mgoprof.ru</code>.</p>
</div>
</body>
</html>
