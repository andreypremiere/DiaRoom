-- Таблица комнат (НУЖНО ПЕРЕСОЗДАТЬ)
CREATE TABLE IF NOT EXISTS rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,

    is_configured BOOLEAN DEFAULT FALSE,
    
    room_name VARCHAR(100) NOT NULL,
    room_unique_id VARCHAR(100) NOT NULL UNIQUE, 
    
    avatar_url TEXT,
    background_url TEXT,
    bio TEXT,
    
    settings JSONB NOT NULL DEFAULT '{}',
    
    followers_count INT NOT NULL DEFAULT 0,
    following_count INT NOT NULL DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_room_unique_id ON rooms(room_unique_id);
CREATE INDEX IF NOT EXISTS idx_user_id ON rooms(user_id);
CREATE INDEX IF NOT EXISTS idx_rooms_unique_id_lower ON rooms (LOWER(room_unique_id));

-- Таблица категорий
CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(50) NOT NULL UNIQUE, 
    name VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Связующая таблица (Исправленная)
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

-- Индекс для обратного поиска (найти все комнаты в категории)
CREATE INDEX IF NOT EXISTS idx_room_categories_cat_id ON room_categories(category_id);

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
    -- ('music', 'Музыка'),
    -- ('sound-design', 'Саунд-дизайн'),
    -- ('podcasts', 'Подкасты'),
    ('literature', 'Литература и Статьи'),
    -- ('gamedev', 'Игры'),
    -- ('it-tech', 'Код и Технологии'),
    ('fashion', 'Мода и Стиль'),
    ('architecture-interior', 'Архитектура и Интерьер'),
    ('craft-diy', 'Крафт и DIY')
ON CONFLICT (slug) 
DO UPDATE SET name = EXCLUDED.name;
