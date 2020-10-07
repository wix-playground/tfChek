# syntax=docker/dockerfile:experimental

#Stage 1
FROM debian:stable-slim
WORKDIR /application
LABEL maintainer="Maksym Shkolnyi <maksymsh@wix.com>"

RUN apt update
RUN apt -y install  ca-certificates && update-ca-certificates
RUN apt -y install build-essential git gnupg2 curl ncurses-bin zip procps openssh-client
#Install RVM
RUN gpg2 --recv-keys 409B6B1796C275462A1703113804BB82D39DC0E3 7D2BAF1CF37B13E2069D6956105BD0E739499BDB && curl -sSL https://get.rvm.io | bash -s stable
#Install RVM Ruby
RUN bash -c ' export PATH=$PATH:/usr/local/rvm/bin && rvm install 2.7.1'
#Install gems
RUN bash -c 'export PATH=$PATH:/usr/local/rvm/rubies/ruby-2.7.1/bin && gem install netaddr -v 2.0.4 && gem install colorize  zip  && gem install json -v 2.3.0 && \
    gem install ffi -v 1.13.1 && \
    gem install process-terminal -v 0.2.0 && \
    gem install process-group -v 1.2.3 && \
    gem install process-pipeline -v 1.0.2 && \
    gem install graphviz -v 1.2.0'

#Add user
RUN addgroup --system deployer && adduser --system --ingroup deployer --uid 5500 deployer
#Temporary workaround to fix broken GitHub Authentication
RUN mkdir /home/deployer/.ssh && chmod 700 /home/deployer/.ssh
RUN mkdir /home/deployer/.chef && chmod 770 /home/deployer/.chef
COPY luggage/ssh_config /home/deployer/.ssh/config
COPY luggage/github_know_hosts /home/deployer/.ssh/known_hosts
RUN chown -R deployer:deployer /home/deployer/.ssh

#Configure AWS access for terraform
RUN mkdir /home/deployer/.aws && chown deployer:deployer /home/deployer/.aws
COPY --chown=deployer:deployer luggage/aws_config /home/deployer/.aws/config

#Copy files
COPY  --chown=deployer:deployer /bin/tfChek .
COPY  --chown=deployer:deployer /luggage/tfChek.sh .
COPY  --chown=deployer:deployer /templates /templates
COPY  --chown=deployer:deployer /static static
RUN chown -R deployer:deployer * && mkdir -p /var/tfChek/out && chown -R deployer:deployer /var/tfChek && mkdir /var/run/tfChek && chown -R deployer:deployer /var/run/tfChek

#Switch user
USER deployer

# Expose port 8080 to the outside world
EXPOSE 8085

# Command to run the executable
# I have to use enrtypoint to re-export PORT to TFCHEK_PORT variable
ENTRYPOINT [ "./tfChek.sh" ]