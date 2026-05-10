-- Тип самого сообщения
CREATE TYPE message_type AS ENUM ('standard', 'voice_note', 'video_note');

-- Тип вложения
CREATE TYPE attachment_type AS ENUM ('photo', 'video', 'voice_note', 'video_note');

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL, -- ID комнаты
    
    msg_type message_type NOT NULL,
    content TEXT,             -- Текст сообщения (может быть NULL)
    
    -- ID внешнего объекта (из мастерской).
    attached_object_workshop_id UUID,  
	-- ID внешнего объекта (публикация)
	attached_object_post_id UUID, 
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ    -- Soft delete (чтобы удаленные сообщения не ломали историю)
);

-- Индексы для быстрого поиска сообщений в канале
CREATE INDEX idx_messages_room_id ON messages(room_id, created_at DESC);

CREATE TABLE attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    
    att_type attachment_type NOT NULL,
    s3_key TEXT NOT NULL,     -- Ключ для S3 хранилища
    
    -- Метаданные
    file_size_bytes BIGINT DEFAULT 0,
    duration BIGINT,         -- Длительность (для голосовых и кружков)
        
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_attachments_message_id ON attachments(message_id, created_at);

ALTER TABLE attachments 
ADD COLUMN preview_s3_key TEXT; -- Ключ для превью (миниатюры) в S3