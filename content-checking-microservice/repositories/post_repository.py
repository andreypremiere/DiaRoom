from typing import List, Dict, Any
from sqlalchemy import text
from sqlalchemy.ext.asyncio import create_async_engine

from config import settings


class PostRepository:
    def __init__(self):
        self.engine = create_async_engine(settings.DATABASE_URL)

    async def get_canvas_payload_by_post(self, post_id: str) -> list | None:
        try:
            async with self.engine.connect() as conn:
                query = text("""
                    SELECT c.payload
                    FROM canvases c
                    JOIN posts p ON p.canvas_id = c.id
                    WHERE p.id = CAST(:post_id AS UUID)
                """)
                result = await conn.execute(query, {"post_id": post_id})
                row = result.mappings().one_or_none()
                return row["payload"] if row else None
        except Exception as e:
            print(f"Ошибка получения payload для {post_id}: {e}")
            return None

    async def update_posts(self, results: List[Dict[str, Any]]):
        if not results:
            return

        from datetime import datetime, timezone
        now = datetime.now(timezone.utc)

        data_to_update = [
            {
                "status": "failed" if res["is_flagged"] else "passed",
                "post_status": "rejected" if res["is_flagged"] else "published",
                "now": now,
                "reason": res["verdict"] if res["is_flagged"] else None,
                "post_id": res["post_id"],
                "published_at": None if res["is_flagged"] else now
            }
            for res in results
        ]

        try:
            async with self.engine.begin() as conn:
                query = text("""
                    UPDATE posts
                    SET ai_check_status = :status,
                        ai_check_at = :now,
                        status = :post_status,
                        ai_check_reason = :reason,
                        updated_at = :now,
                        published_at = :published_at
                    WHERE id = :post_id
                """)
                await conn.execute(query, data_to_update)
            print(f"БД обновлена: {len(results)} постов.")
        except Exception as e:
            print(f"Ошибка обновления БД: {e}")