-- Включаем расширение для генерации UUID, если оно еще не включено
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Почта для связи и подтверждения
    email VARCHAR(250) NOT NULL UNIQUE,
    
    -- Статус активации (по умолчанию FALSE)
    is_activated BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Хэш пароля 
    hash_password TEXT NOT NULL,
    
    -- Время регистрации с часовым поясом
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

--- ИНДЕКСЫ ДЛЯ ОПТИМИЗАЦИИ ---

CREATE INDEX IF NOT EXISTS idx_users_email_lower ON users (LOWER(email));

CREATE INDEX IF NOT EXISTS idx_users_active ON users (is_activated) WHERE is_activated = TRUE;