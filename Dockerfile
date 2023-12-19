FROM golang:1.21.5 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM scratch
COPY --from=builder /src/main /app/main
WORKDIR /app
ENTRYPOINT [ "/app/main" ]