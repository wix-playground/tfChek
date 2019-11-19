#Stage 0
FROM golang:1.13-alpine3.10
# Add Maintainer Info
LABEL maintainer="Maksym Shkolnyi <maksymsh@wix.com>"
# Set the Current Working Directory inside the container
WORKDIR /build
# Copy go mod and sum files
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download
# Copy the source from the current directory to the Working Directory inside the container
COPY . .
#RUN mkdir configs
#COPY tfChek.yml configs/
# Build the Go app
RUN go build -o tfChek .


#Stage 1
FROM bash:5
WORKDIR /application
LABEL maintainer="Maksym Shkolnyi <maksymsh@wix.com>"
RUN apk --no-cache add ca-certificates
#RUN mkdir /configs && chown 5500 /configs

#Add user
RUN addgroup --system deployer && adduser --system --ingroup deployer --uid 5500 deployer

#Copy files
COPY --from=0 /build/tfChek .
COPY --from=0 /build/tfChek.sh .
#COPY --from=0 /build/tfChek.yml .
COPY --from=0 /build/templates /templates
COPY --from=0 /build/static static
RUN chown -R deployer:deployer * && mkdir /var/tfChek && chown -R deployer:deployer /var/tfChek
#Switch user
USER deployer
# Expose port 8080 to the outside world
EXPOSE 8085

# Command to run the executable
#CMD ["./tfChek"]
ENTRYPOINT [ "./tfChek.sh" ]

