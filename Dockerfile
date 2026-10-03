FROM golang:1.27-alpine AS build
WORKDIR /src

RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/server ./server

EXPOSE 3000
USER nonroot:nonroot
ENTRYPOINT ["./server"]
