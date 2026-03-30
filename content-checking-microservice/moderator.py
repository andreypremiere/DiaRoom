import torch
from transformers import AutoTokenizer, AutoModelForSequenceClassification
import numpy as np

# Путь к модели на Hugging Face
MODEL_NAME = "cointegrated/rubert-tiny-toxicity"

# Загружаем один раз при импорте модуля
print("⏳ Loading AI model...", flush=True)
tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME)
model = AutoModelForSequenceClassification.from_pretrained(MODEL_NAME)

# Если есть возможность, переносим на GPU (в Docker обычно CPU)
device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
model.to(device)
print(f"✅ Model loaded on {device}", flush=True)


def analyze_long_text(text: str):
    if not text or not text.strip():
        return {"label": "non-toxic", "score": 0.0, "all_scores": {}}

    # 1. Токенизируем текст БЕЗ обрезки (truncation=False)
    # Это позволит нам увидеть все токены текста, какими бы длинными они ни были
    full_inputs = tokenizer(text, return_tensors="pt", add_special_tokens=True)
    input_ids = full_inputs['input_ids'][0]

    total_tokens = len(input_ids)
    max_len = 512
    step = 400  # Делаем небольшой нахлест (overlap), чтобы не потерять контекст на стыке кусков

    all_chunks_results = []
    labels = ["non-toxic", "insult", "obscenity", "threat", "dangerous"]

    # 2. Проходимся по тексту "окном" в 512 токенов
    for i in range(0, total_tokens, step):
        chunk_ids = input_ids[i: i + max_len].unsqueeze(0).to(device)
        # Если последний кусок слишком короткий (например, 1 токен), пропускаем
        if chunk_ids.size(1) < 5:
            continue

        with torch.no_grad():
            logits = model(chunk_ids).logits
            probs = torch.sigmoid(logits).cpu().numpy()[0]

        all_chunks_results.append(probs)

        # Если текст очень длинный, не проверяем больше 10 чанков (защита от зависания)
        if len(all_chunks_results) >= 10:
            break

    # 3. Агрегируем результаты: берем МАКСИМАЛЬНУЮ токсичность из всех кусков
    # Это логика "нашел грязь — баню всё"
    final_probs = np.max(all_chunks_results, axis=0)

    all_scores = {labels[i]: round(float(final_probs[i]), 4) for i in range(len(labels))}

    print(f"📖 Проверено чанков: {len(all_chunks_results)} (Всего токенов: {total_tokens})")
    print(f"📊 Итоговые макс-скоры: {all_scores}", flush=True)

    # 4. Финальный вердикт (используем наш агрессивный порог 0.5)
    toxic_signals = final_probs[1:]
    max_idx = toxic_signals.argmax()
    max_score = toxic_signals[max_idx]

    if max_score > 0.5:
        return {
            "label": labels[max_idx + 1],
            "score": float(max_score),
            "all_scores": all_scores
        }

    return {"label": "non-toxic", "score": float(final_probs[0]), "all_scores": all_scores}