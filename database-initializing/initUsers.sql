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


CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL, -- Ссылка на твою таблицу пользователей
    refresh_token VARCHAR(255) UNIQUE NOT NULL,
    user_agent TEXT, -- Чтобы показывать "вход с iPhone / Chrome"
    client_ip VARCHAR(45),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
	CONSTRAINT fk_user 
        FOREIGN KEY (user_id) 
        REFERENCES users(id) 
        ON DELETE CASCADE
);