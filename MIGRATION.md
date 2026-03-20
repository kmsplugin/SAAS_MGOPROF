# MIGRATION.md — PHP/SQLite → Go/PostgreSQL
# Система регистрации MGOPROF

## Что имеем (PHP/SQLite)

### Схема БД (4 таблицы)
```
reg_events        — мероприятия
reg_users         — участники (email уникален)
reg_registrations — регистрации на мероприятия + OTP
reg_logs          — лог событий
```

### Бизнес-логика
1. Участник заполняет форму → POST /api/register.php
2. Система создаёт/обновляет пользователя
3. Генерирует OTP (6 цифр, TTL 10 мин)
4. Отправляет OTP + пароль кабинета на email
5. Участник вводит OTP → POST /api/verify_otp.php
6. Статус регистрации → 'verified'
7. Личный кабинет — войти по email+пароль

### Что хорошо сделано (сохранить логику)
- Cooldown 10 сек на повторную отправку OTP
- Статусы: pending → verified
- Гео по IP (MaxMind)
- Автовыдача пароля новым пользователям
- Защита от дублей (unique event_id + user_id)

---

## Целевая архитектура (Go)

```
mgoprof-saas/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── handler/
│   │   ├── event_handler.go
│   │   ├── registration_handler.go
│   │   ├── auth_handler.go
│   │   └── admin_handler.go
│   ├── service/
│   │   ├── registration_service.go
│   │   ├── auth_service.go
│   │   └── event_service.go
│   ├── repository/
│   │   ├── user_repo.go
│   │   ├── event_repo.go
│   │   └── registration_repo.go
│   ├── model/
│   │   └── models.go
│   ├── middleware/
│   │   └── middleware.go
│   └── mailer/
│       └── mailer.go
├── migrations/
│   ├── 001_create_events.up.sql
│   ├── 002_create_users.up.sql
│   ├── 003_create_registrations.up.sql
│   └── 004_create_logs.up.sql
├── frontend/          (Next.js — отдельно)
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── CLAUDE.md
```

---

## PostgreSQL схема (миграция с SQLite)

### migrations/001_create_events.up.sql
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE reg_events (
    id          SERIAL PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT,
    event_date  DATE NOT NULL,
    event_time  TIME NOT NULL,
    cabinet_link TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ
);

CREATE INDEX idx_events_is_active ON reg_events(is_active);
```

### migrations/002_create_users.up.sql
```sql
CREATE TABLE reg_users (
    id                  SERIAL PRIMARY KEY,
    email               TEXT NOT NULL UNIQUE,
    last_name           TEXT NOT NULL,
    first_name          TEXT NOT NULL,
    patronymic          TEXT,
    organization        TEXT NOT NULL,
    district            TEXT NOT NULL,
    is_union_member     BOOLEAN NOT NULL DEFAULT FALSE,
    union_ticket        TEXT,
    extra_info          TEXT,
    last_ip             TEXT,
    geo_country         TEXT,
    geo_region          TEXT,
    geo_city            TEXT,
    user_agent          TEXT,
    password_hash       TEXT,
    password_updated_at TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON reg_users(email);
```

### migrations/003_create_registrations.up.sql
```sql
CREATE TABLE reg_registrations (
    id              SERIAL PRIMARY KEY,
    event_id        INTEGER NOT NULL REFERENCES reg_events(id) ON DELETE CASCADE,
    user_id         INTEGER NOT NULL REFERENCES reg_users(id) ON DELETE CASCADE,
    otp_code        TEXT NOT NULL,
    otp_expires_at  TIMESTAMPTZ NOT NULL,
    otp_verified_at TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'verified', 'cancelled')),
    ip_address      TEXT,
    geo_country     TEXT,
    geo_region      TEXT,
    geo_city        TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ,
    UNIQUE(event_id, user_id)
);

CREATE INDEX idx_reg_event_user  ON reg_registrations(event_id, user_id);
CREATE INDEX idx_reg_status      ON reg_registrations(status);
CREATE INDEX idx_reg_created_at  ON reg_registrations(created_at DESC);
```

### migrations/004_create_logs.up.sql
```sql
CREATE TABLE reg_logs (
    id          SERIAL PRIMARY KEY,
    event_type  TEXT,
    user_email  TEXT,
    ip_address  TEXT,
    message     TEXT,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_logs_created_at  ON reg_logs(created_at DESC);
CREATE INDEX idx_logs_user_email  ON reg_logs(user_email);
CREATE INDEX idx_logs_event_type  ON reg_logs(event_type);
```

---

## Go код

### go.mod
```go
module mgoprof-saas

go 1.23

require (
    github.com/gin-gonic/gin v1.10.0
    github.com/lib/pq v1.10.9
    github.com/jmoiron/sqlx v1.4.0
    github.com/golang-jwt/jwt/v5 v5.2.1
    github.com/google/uuid v1.6.0
    github.com/resend/resend-go/v2 v2.0.0
    github.com/oschwald/maxminddb-golang v1.13.0
    golang.org/x/crypto v0.23.0
    github.com/joho/godotenv v1.5.1
    go.uber.org/zap v1.27.0
)
```

---

### internal/model/models.go
```go
package model

import "time"

type Event struct {
    ID          int        `db:"id"           json:"id"`
    Title       string     `db:"title"        json:"title"`
    Description string     `db:"description"  json:"description"`
    EventDate   string     `db:"event_date"   json:"event_date"`
    EventTime   string     `db:"event_time"   json:"event_time"`
    CabinetLink string     `db:"cabinet_link" json:"cabinet_link"`
    IsActive    bool       `db:"is_active"    json:"is_active"`
    CreatedAt   time.Time  `db:"created_at"   json:"created_at"`
}

type User struct {
    ID                 int        `db:"id"                   json:"id"`
    Email              string     `db:"email"                json:"email"`
    LastName           string     `db:"last_name"            json:"last_name"`
    FirstName          string     `db:"first_name"           json:"first_name"`
    Patronymic         string     `db:"patronymic"           json:"patronymic"`
    Organization       string     `db:"organization"         json:"organization"`
    District           string     `db:"district"             json:"district"`
    IsUnionMember      bool       `db:"is_union_member"      json:"is_union_member"`
    UnionTicket        string     `db:"union_ticket"         json:"union_ticket"`
    PasswordHash       string     `db:"password_hash"        json:"-"`
    LastIP             string     `db:"last_ip"              json:"-"`
    GeoCountry         string     `db:"geo_country"          json:"geo_country"`
    GeoRegion          string     `db:"geo_region"           json:"geo_region"`
    GeoCity            string     `db:"geo_city"             json:"geo_city"`
    CreatedAt          time.Time  `db:"created_at"           json:"created_at"`
}

type Registration struct {
    ID            int        `db:"id"              json:"id"`
    EventID       int        `db:"event_id"        json:"event_id"`
    UserID        int        `db:"user_id"         json:"user_id"`
    OTPCode       string     `db:"otp_code"        json:"-"`
    OTPExpiresAt  time.Time  `db:"otp_expires_at"  json:"-"`
    OTPVerifiedAt *time.Time `db:"otp_verified_at" json:"otp_verified_at"`
    Status        string     `db:"status"          json:"status"`
    IPAddress     string     `db:"ip_address"      json:"-"`
    GeoCountry    string     `db:"geo_country"     json:"geo_country"`
    CreatedAt     time.Time  `db:"created_at"      json:"created_at"`
    UpdatedAt     *time.Time `db:"updated_at"      json:"updated_at"`
}

// Request/Response DTOs

type RegisterRequest struct {
    Email         string `json:"email"          binding:"required,email"`
    EventID       int    `json:"event_id"       binding:"required,min=1"`
    FirstName     string `json:"first_name"     binding:"required"`
    LastName      string `json:"last_name"      binding:"required"`
    Patronymic    string `json:"patronymic"`
    Organization  string `json:"organization"   binding:"required"`
    District      string `json:"district"       binding:"required"`
    IsUnionMember bool   `json:"is_union_member"`
    UnionTicket   string `json:"union_ticket"`
    ExtraInfo     string `json:"extra_info"`
}

type VerifyOTPRequest struct {
    Email   string `json:"email"    binding:"required,email"`
    EventID int    `json:"event_id" binding:"required"`
    OTP     string `json:"otp"      binding:"required,len=6"`
}

type LoginRequest struct {
    Email    string `json:"email"    binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

type ErrorResponse struct {
    Status  string `json:"status"`
    Message string `json:"message"`
}
```

---

### internal/service/registration_service.go
```go
package service

import (
    "context"
    "crypto/rand"
    "fmt"
    "math/big"
    "strings"
    "time"

    "golang.org/x/crypto/bcrypt"
    "go.uber.org/zap"

    "mgoprof-saas/internal/model"
    "mgoprof-saas/internal/repository"
    "mgoprof-saas/internal/mailer"
)

const (
    ResendCooldownSeconds = 10
    OTPLength             = 6
    OTPTTLMinutes         = 10
    PasswordLength        = 10
)

type RegistrationService struct {
    userRepo  *repository.UserRepository
    eventRepo *repository.EventRepository
    regRepo   *repository.RegistrationRepository
    mailer    *mailer.Mailer
    logger    *zap.Logger
    siteURL   string
}

func NewRegistrationService(
    userRepo *repository.UserRepository,
    eventRepo *repository.EventRepository,
    regRepo *repository.RegistrationRepository,
    mailer *mailer.Mailer,
    logger *zap.Logger,
    siteURL string,
) *RegistrationService {
    return &RegistrationService{
        userRepo:  userRepo,
        eventRepo: eventRepo,
        regRepo:   regRepo,
        mailer:    mailer,
        logger:    logger,
        siteURL:   siteURL,
    }
}

type RegisterResult struct {
    Status          string `json:"status"`
    Message         string `json:"message"`
    ShowOTP         bool   `json:"show_otp"`
    PasswordIssued  bool   `json:"password_issued"`
    CabinetLoginURL string `json:"cabinet_login"`
}

// Register — полная логика регистрации (перенос из PHP register.php)
func (s *RegistrationService) Register(ctx context.Context, req model.RegisterRequest, ip, userAgent string) (*RegisterResult, error) {
    email := strings.ToLower(strings.TrimSpace(req.Email))
    cabinetURL := fmt.Sprintf("%s/cabinet/login?email=%s", s.siteURL, email)

    // Проверяем мероприятие
    event, err := s.eventRepo.FindActiveByID(ctx, req.EventID)
    if err != nil || event == nil {
        return nil, fmt.Errorf("мероприятие недоступно для регистрации")
    }

    otp := generateOTP()
    otpExpiresAt := time.Now().Add(OTPTTLMinutes * time.Minute)

    var userPassword *string
    passwordIssued := false
    needSendOTP := true
    message := "Код отправлен на почту."

    // Получаем гео по IP
    geo := getGeoByIP(ip)

    // Транзакция
    err = s.userRepo.WithTx(ctx, func(ctx context.Context) error {

        // Найти или создать пользователя
        user, err := s.userRepo.FindByEmail(ctx, email)
        if err != nil {
            return fmt.Errorf("ошибка поиска пользователя: %w", err)
        }

        var userID int

        if user != nil {
            userID = user.ID

            // Обновляем данные
            err = s.userRepo.Update(ctx, userID, req, ip, geo, userAgent)
            if err != nil {
                return fmt.Errorf("ошибка обновления пользователя: %w", err)
            }

            // Выдаём пароль если ещё нет
            if user.PasswordHash == "" {
                pwd := generatePassword(PasswordLength)
                hash, _ := bcrypt.GenerateFromPassword([]byte(pwd), 12)
                err = s.userRepo.SetPassword(ctx, userID, string(hash))
                if err != nil {
                    return err
                }
                userPassword = &pwd
                passwordIssued = true
            }
        } else {
            // Новый пользователь
            pwd := generatePassword(PasswordLength)
            hash, _ := bcrypt.GenerateFromPassword([]byte(pwd), 12)

            userID, err = s.userRepo.Create(ctx, email, req, ip, geo, userAgent, string(hash))
            if err != nil {
                return fmt.Errorf("ошибка создания пользователя: %w", err)
            }
            userPassword = &pwd
            passwordIssued = true
        }

        // Проверяем существующую регистрацию
        existingReg, err := s.regRepo.FindByEventAndUser(ctx, req.EventID, userID)
        if err != nil {
            return err
        }

        if existingReg != nil {
            // Уже верифицирован
            if existingReg.Status == "verified" {
                return &AlreadyRegisteredError{CabinetURL: cabinetURL}
            }

            // Cooldown проверка
            lastActivity := existingReg.UpdatedAt
            if lastActivity == nil {
                lastActivity = &existingReg.CreatedAt
            }
            secondsSince := time.Since(*lastActivity).Seconds()

            if secondsSince < ResendCooldownSeconds {
                needSendOTP = false
                message = "Регистрация уже начата. Проверьте письмо и введите код подтверждения."
            } else {
                err = s.regRepo.UpdateOTP(ctx, existingReg.ID, otp, otpExpiresAt, ip, geo)
                if err != nil {
                    return err
                }
                message = "Новый код отправлен на почту."
            }
        } else {
            // Новая регистрация
            err = s.regRepo.Create(ctx, req.EventID, userID, otp, otpExpiresAt, ip, geo)
            if err != nil {
                return err
            }
        }

        return nil
    })

    if err != nil {
        if alreadyErr, ok := err.(*AlreadyRegisteredError); ok {
            return &RegisterResult{
                Status:          "already_registered",
                Message:         "Вы уже зарегистрированы. Войдите в кабинет.",
                CabinetLoginURL: alreadyErr.CabinetURL,
            }, nil
        }
        return nil, err
    }

    // Отправляем письмо
    if needSendOTP {
        err = s.mailer.SendRegistration(email, req.FirstName, otp, userPassword, cabinetURL, event.Title)
        if err != nil {
            s.logger.Error("email send failed",
                zap.String("email", email),
                zap.Error(err),
            )
            return nil, fmt.Errorf("регистрация сохранена, но письмо не отправлено")
        }
    }

    if passwordIssued {
        message += " Пароль для кабинета также отправлен на почту."
    }

    status := "success"
    if !needSendOTP {
        status = "pending"
    }

    return &RegisterResult{
        Status:          status,
        Message:         message,
        ShowOTP:         true,
        PasswordIssued:  passwordIssued,
        CabinetLoginURL: cabinetURL,
    }, nil
}

// VerifyOTP — подтверждение кода
func (s *RegistrationService) VerifyOTP(ctx context.Context, req model.VerifyOTPRequest) error {
    email := strings.ToLower(strings.TrimSpace(req.Email))

    user, err := s.userRepo.FindByEmail(ctx, email)
    if err != nil || user == nil {
        return fmt.Errorf("пользователь не найден")
    }

    reg, err := s.regRepo.FindByEventAndUser(ctx, req.EventID, user.ID)
    if err != nil || reg == nil {
        return fmt.Errorf("регистрация не найдена")
    }

    if reg.Status == "verified" {
        return fmt.Errorf("регистрация уже подтверждена")
    }

    if time.Now().After(reg.OTPExpiresAt) {
        return fmt.Errorf("код подтверждения истёк. Запросите новый.")
    }

    if reg.OTPCode != req.OTP {
        return fmt.Errorf("неверный код подтверждения")
    }

    return s.regRepo.SetVerified(ctx, reg.ID)
}

// generateOTP — 6-значный код
func generateOTP() string {
    n, _ := rand.Int(rand.Reader, big.NewInt(1_000_000))
    return fmt.Sprintf("%06d", n.Int64())
}

// generatePassword — случайный пароль
func generatePassword(length int) string {
    alphabet := "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"
    b := make([]byte, length)
    for i := range b {
        n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
        b[i] = alphabet[n.Int64()]
    }
    return string(b)
}

type AlreadyRegisteredError struct {
    CabinetURL string
}

func (e *AlreadyRegisteredError) Error() string {
    return "already_registered"
}
```

---

### internal/handler/registration_handler.go
```go
package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"

    "mgoprof-saas/internal/model"
    "mgoprof-saas/internal/service"
)

type RegistrationHandler struct {
    svc    *service.RegistrationService
    logger *zap.Logger
}

func NewRegistrationHandler(svc *service.RegistrationService, logger *zap.Logger) *RegistrationHandler {
    return &RegistrationHandler{svc: svc, logger: logger}
}

func (h *RegistrationHandler) RegisterRoutes(r *gin.RouterGroup) {
    r.POST("/register", h.Register)
    r.POST("/verify-otp", h.VerifyOTP)
}

// Register godoc
// @Summary     Регистрация на мероприятие
// @Tags        Registration
// @Accept      json
// @Produce     json
// @Param       body body model.RegisterRequest true "Данные участника"
// @Success     200 {object} service.RegisterResult
// @Failure     422 {object} model.ErrorResponse
// @Router      /api/register [post]
func (h *RegistrationHandler) Register(c *gin.Context) {
    var req model.RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusUnprocessableEntity, model.ErrorResponse{
            Status:  "error",
            Message: "Заполните обязательные поля.",
        })
        return
    }

    ip := extractIP(c)
    ua := c.Request.UserAgent()

    result, err := h.svc.Register(c.Request.Context(), req, ip, ua)
    if err != nil {
        h.logger.Error("registration failed",
            zap.String("email", req.Email),
            zap.Error(err),
        )
        c.JSON(http.StatusInternalServerError, model.ErrorResponse{
            Status:  "error",
            Message: err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, result)
}

// VerifyOTP godoc
// @Summary     Подтверждение OTP кода
// @Tags        Registration
// @Accept      json
// @Produce     json
// @Param       body body model.VerifyOTPRequest true "Email + OTP"
// @Success     200 {object} map[string]string
// @Failure     400 {object} model.ErrorResponse
// @Router      /api/verify-otp [post]
func (h *RegistrationHandler) VerifyOTP(c *gin.Context) {
    var req model.VerifyOTPRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, model.ErrorResponse{
            Status:  "error",
            Message: "Неверный формат запроса.",
        })
        return
    }

    if err := h.svc.VerifyOTP(c.Request.Context(), req); err != nil {
        c.JSON(http.StatusBadRequest, model.ErrorResponse{
            Status:  "error",
            Message: err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status":  "success",
        "message": "Регистрация подтверждена!",
    })
}

func extractIP(c *gin.Context) string {
    if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
        return ip
    }
    if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
        return ip
    }
    return c.ClientIP()
}
```

---

### internal/mailer/mailer.go
```go
package mailer

import (
    "fmt"
    "github.com/resend/resend-go/v2"
)

type Mailer struct {
    client    *resend.Client
    fromEmail string
    fromName  string
    siteURL   string
}

func New(apiKey, fromEmail, fromName, siteURL string) *Mailer {
    return &Mailer{
        client:    resend.NewClient(apiKey),
        fromEmail: fromEmail,
        fromName:  fromName,
        siteURL:   siteURL,
    }
}

func (m *Mailer) SendRegistration(
    to, firstName, otp string,
    password *string,
    loginURL, eventTitle string,
) error {
    safeName := firstName
    if safeName == "" {
        safeName = "участник"
    }

    passwordBlock := ""
    if password != nil && *password != "" {
        passwordBlock = fmt.Sprintf(`
            <p><strong>Пароль для личного кабинета:</strong> %s</p>
            <p>Ссылка для входа: <a href="%s">%s</a></p>
        `, *password, loginURL, loginURL)
    }

    html := fmt.Sprintf(`
        <p>Здравствуйте, %s!</p>
        <p>Вы начали регистрацию на мероприятие:</p>
        <p><strong>%s</strong></p>
        <p><strong>Код подтверждения (OTP):</strong> %s</p>
        %s
        <p>Если вы не запрашивали регистрацию, просто проигнорируйте это письмо.</p>
    `, safeName, eventTitle, otp, passwordBlock)

    params := &resend.SendEmailRequest{
        From:    fmt.Sprintf("%s <%s>", m.fromName, m.fromEmail),
        To:      []string{to},
        Subject: "Подтверждение регистрации",
        Html:    html,
    }

    _, err := m.client.Emails.Send(params)
    return err
}

func (m *Mailer) SendAdminOTP(to, name, otp string) error {
    html := fmt.Sprintf(`
        <p>Здравствуйте, %s!</p>
        <p>Код подтверждения для входа в панель администратора:</p>
        <p><strong>%s</strong></p>
        <p>Код действителен 10 минут.</p>
    `, name, otp)

    params := &resend.SendEmailRequest{
        From:    fmt.Sprintf("%s <%s>", m.fromName, m.fromEmail),
        To:      []string{to},
        Subject: "Код входа в панель администратора",
        Html:    html,
    }

    _, err := m.client.Emails.Send(params)
    return err
}
```

---

### cmd/server/main.go
```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    "github.com/joho/godotenv"
    "go.uber.org/zap"

    "mgoprof-saas/internal/handler"
    "mgoprof-saas/internal/mailer"
    "mgoprof-saas/internal/repository"
    "mgoprof-saas/internal/service"
)

func main() {
    // Загружаем .env
    _ = godotenv.Load()

    // Logger
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    // БД
    db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatalf("DB connect failed: %v", err)
    }
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    defer db.Close()

    // Репозитории
    userRepo  := repository.NewUserRepository(db)
    eventRepo := repository.NewEventRepository(db)
    regRepo   := repository.NewRegistrationRepository(db)

    // Mailer (Resend — работает из России)
    mail := mailer.New(
        os.Getenv("RESEND_API_KEY"),
        os.Getenv("MAIL_FROM_EMAIL"),
        os.Getenv("MAIL_FROM_NAME"),
        os.Getenv("SITE_URL"),
    )

    // Сервисы
    regSvc := service.NewRegistrationService(
        userRepo, eventRepo, regRepo, mail, logger,
        os.Getenv("SITE_URL"),
    )
    authSvc := service.NewAuthService(userRepo, logger)

    // Хендлеры
    regHandler  := handler.NewRegistrationHandler(regSvc, logger)
    authHandler := handler.NewAuthHandler(authSvc, logger)

    // Роутер
    r := gin.New()
    r.Use(gin.Recovery())
    r.Use(corsMiddleware())

    // API
    api := r.Group("/api")
    regHandler.RegisterRoutes(api)
    authHandler.RegisterRoutes(api)

    // Health check
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now()})
    })

    // Graceful shutdown
    srv := &http.Server{
        Addr:    ":" + getEnv("PORT", "8080"),
        Handler: r,
    }

    go func() {
        logger.Info("server started", zap.String("addr", srv.Addr))
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("server error", zap.Error(err))
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
    logger.Info("server stopped")
}

func corsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", os.Getenv("ALLOWED_ORIGIN"))
        c.Header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        c.Next()
    }
}

func getEnv(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}
```

---

### .env (шаблон)
```env
DATABASE_URL=postgres://mgoprof:secret@localhost:5432/mgoprof
PORT=8080
SITE_URL=https://mgoprof.ru

RESEND_API_KEY=re_xxxxxxxxxx
MAIL_FROM_EMAIL=noreply@mgoprof.ru
MAIL_FROM_NAME=MGOPROF

JWT_SECRET=ваш_секрет_минимум_32_символа
ALLOWED_ORIGIN=https://mgoprof.ru

ADMIN_EMAIL=kms-oleg@mail.ru
ADMIN_PASSWORD_HASH=$2y$12$oTudt5e4PyXCKwKCnpwmLOjJ2VyS0s1otTF0Bn/yc2RyodPqE/x/.
```

---

### docker-compose.yml
```yaml
version: '3.9'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: mgoprof
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: mgoprof
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U mgoprof"]
      interval: 5s

  app:
    build: .
    env_file: .env
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  postgres_data:
```

---

### Dockerfile
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server/main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

---

## Миграция данных из SQLite в PostgreSQL

```bash
# 1. Экспорт из SQLite
sqlite3 database.sqlite ".mode csv" ".output events.csv" "SELECT * FROM reg_events;"
sqlite3 database.sqlite ".mode csv" ".output users.csv" "SELECT * FROM reg_users;"
sqlite3 database.sqlite ".mode csv" ".output registrations.csv" "SELECT * FROM reg_registrations;"
sqlite3 database.sqlite ".mode csv" ".output logs.csv" "SELECT * FROM reg_logs;"

# 2. Импорт в PostgreSQL
psql $DATABASE_URL -c "\COPY reg_events FROM 'events.csv' CSV HEADER"
psql $DATABASE_URL -c "\COPY reg_users FROM 'users.csv' CSV HEADER"
psql $DATABASE_URL -c "\COPY reg_registrations FROM 'registrations.csv' CSV HEADER"
psql $DATABASE_URL -c "\COPY reg_logs FROM 'logs.csv' CSV HEADER"

# 3. Сбросить sequences (SERIAL счётчики)
psql $DATABASE_URL -c "SELECT setval('reg_events_id_seq', (SELECT MAX(id) FROM reg_events))"
psql $DATABASE_URL -c "SELECT setval('reg_users_id_seq', (SELECT MAX(id) FROM reg_users))"
psql $DATABASE_URL -c "SELECT setval('reg_registrations_id_seq', (SELECT MAX(id) FROM reg_registrations))"
```

---

## Порядок работы с Claude Code

```bash
# Шаг 1 — создаём структуру
/project:feature "создай структуру Go-проекта по MIGRATION.md"

# Шаг 2 — миграции БД
/project:feature "создай SQL-миграции из раздела PostgreSQL схема"

# Шаг 3 — репозитории
/project:feature "напиши UserRepository, EventRepository, RegistrationRepository"

# Шаг 4 — сервис регистрации
/project:feature "реализуй RegistrationService по коду из MIGRATION.md"

# Шаг 5 — тесты
/project:test-and-fix

# Шаг 6 — проверка
/project:review
```
