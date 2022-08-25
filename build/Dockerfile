FROM --platform=$BUILDPLATFORM golang:1.19 AS builder
LABEL maintainer="<support@quizinch.com>"
WORKDIR /quiz
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build app, cross-platform
#  - rebuild all packages (-a)
ARG TARGETOS TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -a -installsuffix cgo -o quiz ./cmd/web

# New container for target platform, with certificates and ffmpeg
FROM --platform=$TARGETPLATFORM alpine
RUN apk --no-cache add ca-certificates ffmpeg

# Copy image from previous stage
COPY --from=builder /quiz/quiz /app/

# Startup command
CMD ["/app/quiz"]