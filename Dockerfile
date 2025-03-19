FROM golang:1.24.1-alpine as dev

ENV ROOT=/go/src/app
ENV CGO_ENABLED 0
WORKDIR ${ROOT}/cmd

RUN apk update && apk add git
COPY go.mod go.sum ./
COPY . .


RUN --mount=type=secret,id=GITHUB_TOKEN \
    echo "machine github.com login $(cat /run/secrets/GITHUB_TOKEN) password x-oauth-basic" > ~/.netrc
RUN go mod tidy

CMD ["go", "run", "./cmd/main.go"]


FROM golang:1.24 as builder

ENV ROOT=/go/src/app
WORKDIR ${ROOT}/cmd

RUN apt update && apt install -y git
COPY go.mod go.sum ./
COPY . .

RUN --mount=type=secret,id=GITHUB_TOKEN \
    echo "machine github.com login $(cat /run/secrets/GITHUB_TOKEN) password x-oauth-basic" > ~/.netrc
RUN go mod tidy

COPY . ${ROOT}
RUN CGO_ENABLED=0 go build -o $ROOT/binary

FROM gcr.io/distroless/base as prod

ENV ROOT=/go/src/app
WORKDIR ${ROOT}
COPY --from=builder --chown=nonroot:nonroot ${ROOT}/binary ${ROOT}

ENTRYPOINT ["/go/src/app/binary"]