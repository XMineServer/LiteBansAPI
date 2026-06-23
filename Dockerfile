FROM golang:1.26.1-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/litebans-api ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/litebans-api /litebans-api

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/litebans-api"]
