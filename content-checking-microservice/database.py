from sqlalchemy import create_engine
import redis.asyncio as asyncio_redis
from sqlalchemy.ext.asyncio import create_async_engine

DATABASE_URL = "postgresql://postgres:1111@postgresql-posts:5432/db_posts"
REDIS_URL = "redis://redis-queue:6379/0"

# Настройка SQLAlchemy
def get_canvas_payload_by_post(post_id: str):
    """
    Получает payload холста, связанного с конкретным постом.
    """
    with engine.connect() as conn:
        query = text("""
            SELECT c.payload 
            FROM canvases c
            JOIN posts p ON p.canvas_id = c.id
            WHERE p.id = :post_id
        """)

        result = conn.execute(query, {"post_id": post_id}).mappings().one_or_none()

        if result:
            return result['payload']
        return None


async_engine = create_async_engine("postgresql+asyncpg://postgres:1111@postgresql-posts:5432/db_posts")

# Используем асинхронный метод from_url
redis_client = asyncio_redis.from_url(
    REDIS_URL,
    decode_responses=True,
    max_connections=10
)