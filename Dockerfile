#FROM alpine:3.7
ARG PROXY_IMAGE_URL=blank

FROM "${PROXY_IMAGE_URL}"/ubi8/ubi-minimal:8.4-205

# Default values
ARG git_commit_id=unknown
ARG git_remote_url=unknown
ARG build_date=unknown
ARG jenkins_build_number=unknown
ARG REPO_SOURCE_URL=blank
ARG BUILD_URL=blank

# Add Labels to image to show build details
LABEL git-commit-id=${git_commit_id}
LABEL git-remote-url=${git_remote_url}
LABEL build-date=${build_date}
LABEL jenkins-build-number=${jenkins_build_number}
LABEL razee.io/source-url="${REPO_SOURCE_URL}"
LABEL razee.io/build-url="${BUILD_URL}"

#RUN apk update && apk --no-cache add ca-certificates nfs-utils rpcbind
RUN microdnf update && microdnf install -y ca-certificates

RUN mkdir -p /home/ibm-csi-drivers/
ADD ibm-vpc-block-csi-driver /home/ibm-csi-drivers
RUN chmod +x /home/ibm-csi-drivers/ibm-vpc-block-csi-driver

USER 2121:2121

ENTRYPOINT ["/home/ibm-csi-drivers/ibm-vpc-block-csi-driver"]
