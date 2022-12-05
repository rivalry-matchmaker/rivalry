FROM golang:1.19-bullseye as build
WORKDIR /app

COPY go.sum go.mod ./

RUN go mod download

COPY . .

ARG COMMAND
ARG LOCATION=cmd

RUN go build -o build/app ${LOCATION}/${COMMAND}/main.go

FROM gcr.io/distroless/base-debian11:nonroot

# Expose port 50051 for gRPC
EXPOSE 50051/tcp

COPY --chown=nonroot:nonroot --from=build /app/build/app /
ENTRYPOINT ["/app"]
