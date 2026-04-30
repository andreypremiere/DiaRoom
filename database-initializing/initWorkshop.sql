CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- тип ENUM для статуса загрузки
CREATE TYPE item_status AS ENUM ('uploading', 'ready', 'failed');
CREATE TYPE type_item AS ENUM ('photo', 'video', 'canvas');

-- Таблица папок
CREATE TABLE IF NOT EXISTS folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL,
    parent_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Индексы для папок
CREATE INDEX idx_folders_room_id ON folders(room_id);
CREATE INDEX idx_folders_parent_id ON folders(parent_id);

-- Таблица значений (файлов/контента)
CREATE TABLE IF NOT EXISTS items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL,
    folder_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    preview_url TEXT DEFAULT '',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    item_type type_item NOT NULL,
    status item_status DEFAULT 'uploading',
    payload JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Индексы для значений
CREATE INDEX idx_items_room_id ON items(room_id);
CREATE INDEX idx_items_folder_id ON items(folder_id);

-- 5. Таблица статистики (квоты)
CREATE TABLE IF NOT EXISTS room_storage_stats (
    room_id UUID PRIMARY KEY,
    total_size BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

--- ТРИГГЕРЫ ДЛЯ ОБНОВЛЕНИЯ СТАТИСТИКИ ХРАНИЛИЩА

-- Функция триггера
CREATE OR REPLACE FUNCTION fn_update_room_storage_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- Случай INSERT: Добавляем размер к комнате
    IF (TG_OP = 'INSERT') THEN
        INSERT INTO room_storage_stats (room_id, total_size, updated_at)
        VALUES (NEW.room_id, NEW.size_bytes, NOW())
        ON CONFLICT (room_id) DO UPDATE 
        SET total_size = room_storage_stats.total_size + EXCLUDED.total_size,
            updated_at = NOW();
            
    -- Случай UPDATE: Если изменился размер файла
    ELSIF (TG_OP = 'UPDATE') THEN
        IF (OLD.size_bytes <> NEW.size_bytes) THEN
            UPDATE room_storage_stats 
            SET total_size = total_size - OLD.size_bytes + NEW.size_bytes,
                updated_at = NOW()
            WHERE room_id = NEW.room_id;
        END IF;

    -- Случай DELETE: Вычитаем размер
    ELSIF (TG_OP = 'DELETE') THEN
        UPDATE room_storage_stats 
        SET total_size = total_size - OLD.size_bytes,
            updated_at = NOW()
        WHERE room_id = OLD.room_id;
    END IF;
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Создание триггера
CREATE TRIGGER trg_items_storage_stats
AFTER INSERT OR UPDATE OR DELETE ON items
FOR EACH ROW EXECUTE FUNCTION fn_update_room_storage_stats();