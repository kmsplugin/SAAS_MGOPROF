<?php
require_once __DIR__ . '/../config/bootstrap.php';
require_admin_auth();

$rows = $pdo->query("SELECT
    r.created_at AS reg_datetime,
    e.title AS event_title,
    u.last_name,
    u.first_name,
    u.patronymic,
    u.organization,
    u.district,
    u.email,
    CASE WHEN u.is_union_member = 1 THEN 'Да' ELSE 'Нет' END AS union_member,
    u.union_ticket,
    u.extra_info,
    r.ip_address,
    r.geo_country,
    r.geo_region,
    r.geo_city,
    r.status
FROM reg_registrations r
INNER JOIN reg_events e ON e.id = r.event_id
INNER JOIN reg_users u ON u.id = r.user_id
ORDER BY r.created_at DESC")->fetchAll();

header('Content-Type: application/vnd.ms-excel; charset=UTF-8');
header('Content-Disposition: attachment; filename=selector_report_' . date('Y-m-d_H-i') . '.xls');
echo "\xEF\xBB\xBF";

echo '<?xml version="1.0"?>';
?>
<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet"
 xmlns:o="urn:schemas-microsoft-com:office:office"
 xmlns:x="urn:schemas-microsoft-com:office:excel"
 xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet">
    <Worksheet ss:Name="Регистрации">
        <Table>
            <Row>
                <Cell><Data ss:Type="String">Дата и время</Data></Cell>
                <Cell><Data ss:Type="String">Мероприятие</Data></Cell>
                <Cell><Data ss:Type="String">Фамилия</Data></Cell>
                <Cell><Data ss:Type="String">Имя</Data></Cell>
                <Cell><Data ss:Type="String">Отчество</Data></Cell>
                <Cell><Data ss:Type="String">Организация</Data></Cell>
                <Cell><Data ss:Type="String">Округ</Data></Cell>
                <Cell><Data ss:Type="String">Почта</Data></Cell>
                <Cell><Data ss:Type="String">Член профсоюза</Data></Cell>
                <Cell><Data ss:Type="String">Номер билета</Data></Cell>
                <Cell><Data ss:Type="String">Другое</Data></Cell>
                <Cell><Data ss:Type="String">IP</Data></Cell>
                <Cell><Data ss:Type="String">Страна</Data></Cell>
                <Cell><Data ss:Type="String">Регион</Data></Cell>
                <Cell><Data ss:Type="String">Город</Data></Cell>
                <Cell><Data ss:Type="String">Статус</Data></Cell>
            </Row>
            <?php foreach ($rows as $row): ?>
            <Row>
                <Cell><Data ss:Type="String"><?= e(date('d.m.Y H:i', strtotime($row['reg_datetime']))) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e($row['event_title']) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e($row['last_name']) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e($row['first_name']) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e((string)$row['patronymic']) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e($row['organization']) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e($row['district']) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e($row['email']) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e($row['union_member']) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e((string)$row['union_ticket']) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e((string)$row['extra_info']) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e((string)$row['ip_address']) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e((string)$row['geo_country']) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e((string)$row['geo_region']) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e((string)$row['geo_city']) ?></Data></Cell>
                <Cell><Data ss:Type="String"><?= e($row['status']) ?></Data></Cell>
            </Row>
            <?php endforeach; ?>
        </Table>
    </Worksheet>
</Workbook>
