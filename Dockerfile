# Use CentOS as base image
FROM centos:7

# # Install necessary packages
# #RUN yum -y update 
# RUN yum -y install wget 

# # Install Go
# ENV GOLANG_VERSION 1.17.5
# RUN wget -q https://golang.org/dl/go$GOLANG_VERSION.linux-amd64.tar.gz && \
#     tar -C /usr/local -xzf go$GOLANG_VERSION.linux-amd64.tar.gz && \
#     rm -f go$GOLANG_VERSION.linux-amd64.tar.gz

# ENV PATH $PATH:/usr/local/go/bin

# # Set Go environment variables
# ENV GOPATH /go
# ENV PATH $GOPATH/bin:$PATH

# # Create app directory
# WORKDIR /app

# # Copy the source code into the container
# COPY main.go .

# # Install dependencies
# RUN go mod init main.go && \
#     go mod tidy

ARG GO_VERSION=1.21.3
ARG GOOS=linux
ARG GOARCH=amd64

# Used to instruct our Makefile to delete intermediate images on `make clean`
LABEL autodelete=true

RUN yum install -y tar gzip git gcc && \
    curl -L https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz -o /tmp/go.tar.gz && \
    tar -xzf /tmp/go.tar.gz -C /opt/ && \
    rm -f /tmp/go.tar.gz && \
    mkdir -p /go/{src,pkg,bin} && \
    yum clean all

ENV GOROOT=/opt/go
ENV GOPATH=/go
ENV PATH=${PATH}:/opt/go/bin
ENV CGO_ENABLED=0

COPY main.go /root/geolocation/main.go
WORKDIR /root/geolocation

RUN go mod init geolocation && \
    go mod tidy && \
    go build -o geolocation

RUN cp /root/geolocation/geolocation /bin/geolocation

USER nobody

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/geolocation"]
