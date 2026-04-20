FROM golang:1.25-alpine AS build
WORKDIR /src
COPY . .
ARG SERVICE
RUN go build -o /out/service ./${SERVICE}/cmd/${SERVICE}

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/service /app/service
EXPOSE 8080
ENTRYPOINT ["/app/service"]
