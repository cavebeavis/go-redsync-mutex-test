###################################################################################################################################
# ********************************************************** GLOBAL ARGS ******************************************************** #
# ******************************************************************************************************************************* #
#                                                                                                                                 #
# These are to be used through both Build and Final stages.                                                                       #
#                                                                                                                                 #
###################################################################################################################################
ARG USER=app
ARG UID=4242
ARG GID=2424

# Set defaults on the Arguments but these should be passed into the docker build
# using the --build-arg <varname>=<value> flag.
ARG APP_NAME="redsync-test"
ARG VERSION="v0.0.0"
ARG ENV="dev"
ARG REDIS_ADDR="0.0.0.0:6379"

###################################################################################################################################
# ******************************************************** BUILD CONTAINER ****************************************************** #
# ******************************************************************************************************************************* #
#                                                                                                                                 #
# In order to build this from the repos root directory (i.e. ../../):                                                             #
#                                                                                                                                 #
#          $ docker build -f ./redsync/go.Dockerfile \                                                                            #
#            --build-arg APP_NAME=redsync-test \                                                                                  #
#            --build-arg VERSION=v0.0.1 \                                                                                         #
#            --build-arg ENV=dev \                                                                                                #
#            -t go-redsync-test \                                                                                                 #
#            .                                                                                                                    #
#                                                                                                                                 #
# To manually run the docker image:                                                                                               #
#                                                                                                                                 #
#          $ docker run go-redsync-test                                                                                           # 
#                                                                                                                                 #
###################################################################################################################################
FROM golang:1.17-alpine AS builder

# Repeating these according to: https://stackoverflow.com/a/53682110
ARG USER
ARG UID
ARG GID
ARG APP_NAME
ARG VERSION
ARG ENV
ARG REDIS_ADDR

RUN apk update && \
    apk upgrade && \
    apk add build-base

WORKDIR /go/src/app

# Context is set as "../" in docker-compose file.
COPY ./redsync/ ./cmd
COPY ./go.mod .
COPY ./go.sum .

RUN go mod tidy

# https://www.digitalocean.com/community/tutorials/using-ldflags-to-set-version-information-for-go-applications
RUN CGO_ENABLED=0 GOOS=linux \ 
    go build \
    -ldflags="-X 'main.appName=$APP_NAME' -X 'main.version=$VERSION' -X 'main.env=$ENV' -X 'main.redisAddr=$REDIS_ADDR'" \
    -o "$APP_NAME" ./cmd/main.go

# https://stackoverflow.com/a/55757473
RUN addgroup \
    --gid "$GID" \
    "$USER" && \
    adduser \
    --disabled-password \
    --gecos "" \
    --ingroup "$USER" \
    --no-create-home \
    --uid "$UID" \
    "$USER"

RUN chown -R "$USER":"$USER" /go/src/app && \
    chmod +x "$APP_NAME"


###################################################################################################################################
# ******************************************************** FINAL CONTAINER ****************************************************** #
# ******************************************************************************************************************************* #
###################################################################################################################################
FROM alpine:latest

# Repeating these according to: https://stackoverflow.com/a/53682110
ARG USER
ARG UID
ARG GID
ARG APP_NAME
ARG VERSION

# This is because ARG are only applicable at build time. See https://stackoverflow.com/a/35562189.
ENV APP_NAME "$APP_NAME"

WORKDIR /home/app

# https://stackoverflow.com/a/55757473
RUN apk update && \
    apk upgrade && \
    addgroup \
    --gid "$GID"\
    "$USER" && \
    adduser \
    --disabled-password \
    --gecos "" \
    --home "$(pwd)" \
    --ingroup "$USER" \
    --no-create-home \
    --uid "$UID" \
    "$USER"

RUN groups

USER "$USER"

COPY --from=builder /go/src/app/"$APP_NAME" .

ENTRYPOINT ./$APP_NAME