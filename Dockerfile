# Imagen mínima de runtime — la compilación ocurre en el CI (manual-release.yml
# Job 2) y el binario se pasa como contexto de build, eliminando toda compilación
# Go dentro del contenedor (reduce tiempo de imagen de ~12-18 min a ~1 min).
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY main .

RUN chmod +x ./main

EXPOSE 8080

CMD ["./main"]
