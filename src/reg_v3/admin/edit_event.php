<?php
require_once __DIR__ . '/../config/bootstrap.php';
require_admin_auth();

$id = isset($_GET['id']) ? (int)$_GET['id'] : 0;
$event = [
    'title' => '',
    'description' => '',
    'event_date' => date('Y-m-d'),
    'event_time' => '12:00',
    'cabinet_link' => '',
    'is_active' => 1,
];

if ($id > 0) {
    $stmt = $pdo->prepare("SELECT * FROM reg_events WHERE id = ? LIMIT 1");
    $stmt->execute([$id]);
    $dbEvent = $stmt->fetch();
    if ($dbEvent) {
        $event = $dbEvent;
    }
}

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $data = [
        post('title'),
        post('description'),
        post('event_date'),
        post('event_time'),
        post('cabinet_link'),
        isset($_POST['is_active']) ? 1 : 0,
    ];

    if ($id > 0) {
        $data[] = $id;
        $stmt = $pdo->prepare("UPDATE reg_events SET title = ?, description = ?, event_date = ?, event_time = ?, cabinet_link = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?");
        $stmt->execute($data);
        write_log($pdo, 'event_updated', '', 'Event #' . $id . ' updated');
    } else {
        $stmt = $pdo->prepare("INSERT INTO reg_events (title, description, event_date, event_time, cabinet_link, is_active) VALUES (?, ?, ?, ?, ?, ?)");
        $stmt->execute($data);
        write_log($pdo, 'event_created', '', 'Event #' . $pdo->lastInsertId() . ' created');
    }

    header('Location: index.php');
    exit;
}
?>
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <title><?= $id > 0 ? 'Редактирование' : 'Новое мероприятие' ?></title>
    <style>
        body{font-family:Arial,sans-serif;background:#f4f7fb;padding:24px}.card{max-width:760px;margin:0 auto;background:#fff;padding:26px;border-radius:18px;box-shadow:0 12px 28px rgba(0,0,0,.06)}
        label{display:block;font-weight:700;margin-bottom:6px} input,textarea{width:100%;padding:12px 14px;border:1px solid #d8e0ea;border-radius:12px;box-sizing:border-box} textarea{min-height:120px;resize:vertical}
        .row{margin-bottom:16px}.grid{display:grid;grid-template-columns:1fr 1fr;gap:16px}.btn{display:inline-block;background:#007bff;color:#fff;border:0;border-radius:12px;padding:14px 18px;font-weight:700;cursor:pointer;text-decoration:none}
    </style>
</head>
<body>
<div class="card">
    <h1><?= $id > 0 ? 'Редактирование мероприятия' : 'Новое мероприятие' ?></h1>
    <form method="post">
        <div class="row">
            <label>Наименование мероприятия</label>
            <input type="text" name="title" value="<?= e($event['title']) ?>" required>
        </div>
        <div class="row">
            <label>Описание</label>
            <textarea name="description"><?= e($event['description']) ?></textarea>
        </div>
        <div class="grid row">
            <div>
                <label>Дата</label>
                <input type="date" name="event_date" value="<?= e($event['event_date']) ?>" required>
            </div>
            <div>
                <label>Время</label>
                <input type="time" name="event_time" value="<?= e(substr($event['event_time'], 0, 5)) ?>" required>
            </div>
        </div>
        <div class="row">
            <label>Ссылка для личного кабинета / трансляции</label>
            <input type="url" name="cabinet_link" value="<?= e($event['cabinet_link']) ?>" placeholder="https://...">
        </div>
        <div class="row">
            <label><input type="checkbox" name="is_active" <?= (int)$event['is_active'] === 1 ? 'checked' : '' ?>> Активно для регистрации</label>
        </div>
        <button class="btn" type="submit">Сохранить</button>
        <a class="btn" href="index.php" style="background:#64748b">Назад</a>
        <a class="btn" href="login.php?logout=1" style="background:#dc2626">Выйти</a>
    </form>
</div>
</body>
</html>
