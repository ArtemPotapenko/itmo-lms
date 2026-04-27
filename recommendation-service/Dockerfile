FROM python:3.12-alpine

WORKDIR /app
COPY . /app
RUN pip install --no-cache-dir -r /app/requirements.txt

EXPOSE 8091

ENTRYPOINT ["python3", "/app/cmd/recommendation-service/main.py"]
