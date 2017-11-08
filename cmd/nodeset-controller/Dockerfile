FROM debian:jessie
LABEL maintainer "henrik@loodse.com"

RUN apt-get update && apt-get install -y ca-certificates

ADD _output/nodeset-controller /nodeset-controller
