<?php
require_once __DIR__ . '/config/bootstrap.php';

if (!empty($_SESSION['user_id'])) {
    header('Location: ' . base_url('/cabinet/'));
    exit;
}

header('Location: ' . base_url('/widget/'));
exit;
