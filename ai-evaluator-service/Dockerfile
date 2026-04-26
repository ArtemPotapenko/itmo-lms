FROM python:3.12-alpine

WORKDIR /app
COPY . /app

EXPOSE 8090

ENTRYPOINT ["python3", "/app/cmd/ai-evaluator-service/main.py"]
