FROM ubuntu:16.04

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

# Install required filesystem utilities for CSI driver operations
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        # Core utilities
        ca-certificates \
        apt \
        # Device management (provides udevadm for device discovery)
        udev \
        # Filesystem detection and mounting (provides blkid, mount, umount)
        util-linux \
        mount \
        # ext2/3/4 filesystem support (provides mkfs.ext*, fsck, resize2fs)
        e2fsprogs \
        # XFS filesystem support (provides mkfs.xfs, xfs_repair, xfs_growfs)
        xfsprogs \
        # NFS support
        nfs-common && \
    # Security updates
    apt-get upgrade -y && \
    # Cleanup to reduce image size
    rm -rf /var/lib/apt/lists/*

# Verify all required commands are present - fail build if any are missing
RUN echo "Verifying required CSI driver dependencies..." && \
    MISSING_COMMANDS="" && \
    # Check device management commands
    command -v udevadm >/dev/null 2>&1 || MISSING_COMMANDS="$MISSING_COMMANDS udevadm" && \
    # Check filesystem detection commands
    command -v blkid >/dev/null 2>&1 || MISSING_COMMANDS="$MISSING_COMMANDS blkid" && \
    # Check mount/unmount commands
    command -v mount >/dev/null 2>&1 || MISSING_COMMANDS="$MISSING_COMMANDS mount" && \
    command -v umount >/dev/null 2>&1 || MISSING_COMMANDS="$MISSING_COMMANDS umount" && \
    # Check ext2/3/4 filesystem commands
    command -v mkfs.ext2 >/dev/null 2>&1 || MISSING_COMMANDS="$MISSING_COMMANDS mkfs.ext2" && \
    command -v mkfs.ext3 >/dev/null 2>&1 || MISSING_COMMANDS="$MISSING_COMMANDS mkfs.ext3" && \
    command -v mkfs.ext4 >/dev/null 2>&1 || MISSING_COMMANDS="$MISSING_COMMANDS mkfs.ext4" && \
    command -v fsck >/dev/null 2>&1 || MISSING_COMMANDS="$MISSING_COMMANDS fsck" && \
    command -v resize2fs >/dev/null 2>&1 || MISSING_COMMANDS="$MISSING_COMMANDS resize2fs" && \
    # Check XFS filesystem commands
    command -v mkfs.xfs >/dev/null 2>&1 || MISSING_COMMANDS="$MISSING_COMMANDS mkfs.xfs" && \
    command -v xfs_repair >/dev/null 2>&1 || MISSING_COMMANDS="$MISSING_COMMANDS xfs_repair" && \
    command -v xfs_growfs >/dev/null 2>&1 || MISSING_COMMANDS="$MISSING_COMMANDS xfs_growfs" && \
    # Fail build if any commands are missing
    if [ -n "$MISSING_COMMANDS" ]; then \
        echo "ERROR: Required commands are missing:$MISSING_COMMANDS" && \
        echo "CSI driver will not function correctly without these commands." && \
        exit 1; \
    fi && \
    echo "✓ All required CSI driver dependencies verified successfully"

RUN mkdir -p /home/ibm-csi-drivers/
ADD ibm-vpc-block-csi-driver /home/ibm-csi-drivers
RUN chmod +x /home/ibm-csi-drivers/ibm-vpc-block-csi-driver
USER 2121:2121

ENTRYPOINT ["/home/ibm-csi-drivers/ibm-vpc-block-csi-driver"]
