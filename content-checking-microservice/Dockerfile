FROM python:3.12-slim

WORKDIR /app

RUN apt-get update && apt-get install -y libpq-dev gcc && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
# Сначала обновляем pip
RUN pip install --no-cache-dir --upgrade pip

# Ставим Torch отдельно, явно указывая индекс для CPU
RUN pip install --no-cache-dir torch --index-url https://download.pytorch.org/whl/cpu

# Ставим всё остальное из requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]