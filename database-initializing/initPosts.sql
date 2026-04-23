-- =============================================
-- 1. ENUM'ы (статусы)
-- =============================================
CREATE TYPE post_status AS ENUM (
    'processing',   -- идёт загрузка файлов / генерация
	'checking',
    'published',    -- успешно опубликован
    'hidden',       -- скрыт
    'rejected',     -- отклонён
	'failed'
);

CREATE TYPE ai_check_status AS ENUM (
    'notChecked',
    'passed',
    'failed'
);

-- =============================================
-- 2. Справочник категорий
-- =============================================
CREATE TABLE IF NOT EXISTS categories (
    slug VARCHAR(100) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =============================================
-- 3. Таблица холстов (Canvases)
-- =============================================
CREATE TABLE IF NOT EXISTS canvases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =============================================
-- 4. Основная таблица постов (переделанная)
-- =============================================
CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    room_id UUID NOT NULL,                   
    category_slug VARCHAR(100) NOT NULL,
	canvas_id UUID UNIQUE,

    -- === Статусы ===
    status post_status NOT NULL ,
    ai_check_status ai_check_status NOT NULL,
    
    -- === Дополнительная информация ===
    title VARCHAR(160) NOT NULL,
    preview_url TEXT DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    
    -- === Даты и статистика ===
    published_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- === Модерация и удаление ===
    ai_check_at TIMESTAMP WITH TIME ZONE,
    ai_check_reason TEXT,

    -- === Статистика ===
    views_count BIGINT DEFAULT 0,
    likes_count BIGINT DEFAULT 0,
    comments_count BIGINT DEFAULT 0,

    -- CONSTRAINT fk_post_room     FOREIGN KEY (room_id)     REFERENCES rooms(id) ON DELETE CASCADE,
    CONSTRAINT fk_post_canvas   FOREIGN KEY (canvas_id)   REFERENCES canvases(id) ON DELETE CASCADE,
    CONSTRAINT fk_post_category FOREIGN KEY (category_slug) REFERENCES categories(slug) ON DELETE RESTRICT
);

-- =============================================
-- 5. Индексы (очень важны!)
-- =============================================
CREATE INDEX idx_posts_room_id      ON posts(room_id);
CREATE INDEX idx_posts_status       ON posts(status);
CREATE INDEX idx_posts_ai_status    ON posts(ai_check_status);
CREATE INDEX idx_posts_published_at ON posts(published_at DESC);
CREATE INDEX idx_posts_category ON posts(category_slug);

-- =============================================
-- 6. Хештеги
-- =============================================
CREATE TABLE IF NOT EXISTS hashtags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(80) NOT NULL UNIQUE,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS posts_hashtags (
    post_id UUID NOT NULL,
    hashtag_id INTEGER NOT NULL,
    PRIMARY KEY (post_id, hashtag_id),
    CONSTRAINT fk_ph_post    FOREIGN KEY (post_id)    REFERENCES posts(id)    ON DELETE CASCADE,
    CONSTRAINT fk_ph_hashtag FOREIGN KEY (hashtag_id) REFERENCES hashtags(id) ON DELETE CASCADE
);

-- =============================================
-- 7. Заполнение категорий
-- =============================================
INSERT INTO categories (slug, name)
VALUES 
    ('visual-arts', 'Арт и Иллюстрация'),
    ('traditional-art', 'Живопись и Рисование'),
    ('photography', 'Фотография'),
    ('3d-modeling', '3D Моделирование'),
    ('graphic-design', 'Графический дизайн'),
    ('video-production', 'Видеопроизводство'),
    ('motion-design', 'Моушн дизайн'),
    ('animation', 'Анимация'),
    ('music', 'Музыка'),
    ('sound-design', 'Саунд-дизайн'),
    ('podcasts', 'Подкасты'),
    ('literature', 'Литература и Статьи'),
    ('gamedev', 'Игры'),
    ('it-tech', 'Код и Технологии'),
    ('fashion', 'Мода и Стиль'),
    ('architecture-interior', 'Архитектура и Интерьер'),
    ('craft-diy', 'Крафт и DIY'),
    ('lifestyle-blog', 'Жизнь и Блог')
ON CONFLICT (slug) 
DO UPDATE SET name = EXCLUDED.name;