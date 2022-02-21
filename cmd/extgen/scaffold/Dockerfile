FROM golang:1.18rc1 as builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
# RUN go mod download

# Copy the go source
COPY . /workspace

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o {{ .Name }} ./cmd/{{ .Name }}

# Use distroless as minimal base image to package the {{ .Name }} binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/{{ .Name }} .
USER 65532:65532

ENTRYPOINT ["/{{ .Name }}"]
