from sqlalchemy import text
from database import engine


def get_canvas_payload_by_post(post_id: str):
    """
    Получает payload холста, связанного с конкретным постом.
    """
    with engine.connect() as conn:
        # Один SQL-запрос с JOIN будет быстрее, чем два отдельных
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