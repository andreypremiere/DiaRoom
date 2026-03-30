import torch
from transformers import AutoTokenizer, AutoModelForSequenceClassification
import numpy as np

MODEL_NAME = "cointegrated/rubert-tiny-toxicity"

print("⏳ Loading AI model...", flush=True)
tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME, use_fast=True)
model = AutoModelForSequenceClassification.from_pretrained(MODEL_NAME)

device = torch.device("cpu")
model.to(device)
print(f"✅ Model loaded on {device}", flush=True)


def analyze_text(list_post: list):
    chunk_to_post_map = []
    all_chunks = []
    max_len = 512
    step = 400
    labels = ["non-toxic", "insult", "obscenity", "threat", "dangerous"]
    treshold = 0.5
    treshold_confidence = 0.98

    for idx, post in enumerate(list_post):
        tokens = tokenizer.encode(post['full_text'], add_special_tokens=True)
        total_tokens = len(tokens)

        for i in range(0, total_tokens, step):
            chunk = tokens[i: i + max_len]
            if len(chunk) < 2:
                continue

            all_chunks.append(chunk)
            chunk_to_post_map.append(idx)

    if not all_chunks:
        return None

    batch_features = tokenizer.pad(
        {"input_ids": all_chunks},
        padding='max_length',
        max_length=max_len,
        return_tensors="pt",
    ).to(device)

    with torch.no_grad():
        outputs = model(**batch_features)
        logits = outputs.logits
        probs = torch.sigmoid(logits).cpu().numpy()

    toxic_part = probs[:, 1:]
    max_ids = np.argmax(toxic_part, axis=1) + 1
    max_vals = probs[np.arange(len(probs)), max_ids]

    mask = (max_vals > treshold) & (probs[:, 0] > treshold_confidence)

    probs[np.arange(len(probs))[mask], max_ids[mask]] = 0

    post_aggregated_probs = [np.zeros(len(labels)) for _ in range(len(list_post))]

    chunk_to_post_map = np.array(chunk_to_post_map)

    for post_idx in range(len(list_post)):
        mask = chunk_to_post_map == post_idx
        if np.any(mask):
            post_aggregated_probs[post_idx] = probs[mask].max(axis=0)

    final_results = []

    for idx, final_probs in enumerate(post_aggregated_probs):
        all_scores = {labels[i]: round(float(final_probs[i]), 4) for i in range(len(labels))}

        toxic_signals = final_probs[1:]
        max_idx = toxic_signals.argmax()
        max_score = toxic_signals[max_idx]

        threshold = 0.45

        if max_score > threshold:
            verdict_label = labels[max_idx + 1]
            final_score = float(max_score)
        else:
            verdict_label = "non-toxic"
            final_score = float(final_probs[0])

        final_results.append({
            "post_id": list_post[idx]['post_id'],
            "verdict": verdict_label,
            "confidence": final_score,
            "all_scores": all_scores,
            "is_flagged": verdict_label != "non-toxic"
        })

    return final_results







