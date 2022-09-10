FROM golang:1.19 as builder

RUN apk add --no-cache git

WORKDIR /workspace

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy the go source
COPY . /workspace

# Build
RUN CGO_ENABLED=0 go build -a -o {{ .Name }} ./cmd/{{ .Name }}

FROM alpine
WORKDIR /
COPY --from=builder /workspace/{{ .Name }} .
USER 65532:65532

ENTRYPOINT ["/{{ .Name }}"]
