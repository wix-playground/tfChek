# syntax=docker/dockerfile:experimental

#Stage 0
FROM golang:1.14.4-alpine3.12

# Add Maintainer Info
LABEL maintainer="Maksym Shkolnyi <maksymsh@wix.com>"
# Set the Current Working Directory inside the container
WORKDIR /build
# Copy go mod and sum files
COPY go.mod go.sum ./
#Add git to be able to download dependencies form a private repositories
RUN apk add git

# Install ssh client and git
RUN apk add --no-cache openssh-client
# Download public key for github.com
RUN mkdir -p -m 0600 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts
# Update git config
RUN git config --global url."ssh://git@github.com/".insteadOf "https://github.com/"
## Clone private repository
#RUN --mount=type=ssh git clone git@github.com:wix-system/tfResDif.git wtf

# Download all dependencies. Depapk add build-baseendencies will be cached if the go.mod and go.sum files are not changed
ENV GOPRIVATE=github.com/wix-system
RUN --mount=type=ssh go mod download
# Copy the source from the current directory to the Working Directory inside the container
COPY . .
# Install GCC
RUN apk add build-base

#Mount GitHub token
#RUN --mount=type=secret,id=github cat /run/secrets/github
# Build the Go app
RUN go build -o tfChek .

#Stage 1
FROM bash:4.4.23
WORKDIR /application
LABEL maintainer="Maksym Shkolnyi <maksymsh@wix.com>"
RUN apk --no-cache add ca-certificates

#Add user
RUN addgroup --system deployer && adduser --system --ingroup deployer --uid 5500 deployer

#Temporary workaround to fix broken GitHub Authentication
RUN apk add openssh
RUN mkdir /home/deployer/.ssh && chmod 700 /home/deployer/.ssh
RUN mkdir /home/deployer/.chef && chmod 770 /home/deployer/.chef
COPY luggage/ssh_config /home/deployer/.ssh/config
COPY luggage/github_know_hosts /home/deployer/.ssh/known_hosts
RUN chown -R deployer:deployer /home/deployer/.ssh
RUN apk add git

#Configure AWS access for terraform
RUN mkdir /home/deployer/.aws && chown deployer:deployer /home/deployer/.aws
COPY --chown=deployer:deployer luggage/aws_config /home/deployer/.aws/config

#Install ruby
RUN apk add ruby=2.7.1-r3 ruby-dev=2.7.1-r3
RUN gem install bundler -v 2.1.4
RUN gem install netaddr -v 2.0.4
RUN gem install colorize  zip
#Graphvis need building tools
RUN apk add build-base && \
gem install json -v 2.3.0 && \
gem install ffi -v 1.13.1 && \
gem install process-terminal -v 0.2.0 && \
gem install process-group -v 1.2.3 && \
gem install process-pipeline -v 1.0.2 && \
gem install graphviz -v 1.2.0 && \
apk del build-base
#Install bash dependencies
RUN apk add ncurses curl zip

#Temporary layer (DELETE AFTER OCTOBER 2020)
#Add key for star wixpress until 2020 certificate key
RUN mkdir /home/deployer/luggage && chown deployer:deployer /home/deployer/luggage
COPY  --chown=deployer:deployer /luggage/star_wixpress_com_until_2020.* /home/deployer/luggage/
#End of temporary layer (DELETE AFTER OCTOBER 2020)

#Copy files
COPY  --chown=deployer:deployer --from=0 /build/tfChek .
COPY  --chown=deployer:deployer --from=0 /build/luggage/tfChek.sh .
COPY  --chown=deployer:deployer --from=0 /build/templates /templates
COPY  --chown=deployer:deployer --from=0 /build/static static
RUN chown -R deployer:deployer * && mkdir -p /var/tfChek/out && chown -R deployer:deployer /var/tfChek && mkdir /var/run/tfChek && chown -R deployer:deployer /var/run/tfChek
#Switch user
USER deployer

# Expose port 8080 to the outside world
EXPOSE 8085


# Command to run the executable
# I have to use enrtypoint to re-export PORT to TFCHEK_PORT variable
ENTRYPOINT [ "./tfChek.sh" ]


