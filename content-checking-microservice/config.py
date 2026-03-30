from typing import ClassVar


class Settings:
    MODEL_PATH: str = "/app/models/rubert-tiny-toxicity"

    DATABASE_URL: str = "postgresql+asyncpg://postgres:1111@postgresql-posts:5432/db_posts"
    REDIS_URL: str = "redis://redis-queue:6379/0"

    QUEUE_NAME: str = "new_posts:post_id"

    BATCH_TIMEOUT: float = 0.2
    MAX_BATCH_SIZE: int = 32

    LABELS: ClassVar[list[str]] = ["non-toxic", "insult", "obscenity", "threat", "dangerous"]
    TOXIC_THRESHOLD: float = 0.45
    CONFIDENCE_THRESHOLD: float = 0.98
    CHUNK_MAX_LEN: int = 512
    CHUNK_STEP: int = 400


settings = Settings()
