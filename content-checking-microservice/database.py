from sqlalchemy import create_engine
import redis

DATABASE_URL = "postgresql://postgres:1111@postgresql-posts:5432/db_posts"
REDIS_URL = "redis://redis-queue:6379/0"

# Настройка SQLAlchemy
engine = create_engine(DATABASE_URL, pool_size=5, max_overflow=10)

# Настройка Redis
redis_client = redis.from_url(REDIS_URL, decode_responses=True)
