FROM python:3.12-alpine

WORKDIR /app
COPY . /app

EXPOSE 8091

ENTRYPOINT ["python3", "/app/cmd/recommendation-service/main.py"]
