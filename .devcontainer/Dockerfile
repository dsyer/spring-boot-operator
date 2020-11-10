#-------------------------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See https://go.microsoft.com/fwlink/?linkid=2090316 for license information.
#-------------------------------------------------------------------------------------------------------------

FROM mcr.microsoft.com/vscode/devcontainers/go:0-1

# Avoid warnings by switching to noninteractive
ENV DEBIAN_FRONTEND=noninteractive

# This Dockerfile adds a non-root 'vscode' user with sudo access. However, for Linux,
# this user's GID/UID must match your local user UID/GID to avoid permission issues
# with bind mounts. Update USER_UID / USER_GID if yours is not 1000. See
# https://aka.ms/vscode-remote/containers/non-root-user for details.
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID
ARG DOCKER_GID=131

# Configure apt, install packages and tools
RUN if ! [ "${USER_UID}" = "1000" ]; then (groupadd --gid $USER_GID $USERNAME || echo "Group already exists" ) \
    && useradd -s /bin/bash --uid $USER_UID --gid $USER_GID -m $USERNAME; fi

# download kubebuilder
RUN os=$(go env GOOS) && arch=$(go env GOARCH) && \
    curl -sL https://go.kubebuilder.io/dl/2.0.1/${os}/${arch} | tar -xz -C /tmp/ \
    && mv /tmp/kubebuilder_2.0.1_${os}_${arch} /usr/local/kubebuilder

ENV PATH /usr/local/kubebuilder/bin:$PATH

# download kustomize
RUN opsys=linux && arch=$(go env GOARCH) && curl -s https://api.github.com/repos/kubernetes-sigs/kustomize/releases |\
  grep browser_download |\
  grep ${arch} |\
  grep $opsys |\
  cut -d '"' -f 4 |\
  grep /kustomize/v |\
  sort | tail -n 1 |\
  xargs curl -O -L \
  && tar xzf ./kustomize_v*_${opsys}_amd64.tar.gz && mv kustomize /usr/local/bin \
  && rm -rf ./kustomize_v*

# Install Docker CE CLI
RUN apt-get update \
    && apt-get install -y apt-transport-https ca-certificates curl gnupg-agent software-properties-common lsb-release bash-completion \
    && curl -fsSL https://download.docker.com/linux/$(lsb_release -is | tr '[:upper:]' '[:lower:]')/gpg | apt-key add - 2>/dev/null \
    && add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/$(lsb_release -is | tr '[:upper:]' '[:lower:]') $(lsb_release -cs) stable" \
    && apt-get update \
    && apt-get install -y docker-ce-cli \
    && curl -sSL "https://github.com/docker/compose/releases/download/1.24.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose \
    && chmod +x /usr/local/bin/docker-compose \
    # Clean up
    && apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid $DOCKER_GID docker \
    && usermod -aG docker $USERNAME

RUN curl -sSL -o /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl \
    && chmod +x /usr/local/bin/kubectl
