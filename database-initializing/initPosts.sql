-- 1. Справочник категорий
CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(50) NOT NULL UNIQUE, 
    name VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Таблица холстов (Canvases)
CREATE TABLE IF NOT EXISTS canvases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. Основная таблица постов
CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL,
    canvas_id UUID NOT NULL,
    category_id INTEGER NOT NULL,

	is_published BOOLEAN NOT NULL DEFAULT FALSE,
	
    title VARCHAR(160),
    preview_url TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Внешние ключи
    CONSTRAINT fk_post_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT,
    CONSTRAINT fk_post_canvas   FOREIGN KEY (canvas_id)   REFERENCES canvases(id)   ON DELETE CASCADE
);

-- 4. Хештеги
CREATE TABLE IF NOT EXISTS hashtags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(80) NOT NULL UNIQUE,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 5. Связи Постов и Хештегов
CREATE TABLE IF NOT EXISTS posts_hashtags (
    post_id UUID NOT NULL,
    hashtag_id INTEGER NOT NULL,
    PRIMARY KEY (post_id, hashtag_id),
    CONSTRAINT fk_ph_post FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_ph_hashtag FOREIGN KEY (hashtag_id) REFERENCES hashtags(id) ON DELETE CASCADE
);

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