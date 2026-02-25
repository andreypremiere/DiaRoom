CREATE TABLE IF NOT EXISTS rooms (
    -- Основной ID записи
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Ссылка на пользователя (UUID), который владеет комнатой или привязан к ней
    user_id UUID NOT NULL,
    
    -- Названия
    room_name VARCHAR(100) NOT NULL,
    room_name_id VARCHAR(70) NOT NULL UNIQUE, -- Уникальный короткий адрес (например, @my_room)
    
    -- Контент
    avatar_url TEXT,
    bio TEXT,
    
    -- ID настроения (можно потом сделать отдельную таблицу, пока просто UUID или INT)
    mood_id UUID,
    
    -- Сложные типы данных
    hashtags JSONB DEFAULT '[]',   -- Список тегов, например: ["go", "docker", "web"]
    settings JSONB DEFAULT '{}',   -- Настройки комнаты: {"private": true, "theme": "dark"}
    
    -- Счетчики (обязательно NOT NULL и дефолт 0)
    followers_count INT NOT NULL DEFAULT 0,
    following_count INT NOT NULL DEFAULT 0,
    
    -- Время создания
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Индекс для быстрого поиска по room_id_name (например, для поиска через @название)
CREATE INDEX IF NOT EXISTS idx_room_id_name ON rooms(room_name_id);