<?php
require_once __DIR__ . '/../config/bootstrap.php';

$districts = [
    'ЦАО','САО','СВАО','ВАО','ЮВАО','ЮАО','ЮЗАО','ЗАО','СЗАО','ЗелАО','ТиНАО','Московская область','Другой регион'
];

$stmt = $pdo->query("SELECT id, title, event_date, event_time FROM reg_events WHERE is_active = 1 ORDER BY event_date ASC, event_time ASC");
$events = $stmt->fetchAll();
?>
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Регистрация</title>
    <style>
        :root{
            --o:#ff7c2c;
            --g:#009b35;
            --bg:#f4f7fb;
            --card:#fff;
            --line:#d8e0ea;
            --text:#213547;
            --muted:#6b7280;
            --ok-bg:#ecfdf5;
            --ok-text:#166534;
            --err-bg:#fff1f2;
            --err-text:#9f1239;
            --info-bg:#eff6ff;
            --info-text:#1d4ed8;
        }
        *{box-sizing:border-box}
        body{
            margin:0;
            padding:24px;
            background:var(--bg);
            font-family:Arial,sans-serif;
            color:var(--text);
        }
        .wrap{max-width:760px;margin:0 auto}
        .card{
            background:var(--card);
            border-radius:18px;
            padding:26px;
            box-shadow:0 12px 28px rgba(0,0,0,.06);
        }
        h1,h2{margin:0 0 18px}
        .grid{display:grid;grid-template-columns:1fr 1fr;gap:14px}
        .full{grid-column:1 / -1}
        label{
            display:block;
            font-weight:700;
            font-size:14px;
            margin:0 0 6px;
        }
        input,select,textarea{
            width:100%;
            padding:12px 14px;
            border:1px solid var(--line);
            border-radius:12px;
            font-size:15px;
            background:#fff;
        }
        input:focus,select:focus,textarea:focus{
            outline:none;
            border-color:#93c5fd;
            box-shadow:0 0 0 3px rgba(59,130,246,.10);
        }
        textarea{
            min-height:96px;
            resize:vertical;
        }
        button{
            border:0;
            border-radius:12px;
            padding:14px 18px;
            background:linear-gradient(135deg,var(--o),var(--g));
            color:#fff;
            font-weight:700;
            font-size:16px;
            cursor:pointer;
        }
        button[disabled]{cursor:not-allowed;opacity:.82}
        .muted{color:var(--muted);font-size:14px}
        .hidden{display:none}
        .msg{
            margin-top:12px;
            padding:12px 14px;
            border-radius:12px;
            font-size:14px;
            line-height:1.45;
        }
        .err{background:var(--err-bg);color:var(--err-text)}
        .ok{background:var(--ok-bg);color:var(--ok-text)}
        .info{background:var(--info-bg);color:var(--info-text)}
        .otp{
            font-size:28px;
            letter-spacing:6px;
            text-align:center;
        }
        .actions{
            display:flex;
            gap:12px;
            flex-wrap:wrap;
        }
        .actions button{flex:1}
        .helper-actions{
            display:flex;
            gap:10px;
            flex-wrap:wrap;
            margin-top:12px;
        }
        .helper-btn{
            display:inline-block;
            padding:10px 14px;
            border-radius:10px;
            text-decoration:none;
            font-weight:700;
            font-size:14px;
            line-height:1.2;
        }
        .helper-btn.login{
            background:#2563eb;
            color:#fff;
        }
        .helper-btn.reset{
            background:#f97316;
            color:#fff;
        }
        .helper-btn.simple{
            background:#e2e8f0;
            color:#0f172a;
        }
        .helper-link{
            display:inline-block;
            margin-top:10px;
            color:#0b66ff;
            text-decoration:none;
            font-weight:700;
        }
        .msg-title{
            font-weight:700;
            margin-bottom:6px;
        }
        @media (max-width:700px){
            .grid{grid-template-columns:1fr}
            .actions button{width:100%}
            .helper-actions{flex-direction:column}
            .helper-btn{width:100%;text-align:center}
        }
    </style>
</head>
<body>
<div class="wrap">
    <div class="card" id="step-form">
        <h1>Регистрация на мероприятие</h1>
        <p class="muted">После подтверждения кода вы попадёте в личный кабинет и увидите свои мероприятия.</p>

        <?php if (!$events): ?>
            <div class="msg err">Сейчас нет активных мероприятий для регистрации.</div>
        <?php else: ?>
            <form id="regForm" class="grid" action="<?= e(base_url('/api/register.php')) ?>" method="post" novalidate>
                <div class="full">
                    <label for="event_id">Мероприятие</label>
                    <select id="event_id" name="event_id" required>
                        <?php foreach ($events as $event): ?>
                            <option value="<?= (int)$event['id'] ?>">
                                <?= e($event['title']) ?> — <?= e(date('d.m.Y', strtotime($event['event_date']))) ?> <?= e(substr($event['event_time'], 0, 5)) ?>
                            </option>
                        <?php endforeach; ?>
                    </select>
                </div>

                <div>
                    <label for="last_name">Фамилия</label>
                    <input id="last_name" name="last_name" required>
                </div>

                <div>
                    <label for="first_name">Имя</label>
                    <input id="first_name" name="first_name" required>
                </div>

                <div class="full">
                    <label for="patronymic">Отчество</label>
                    <input id="patronymic" name="patronymic">
                </div>

                <div class="full">
                    <label for="organization">Организация</label>
                    <input id="organization" name="organization" required>
                </div>

                <div>
                    <label for="district">Округ</label>
                    <select id="district" name="district" required>
                        <?php foreach ($districts as $district): ?>
                            <option value="<?= e($district) ?>"><?= e($district) ?></option>
                        <?php endforeach; ?>
                    </select>
                </div>

                <div>
                    <label for="email">Почта</label>
                    <input id="email" name="email" type="email" required autocomplete="email">
                </div>

                <div>
                    <label for="is_union_member">Член профсоюза</label>
                    <select id="is_union_member" name="is_union_member">
                        <option value="1">Да</option>
                        <option value="0" selected>Нет</option>
                    </select>
                </div>

                <div>
                    <label for="union_ticket">Номер профсоюзного билета</label>
                    <input id="union_ticket" name="union_ticket" placeholder="Необязательно">
                </div>

                <div class="full">
                    <label for="extra_info">Другое</label>
                    <textarea id="extra_info" name="extra_info" placeholder="Дополнительная информация от участника"></textarea>
                </div>

                <div class="full actions">
                    <button type="submit" id="submitBtn">Зарегистрироваться</button>
                </div>

                <div id="formMsg" class="full"></div>
            </form>
        <?php endif; ?>
    </div>

    <div class="card hidden" id="step-otp">
        <h2>Подтверждение почты</h2>
        <p class="muted">Код отправлен на указанный email. Введите его ниже.</p>

        <div class="grid">
            <div class="full">
                <label for="otp">6-значный код</label>
                <input
                    class="otp"
                    id="otp"
                    maxlength="6"
                    inputmode="numeric"
                    autocomplete="one-time-code"
                    placeholder="000000"
                >
            </div>

            <div class="full actions">
                <button type="button" id="verifyBtn">Подтвердить</button>
                <button type="button" id="backBtn">Назад</button>
            </div>

            <div id="otpMsg" class="full"></div>
        </div>
    </div>
</div>

<script>
const form = document.getElementById('regForm');
const stepForm = document.getElementById('step-form');
const stepOtp = document.getElementById('step-otp');
const formMsg = document.getElementById('formMsg');
const otpMsg = document.getElementById('otpMsg');
const submitBtn = document.getElementById('submitBtn');
const verifyBtn = document.getElementById('verifyBtn');
const backBtn = document.getElementById('backBtn');
const otpInput = document.getElementById('otp');

const LOCK_SECONDS = 10;
const LOCK_KEY = 'selector_reg_form_lock_until';

const LOGIN_URL = '<?= e(base_url('/cabinet/login.php')) ?>';
const RESET_URL = '<?= e(base_url('/cabinet/forgot_password.php')) ?>';
const VERIFY_URL = '<?= e(base_url('/api/verify_otp.php')) ?>';
const CABINET_URL = '<?= e(base_url('/cabinet/')) ?>';

function goToTop(url) {
    try {
        if (window.top && window.top !== window.self) {
            window.top.location.href = url;
        } else {
            window.location.href = url;
        }
    } catch (e) {
        window.location.href = url;
    }
}

function escapeHtml(value) {
    return String(value ?? '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

function showMsg(node, text, type = 'err', extraHtml = '') {
    if (!node) return;
    const classMap = { ok: 'ok', err: 'err', info: 'info' };
    node.className = 'msg ' + (classMap[type] || 'err');

    if (extraHtml) {
        node.innerHTML = '<div>' + escapeHtml(text) + '</div>' + extraHtml;
    } else {
        node.textContent = text;
    }
}

function clearMsg(node) {
    if (!node) return;
    node.className = '';
    node.innerHTML = '';
    node.textContent = '';
}

function lockUntil() {
    return parseInt(localStorage.getItem(LOCK_KEY) || '0', 10);
}

function setLock(seconds) {
    localStorage.setItem(LOCK_KEY, String(Date.now() + seconds * 1000));
}

function updateSubmitState() {
    if (!submitBtn) return false;

    const until = lockUntil();
    const now = Date.now();

    if (until > now) {
        submitBtn.disabled = true;
        submitBtn.textContent = 'Подождите ' + Math.max(1, Math.ceil((until - now) / 1000)) + ' сек...';
        return true;
    }

    submitBtn.disabled = false;
    submitBtn.textContent = 'Зарегистрироваться';
    return false;
}

function normalizeEmailField() {
    const emailField = document.getElementById('email');
    if (emailField && emailField.value) {
        emailField.value = emailField.value.trim().toLowerCase();
    }
}

function renderAlreadyRegistered(data) {
    const loginUrl = data && data.redirect ? data.redirect : LOGIN_URL;
    const html = `
        <div class="msg-title">Вы уже зарегистрированы на это мероприятие.</div>
        <div>Войдите в личный кабинет по email и паролю.<br>Если пароль не пришёл, запросите новый пароль.</div>
        <div class="helper-actions">
            <a class="helper-btn login" href="${escapeHtml(loginUrl)}">Войти в личный кабинет</a>
            <a class="helper-btn reset" href="${escapeHtml(RESET_URL)}">Выслать новый пароль</a>
        </div>
    `;
    showMsg(formMsg, data.message || 'Вы уже зарегистрированы на это мероприятие.', 'ok', html);
}

async function parseJsonSafe(response) {
    const text = await response.text();
    try {
        return JSON.parse(text);
    } catch (e) {
        return {
            status: 'error',
            message: text && text.trim() ? text.trim() : 'Сервер вернул некорректный ответ.'
        };
    }
}

updateSubmitState();
setInterval(updateSubmitState, 300);

if (form) {
    form.addEventListener('submit', async function (e) {
        e.preventDefault();

        if (updateSubmitState()) {
            showMsg(formMsg, 'Повторная отправка временно заблокирована. Подождите несколько секунд.', 'err');
            return;
        }

        normalizeEmailField();

        if (!form.reportValidity()) {
            return;
        }

        setLock(LOCK_SECONDS);
        updateSubmitState();
        clearMsg(formMsg);
        clearMsg(otpMsg);

        const fd = new FormData(form);

        try {
            const res = await fetch(form.action, {
                method: 'POST',
                body: fd,
                credentials: 'same-origin',
                headers: {
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const data = await parseJsonSafe(res);

            if (data.status === 'success' || data.status === 'pending') {
                stepForm.classList.add('hidden');
                stepOtp.classList.remove('hidden');
                showMsg(otpMsg, data.message || 'Код отправлен. Проверьте почту.', 'ok');
                if (otpInput) {
                    otpInput.focus();
                }
            } else if (data.status === 'already_registered') {
                renderAlreadyRegistered(data);
            } else {
                showMsg(formMsg, data.message || 'Ошибка регистрации.', 'err');
            }
        } catch (error) {
            showMsg(formMsg, 'Ошибка соединения с сервером.', 'err');
        }
    });
}

if (verifyBtn) {
    verifyBtn.addEventListener('click', async function () {
        clearMsg(otpMsg);
        verifyBtn.disabled = true;

        normalizeEmailField();

        const emailField = document.getElementById('email');
        const email = emailField ? emailField.value.trim().toLowerCase() : '';
        const otp = otpInput ? otpInput.value.trim() : '';

        if (!email) {
            showMsg(otpMsg, 'Не найден email для подтверждения. Вернитесь назад и заполните форму снова.', 'err');
            verifyBtn.disabled = false;
            return;
        }

        if (!/^\d{6}$/.test(otp)) {
            showMsg(otpMsg, 'Введите 6-значный код.', 'err');
            verifyBtn.disabled = false;
            if (otpInput) otpInput.focus();
            return;
        }

        const fd = new FormData();
        fd.append('email', email);
        fd.append('otp', otp);

        try {
            const res = await fetch(VERIFY_URL, {
                method: 'POST',
                body: fd,
                credentials: 'same-origin',
                headers: {
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const data = await parseJsonSafe(res);

		if (data.status === 'success') {
    showMsg(otpMsg, 'Подтверждение успешно. Переходим в кабинет…', 'ok');
    const targetUrl = data.redirect || CABINET_URL;
    setTimeout(() => goToTop(targetUrl), 300);
} else {
    showMsg(otpMsg, data.message || 'Неверный код.', 'err');
}
        } catch (error) {
            showMsg(otpMsg, 'Ошибка проверки кода.', 'err');
        } finally {
            verifyBtn.disabled = false;
        }
    });
}

if (otpInput) {
    otpInput.addEventListener('keydown', function (e) {
        if (e.key === 'Enter') {
            e.preventDefault();
            if (verifyBtn && !verifyBtn.disabled) {
                verifyBtn.click();
            }
        }
    });
}

if (backBtn) {
    backBtn.addEventListener('click', function () {
        stepOtp.classList.add('hidden');
        stepForm.classList.remove('hidden');
        clearMsg(otpMsg);
    });
}
</script>
</body>
</html>