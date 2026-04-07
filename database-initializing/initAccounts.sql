CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 2. Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(250) NOT NULL UNIQUE,
    is_activated BOOLEAN NOT NULL DEFAULT FALSE,
    is_configured BOOLEAN DEFAULT FALSE,
    hash_password TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. Таблица комнат
CREATE TABLE IF NOT EXISTS rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    room_name VARCHAR(100) NOT NULL,
    room_unique_id VARCHAR(100) NOT NULL UNIQUE,
    avatar_url TEXT,
    background_url TEXT,
    bio TEXT,
    settings JSONB NOT NULL DEFAULT '{}',
    followers_count INT NOT NULL DEFAULT 0,
    following_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_user_room
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

-- 4. Справочник категорий
CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 5. Связующая таблица (Многие-ко-многим)
CREATE TABLE IF NOT EXISTS room_categories (
    room_id UUID NOT NULL,
    category_id INTEGER NOT NULL,
    PRIMARY KEY (room_id, category_id),
    CONSTRAINT fk_room
        FOREIGN KEY (room_id)
        REFERENCES rooms(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_category
        FOREIGN KEY (category_id)
        REFERENCES categories(id)
        ON DELETE CASCADE
);

-- 6. Сессии пользователей
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    refresh_token VARCHAR(255) UNIQUE NOT NULL,
    user_agent TEXT,
    client_ip VARCHAR(45),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_user_session
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

--- ИНДЕКСЫ ---
CREATE INDEX IF NOT EXISTS idx_users_email_lower ON users (LOWER(email));
CREATE INDEX IF NOT EXISTS idx_users_active ON users (is_activated) WHERE is_activated = TRUE;
CREATE INDEX IF NOT EXISTS idx_rooms_unique_id_lower ON rooms (LOWER(room_unique_id));
CREATE INDEX IF NOT EXISTS idx_room_categories_cat_id ON room_categories(category_id);

--- НАПОЛНЕНИЕ ---
INSERT INTO categories (slug, name)
VALUES
    ('lifestyle-blog', 'Жизнь и Блог'),
    ('visual-arts', 'Арт и Иллюстрация'),
    ('traditional-art', 'Живопись и Рисование'),
    ('photography', 'Фотография'),
    ('3d-modeling', '3D Моделирование'),
    ('graphic-design', 'Графический дизайн'),
    ('video-production', 'Видеопроизводство'),
    ('motion-design', 'Моушн дизайн'),
    ('animation', 'Анимация'),
    ('literature', 'Литература и Статьи'),
    ('fashion', 'Мода и Стиль'),
    ('architecture-interior', 'Архитектура и Интерьер'),
    ('craft-diy', 'Крафт и DIY')
ON CONFLICT (slug)
DO UPDATE SET name = EXCLUDED.name;