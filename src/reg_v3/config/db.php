<?php
$dbFile = __DIR__ . '/database.sqlite';

try {
    $pdo = new PDO('sqlite:' . $dbFile);
    $pdo->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);
    $pdo->setAttribute(PDO::ATTR_DEFAULT_FETCH_MODE, PDO::FETCH_ASSOC);
    $pdo->exec('PRAGMA foreign_keys = ON');
    $pdo->exec('PRAGMA journal_mode = WAL');
    $pdo->exec('PRAGMA synchronous = NORMAL');
    $pdo->exec('PRAGMA busy_timeout = 5000');
} catch (PDOException $e) {
    die('Ошибка подключения к SQLite: ' . $e->getMessage());
}

function sqlite_column_exists(PDO $pdo, string $table, string $column): bool {
    $stmt = $pdo->query("PRAGMA table_info(" . $table . ")");
    foreach ($stmt->fetchAll() as $row) {
        if (($row['name'] ?? '') === $column) {
            return true;
        }
    }
    return false;
}

function sqlite_add_column_if_missing(PDO $pdo, string $table, string $definition): void {
    $parts = preg_split('/\s+/', trim($definition), 2);
    $column = $parts[0] ?? '';
    if ($column === '') {
        return;
    }
    if (!sqlite_column_exists($pdo, $table, $column)) {
        $pdo->exec("ALTER TABLE {$table} ADD COLUMN {$definition}");
    }
}

function ensure_schema(PDO $pdo): void {
    $pdo->exec("CREATE TABLE IF NOT EXISTS reg_events (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        description TEXT,
        event_date TEXT NOT NULL,
        event_time TEXT NOT NULL,
        cabinet_link TEXT,
        is_active INTEGER NOT NULL DEFAULT 1,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TEXT
    )");

    $pdo->exec("CREATE TABLE IF NOT EXISTS reg_users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        email TEXT NOT NULL UNIQUE,
        last_name TEXT NOT NULL,
        first_name TEXT NOT NULL,
        patronymic TEXT,
        organization TEXT NOT NULL,
        district TEXT NOT NULL,
        is_union_member INTEGER NOT NULL DEFAULT 0,
        union_ticket TEXT,
        extra_info TEXT,
        last_ip TEXT,
        geo_country TEXT,
        geo_region TEXT,
        geo_city TEXT,
        user_agent TEXT,
        password_hash TEXT,
        password_updated_at TEXT,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TEXT
    )");

    $pdo->exec("CREATE TABLE IF NOT EXISTS reg_registrations (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER NOT NULL,
        user_id INTEGER NOT NULL,
        otp_code TEXT NOT NULL,
        otp_expires_at TEXT NOT NULL,
        otp_verified_at TEXT,
        status TEXT NOT NULL DEFAULT 'pending',
        ip_address TEXT,
        geo_country TEXT,
        geo_region TEXT,
        geo_city TEXT,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TEXT,
        FOREIGN KEY (event_id) REFERENCES reg_events(id) ON DELETE CASCADE,
        FOREIGN KEY (user_id) REFERENCES reg_users(id) ON DELETE CASCADE,
        UNIQUE(event_id, user_id)
    )");

    $pdo->exec("CREATE TABLE IF NOT EXISTS reg_logs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_type TEXT,
        user_email TEXT,
        ip_address TEXT,
        message TEXT,
        user_agent TEXT,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
    )");

    sqlite_add_column_if_missing($pdo, 'reg_users', 'password_hash TEXT');
    sqlite_add_column_if_missing($pdo, 'reg_users', 'password_updated_at TEXT');

    $pdo->exec("CREATE INDEX IF NOT EXISTS idx_reg_users_email ON reg_users(email)");
    $pdo->exec("CREATE INDEX IF NOT EXISTS idx_reg_registrations_created_at ON reg_registrations(created_at)");
    $pdo->exec("CREATE INDEX IF NOT EXISTS idx_reg_registrations_status ON reg_registrations(status)");
    $pdo->exec("CREATE INDEX IF NOT EXISTS idx_reg_registrations_event_user ON reg_registrations(event_id, user_id)");
}

ensure_schema($pdo);
