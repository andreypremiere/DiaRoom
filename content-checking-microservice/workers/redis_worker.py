import asyncio
import time
from typing import List, Dict, Any

import redis.asyncio as asyncio_redis

from config import settings
from services.moderator_service import ModerationService
from repositories.post_repository import PostRepository


async def redis_worker(
    moderation_service: ModerationService,
    repo: PostRepository,
    redis_client: asyncio_redis.Redis,
):
    print("🚀 Worker started: watching queue...", flush=True)
    cache_list: List[Dict[str, Any]] = []
    last_batch_time = time.time()

    while True:
        try:
            raw_data = await redis_client.brpop(settings.QUEUE_NAME, timeout=1)
            if raw_data:
                post_id = raw_data[1]
                if isinstance(post_id, bytes):
                    post_id = post_id.decode("utf-8")

                payload = await repo.get_canvas_payload_by_post(post_id)
                if payload:
                    full_text = " ".join(
                        block.get("text", "").strip()
                        for block in payload
                        if block.get("type") == "text"
                    )
                    cache_list.append({"post_id": post_id, "full_text": full_text})

            current_time = time.time()
            if len(cache_list) >= settings.MAX_BATCH_SIZE or (
                current_time - last_batch_time > settings.BATCH_TIMEOUT and cache_list
            ):
                print(f"📦 Обработка батча: {len(cache_list)} постов...")
                start = time.perf_counter()
                results = moderation_service.analyze_text(cache_list)
                duration = time.perf_counter() - start
                print(f"⏱️ analyze_text выполнен за {duration:.4f} сек.")

                if results:
                    await repo.update_posts(results)

                cache_list.clear()
                last_batch_time = time.time()

        except Exception as e:
            print(f"❌ Worker error: {e}")
            await asyncio.sleep(5)