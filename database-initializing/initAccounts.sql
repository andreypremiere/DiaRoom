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
    avatar_url TEXT DEFAULT '',
    background_url TEXT DEFAULT '',
    bio TEXT DEFAULT '',
    settings JSONB NOT NULL DEFAULT '{}',
    followers_count INT NOT NULL DEFAULT 0,
    following_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_user_room
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

-- Таблица категорий
CREATE TABLE IF NOT EXISTS categories (
    slug VARCHAR(100) PRIMARY KEY, -- Теперь это PK
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Связующая таблица
CREATE TABLE IF NOT EXISTS room_categories (
    room_id UUID NOT NULL,
    category_slug VARCHAR(50) NOT NULL, -- Ссылаемся на слаг
    PRIMARY KEY (room_id, category_slug),
    CONSTRAINT fk_room FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    CONSTRAINT fk_category FOREIGN KEY (category_slug) REFERENCES categories(slug) ON DELETE CASCADE ON UPDATE CASCADE
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
CREATE INDEX IF NOT EXISTS idx_room_categories_cat_id ON room_categories(category_slug);

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

CREATE TABLE IF NOT EXISTS subscriptions (
    -- Кто подписывается
    follower_id UUID NOT NULL,
    -- На кого подписываются
    following_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- нельзя подписаться дважды
    PRIMARY KEY (follower_id, following_id),

    -- нельзя подписаться на самого себя
    CONSTRAINT check_self_follow CHECK (follower_id <> following_id),

    -- Внешние ключи
    CONSTRAINT fk_follower 
        FOREIGN KEY (follower_id) REFERENCES rooms(id) ON DELETE CASCADE,
    CONSTRAINT fk_following 
        FOREIGN KEY (following_id) REFERENCES rooms(id) ON DELETE CASCADE
);

CREATE INDEX idx_subscriptions_following_id ON subscriptions(following_id);

-- Триггер на обновление счетчиков при подписке/отписке
CREATE OR REPLACE FUNCTION update_follow_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        UPDATE rooms SET following_count = following_count + 1 WHERE id = NEW.follower_id;
        UPDATE rooms SET followers_count = followers_count + 1 WHERE id = NEW.following_id;
    ELSIF (TG_OP = 'DELETE') THEN
        UPDATE rooms SET following_count = following_count - 1 WHERE id = OLD.follower_id;
        UPDATE rooms SET followers_count = followers_count - 1 WHERE id = OLD.following_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_follow_counts
AFTER INSERT OR DELETE ON subscriptions
FOR EACH ROW EXECUTE FUNCTION update_follow_counts();