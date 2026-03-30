import asyncio
from contextlib import asynccontextmanager

from fastapi import FastAPI

from config import settings
from services.moderator_service import ModerationService
from repositories.post_repository import PostRepository
from workers.redis_worker import redis_worker
import redis.asyncio as asyncio_redis


@asynccontextmanager
async def lifespan(app: FastAPI):
    print("Приложение запускается...")

    redis_client = asyncio_redis.from_url(
        settings.REDIS_URL, decode_responses=True, max_connections=10
    )

    moderation_service = ModerationService().load()
    repo = PostRepository()

    app.state.moderation_service = moderation_service
    app.state.repo = repo
    app.state.redis_client = redis_client

    worker_task = asyncio.create_task(
        redis_worker(moderation_service, repo, redis_client)
    )

    yield

    print("Выключение приложения...")
    worker_task.cancel()
    try:
        await worker_task
    except asyncio.CancelledError:
        pass
    await redis_client.aclose()
    await repo.engine.dispose()


app = FastAPI(lifespan=lifespan)


@app.get("/")
async def read_root():
    return {
        "status": "worker_running",
        "model": "rubert-tiny-toxicity (INT8)",
        "batch_size": settings.MAX_BATCH_SIZE,
    }


@app.post("/analyze")
async def manual_analyze(posts: list[dict]):
    return app.state.moderation_service.analyze_text(posts)