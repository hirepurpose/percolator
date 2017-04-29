FROM golang:1.7

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y wget make rsync htop

# Copy the build product
COPY product /service/percolator
# Build our product
WORKDIR /service/percolator

# Set our entry point and expose ports
ENTRYPOINT ["/service/percolator/bin/percolator"]
EXPOSE 3000-3999
