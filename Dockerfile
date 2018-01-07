FROM golang:1.9

ENV DEBIAN_FRONTEND=noninteractive
ENV PBDIST=v3.2.0/protoc-3.2.0-linux-x86_64.zip
RUN apt-get update && apt-get install -y wget make rsync htop

# Copy the build product
COPY product /service/percolator
# Build our product
WORKDIR /service/percolator

# Set our entry point and expose ports
ENTRYPOINT ["/service/percolator/bin/percolator"]
EXPOSE 3000-3999
