#Stage 0
FROM golang:1.13-alpine3.10
# Add Maintainer Info
LABEL maintainer="Maksym Shkolnyi <maksymsh@wix.com>"
# Set the Current Working Directory inside the container
WORKDIR /build
# Copy go mod and sum files
COPY go.mod go.sum ./
# Download all dependencies. Depapk add build-baseendencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download
# Copy the source from the current directory to the Working Directory inside the container
COPY . .
# Install GCC
RUN apk add build-base
# Build the Go app
RUN go build -o tfChek .


#Stage 1
FROM bash:4.4
WORKDIR /application
LABEL maintainer="Maksym Shkolnyi <maksymsh@wix.com>"
RUN apk --no-cache add ca-certificates

#Add user
#NOTE: Using of /usr/local/bin/bash specially to support builtin function during the RVM sourcing
RUN addgroup --system deployer && adduser --system --ingroup deployer --uid 5500 -s /usr/local/bin/bash deployer

#Temporary workaround to fix broken GitHub Authentication
RUN apk add openssh
RUN mkdir /home/deployer/.ssh && chmod 700 /home/deployer/.ssh
COPY ssh_config /home/deployer/.ssh/config
COPY github_know_hosts /home/deployer/.ssh/known_hosts
RUN chown -R deployer:deployer /home/deployer/.ssh
RUN apk add git

#Install ruby
RUN apk add gnupg
RUN apk add curl
RUN apk add tar
#Very harmful line this must be somehow overcome. There should not be any compillers in the docker container
#TODO: Fix this. Remove compillers
RUN apk add musl
RUN apk add build-base
#Install ruby to the userspace
USER deployer
RUN       whoami && id && which bash && ps aux && echo "$$" && \
          gpg2 --recv-keys 409B6B1796C275462A1703113804BB82D39DC0E3 7D2BAF1CF37B13E2069D6956105BD0E739499BDB && \
          curl -sSL https://get.rvm.io | bash -s stable && \
          head -n 12 /home/deployer/.rvm/scripts/rvm && \
          bash -c 'source /home/deployer/.rvm/scripts/rvm && rvm install 2.6.5 && rvm --default use 2.6.5'
#          RUN       su deployer -c 'gpg2 --recv-keys 409B6B1796C275462A1703113804BB82D39DC0E3 7D2BAF1CF37B13E2069D6956105BD0E739499BDB'
#          RUN       su deployer -c '   curl -sSL https://get.rvm.io | bash -s stable '
#          RUN       su deployer -c '   source ~/.rvm/scripts/rvm '
#          RUN       su deployer -c '   rvm install 2.6.5 '
#          RUN       su deployer -c '   rvm --default use 2.6.5'
USER root

#Copy files
COPY --from=0 /build/tfChek .
COPY --from=0 /build/tfChek.sh .
COPY --from=0 /build/templates /templates
COPY --from=0 /build/static static
RUN chown -R deployer:deployer * && mkdir /var/tfChek && chown -R deployer:deployer /var/tfChek && mkdir /var/run/tfChek && chown -R deployer:deployer /var/run/tfChek
#Switch user
USER deployer

# Expose port 8080 to the outside world
EXPOSE 8085

# Command to run the executable
# I have to use enrtypoint to re-export PORT to TFCHEK_PORT variable
ENTRYPOINT [ "./tfChek.sh" ]

