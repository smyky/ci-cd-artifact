# Use CentOS as base image
FROM centos:7

# Install necessary packages
#RUN yum -y update 
RUN yum -y install wget 

# Install Go
ENV GOLANG_VERSION 1.17.5
RUN wget -q https://golang.org/dl/go$GOLANG_VERSION.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go$GOLANG_VERSION.linux-amd64.tar.gz && \
    rm -f go$GOLANG_VERSION.linux-amd64.tar.gz

ENV PATH $PATH:/usr/local/go/bin

# Set Go environment variables
ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH

# Create app directory
WORKDIR /app

# Copy the source code into the container
COPY main.go .

# Install dependencies
RUN go mod init main.go && \
    go mod tidy
