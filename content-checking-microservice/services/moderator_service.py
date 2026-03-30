import numpy as np
import torch
from transformers import AutoTokenizer, AutoModelForSequenceClassification
from typing import List, Dict, Any

from config import settings


class ModerationService:
    def __init__(self):
        self.tokenizer = None
        self.model = None
        self.device = torch.device("cpu")

    def load(self) -> "ModerationService":
        print(f"⏳ Загрузка модели из {settings.MODEL_PATH}...")

        tokenizer = AutoTokenizer.from_pretrained(settings.MODEL_PATH)
        model_fp32 = AutoModelForSequenceClassification.from_pretrained(settings.MODEL_PATH)

        # INT8 квантизация
        self.model = torch.quantization.quantize_dynamic(
            model_fp32,
            {torch.nn.Linear},
            dtype=torch.qint8,
        )

        self.tokenizer = tokenizer
        self.model.to(self.device)
        self.model.eval()

        size_mb = sum(p.numel() * p.element_size() for p in self.model.parameters()) / (1024 ** 2)
        print(f"✅ Модель загружена в INT8 (~{size_mb:.1f} MB) на {self.device}")
        return self

    def analyze_text(self, list_post: List[Dict[str, Any]]) -> List[Dict[str, Any]] | None:
        if not list_post:
            return None

        chunk_to_post_map = []
        all_chunks = []

        for idx, post in enumerate(list_post):
            tokens = self.tokenizer.encode(post['full_text'], add_special_tokens=True)
            for i in range(0, len(tokens), settings.CHUNK_STEP):
                chunk = tokens[i:i + settings.CHUNK_MAX_LEN]
                if len(chunk) < 2:
                    continue
                all_chunks.append(chunk)
                chunk_to_post_map.append(idx)

        if not all_chunks:
            return None

        batch_features = self.tokenizer.pad(
            {"input_ids": all_chunks},
            padding="max_length",
            max_length=settings.CHUNK_MAX_LEN,
            return_tensors="pt",
        ).to(self.device)

        with torch.no_grad():
            outputs = self.model(**batch_features)
            probs = torch.sigmoid(outputs.logits).cpu().numpy()

        toxic_part = probs[:, 1:]
        max_ids = np.argmax(toxic_part, axis=1) + 1
        max_vals = probs[np.arange(len(probs)), max_ids]

        mask = (max_vals > 0.5) & (probs[:, 0] > settings.CONFIDENCE_THRESHOLD)
        probs[np.arange(len(probs))[mask], max_ids[mask]] = 0

        post_aggregated_probs = [np.zeros(len(settings.LABELS)) for _ in range(len(list_post))]
        chunk_to_post_map = np.array(chunk_to_post_map)

        for post_idx in range(len(list_post)):
            mask = chunk_to_post_map == post_idx
            if np.any(mask):
                post_aggregated_probs[post_idx] = probs[mask].max(axis=0)

        final_results = []
        for idx, final_probs in enumerate(post_aggregated_probs):
            all_scores = {settings.LABELS[i]: round(float(final_probs[i]), 4) for i in range(len(settings.LABELS))}

            toxic_signals = final_probs[1:]
            max_idx = toxic_signals.argmax()
            max_score = toxic_signals[max_idx]

            if max_score > settings.TOXIC_THRESHOLD:
                verdict = settings.LABELS[max_idx + 1]
                score = float(max_score)
            else:
                verdict = "non-toxic"
                score = float(final_probs[0])

            final_results.append({
                "post_id": list_post[idx]["post_id"],
                "verdict": verdict,
                "confidence": score,
                "all_scores": all_scores,
                "is_flagged": verdict != "non-toxic"
            })

        return final_results
