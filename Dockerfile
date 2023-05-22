FROM gcr.io/distroless/static

# Copy the binary that goreleaser built
COPY tenant-api /tenant-api

# Run the web service on container startup.
ENTRYPOINT ["/tenant-api"]
CMD ["serve"]
