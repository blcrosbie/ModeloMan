FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/modeloman-server ./cmd/modeloman-server

FROM gcr.io/distroless/static-debian12

WORKDIR /
COPY --from=build /out/modeloman-server /modeloman-server

ENV GRPC_ADDR=:50051
ENV DATA_FILE=/data/modeloman.db.json

VOLUME ["/data"]
EXPOSE 50051

ENTRYPOINT ["/modeloman-server"]
