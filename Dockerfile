FROM golang:1.20.2 AS builder

COPY . /src
WORKDIR /src

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/tenant-api .

# pass in name as --build-arg
FROM gcr.io/distroless/static:nonroot
# `nonroot` coming from distroless
USER 65532:65532

COPY  --from=builder /src/bin/tenant-api /bin/tenant-api

# Run the web service on container startup.
ENTRYPOINT ["/bin/tenant-api"]
CMD ["serve"]
