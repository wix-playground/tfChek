# syntax=docker/dockerfile:experimental

#Stage 0
FROM golang:1.14.4-alpine3.12

#Add user argument
ARG ACCESS_TOKEN_USR="nothing"
ARG ACCESS_TOKEN_PWD="nothing"

# Add Maintainer Info
LABEL maintainer="Maksym Shkolnyi <maksymsh@wix.com>"
# Set the Current Working Directory inside the container
WORKDIR /build
# Copy go mod and sum files
COPY go.mod go.sum ./
#Add git to be able to download dependencies form a private repositories
RUN apk add git

# Create a netrc file using the credentials specified using --build-arg
RUN --mount=type=secret,id=github printf "machine github.com\n\
    login ${ACCESS_TOKEN_USR}\n\
    password $(cat /run/secrets/github)\n\
    \n\
    machine api.github.com\n\
    login ${ACCESS_TOKEN_USR}\n\
    password $(cat /run/secrets/github)\n"\
    >> /root/.netrc
RUN chmod 600 /root/.netrc

#RUN git config --global url.ssh://git@github.com/.insteadof=https://github.com/
CMD /bin/sh -c 'sleep 5000'
## Download all dependencies. Depapk add build-baseendencies will be cached if the go.mod and go.sum files are not changed
#RUN go mod download
## Copy the source from the current directory to the Working Directory inside the container
#COPY . .
## Install GCC
#RUN apk add build-base
#
##Mount GitHub token
##RUN --mount=type=secret,id=github cat /run/secrets/github
## Build the Go app
#RUN go build -o tfChek .

