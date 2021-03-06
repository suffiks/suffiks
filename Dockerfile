# Build the manager binary
FROM golang:1.18.1 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download && go build std

# Copy the go source
COPY embeds.go embeds.go
COPY cmd/suffiks cmd/suffiks
COPY api/ api/
COPY base/ base/
COPY controllers/ controllers/
COPY docparser/ docparser/
COPY docs/ docs/
COPY extension/ extension/
COPY config/crd/bases/ config/crd/bases/
# Build
RUN CGO_ENABLED=0 go build -a -o manager ./cmd/suffiks/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
