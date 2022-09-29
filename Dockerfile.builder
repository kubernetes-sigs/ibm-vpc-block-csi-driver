FROM golang:1.18.2

WORKDIR /go/src/github.com/kubernetes-sigs/ibm-vpc-block-csi-driver
ADD . /go/src/github.com/kubernetes-sigs/ibm-vpc-block-csi-driver

ARG TAG
ARG OS
ARG ARCH

CMD ["./scripts/build-bin.sh"]
