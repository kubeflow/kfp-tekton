FROM ubuntu:18.04

RUN set -xe  \
	&& echo '#!/bin/sh' > /usr/sbin/policy-rc.d  \
	&& echo 'exit 101' >> /usr/sbin/policy-rc.d  \
	&& chmod +x /usr/sbin/policy-rc.d  \
	&& dpkg-divert --local --rename --add /sbin/initctl  \
	&& cp -a /usr/sbin/policy-rc.d /sbin/initctl  \
	&& sed -i 's/^exit.*/exit 0/' /sbin/initctl  \
	&& echo 'force-unsafe-io' > /etc/dpkg/dpkg.cfg.d/docker-apt-speedup  \
	&& echo 'DPkg::Post-Invoke { "rm -f /var/cache/apt/archives/*.deb /var/cache/apt/archives/partial/*.deb /var/cache/apt/*.bin || true"; };' > /etc/apt/apt.conf.d/docker-clean  \
	&& echo 'APT::Update::Post-Invoke { "rm -f /var/cache/apt/archives/*.deb /var/cache/apt/archives/partial/*.deb /var/cache/apt/*.bin || true"; };' >> /etc/apt/apt.conf.d/docker-clean  \
	&& echo 'Dir::Cache::pkgcache ""; Dir::Cache::srcpkgcache "";' >> /etc/apt/apt.conf.d/docker-clean  \
	&& echo 'Acquire::Languages "none";' > /etc/apt/apt.conf.d/docker-no-languages  \
	&& echo 'Acquire::GzipIndexes "true"; Acquire::CompressionTypes::Order:: "gz";' > /etc/apt/apt.conf.d/docker-gzip-indexes  \
	&& echo 'Apt::AutoRemove::SuggestsImportant "false";' > /etc/apt/apt.conf.d/docker-autoremove-suggests
RUN [ -z "$(apt-get indextargets)" ]
RUN mkdir -p /run/systemd  \
	&& echo 'docker' > /run/systemd/container
CMD ["/bin/bash"]
ENV DEBIAN_FRONTEND=noninteractive
ENV DEBCONF_NONINTERACTIVE_SEEN=true

RUN apt-get -q update  \
	&& apt-get install -y apt-utils apt-transport-https ca-certificates curl software-properties-common \
	&& curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add - \
	&& add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu bionic stable" \
	&& add-apt-repository ppa:deadsnakes/ppa
RUN apt-get -q update  \
	&& apt-get upgrade -y  \
	&& apt-get -q clean  \
	&& apt-get -q install -y sudo apt-transport-https zip unzip bzip2 xz-utils git \
	dnsutils gettext wget build-essential openssl locales make docker-ce python3-pip \
	python3.8 python3.8-venv \
	&& update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.8 9 \
	&& update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.6 1 \
	&& curl -s https://packagecloud.io/install/repositories/github/git-lfs/script.deb.sh | sudo bash  \
	&& apt-get -q install git-lfs  \
	&& git lfs install  \
	&& apt-get -q clean  \
	&& rm -rf /var/lib/apt/lists  \
	&& locale-gen en_US.UTF-8
RUN wget https://github.com/mikefarah/yq/releases/download/v4.5.1/yq_linux_amd64.tar.gz -O - | \
	tar xz && mv yq_linux_amd64 /usr/bin/yq
ENV LANG=en_US.UTF-8
RUN git config --global http.sslverify false  \
	&& git config --global url."https://".insteadOf git://  \
	&& git config --global http.postBuffer 1048576000

RUN pip3 install -U pip setuptools && pip3 install wheel

ENV NODE_VERSION=v12.20.1
RUN curl -l https://raw.githubusercontent.com/creationix/nvm/v0.35.3/install.sh -o /tmp/install.sh  \
	&& chmod a+x /tmp/install.sh  \
	&& /tmp/install.sh  \
	&& rm /tmp/install.sh  \
	&& . /root/.nvm/nvm.sh  \
	&& nvm install $NODE_VERSION  \
	&& nvm alias default $NODE_VERSION  \
	&& cd ~/.nvm/versions/node/$NODE_VERSION/lib/  \
	&& npm install npm

ARG GO_VERSION=1.15.7
RUN curl -L https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz | tar -xz  \
    && mv go/ /usr/local/ \
	&& echo "export GOPATH=\$HOME/go" >> ~/.bashrc \
	&& echo "export PATH=\$PATH:/usr/local/go/bin:\$GOPATH/bin" >> ~/.bashrc

ARG KUBECTL_VERSION=v1.18.15
RUN wget --quiet --output-document=/usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl  \
	&& chmod +x /usr/local/bin/kubectl

RUN wget --quiet --output-document=/usr/local/bin/kustomize https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv3.2.1/kustomize_kustomize.v3.2.1_linux_amd64 \
	&& chmod +x /usr/local/bin/kustomize

RUN curl -L https://github.com/tektoncd/cli/releases/download/v0.11.0/tkn_0.11.0_Linux_x86_64.tar.gz | tar -xz tkn \
    && mv tkn /usr/local/bin/

ARG HELM2_VERSION=v2.17.0
ARG HELM3_VERSION=v3.4.2
RUN mkdir -p /tmp/helm_install  \
	&& curl https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get > get_helm.sh  \
	&& chmod 700 get_helm.sh  \
	&& ./get_helm.sh --version ${HELM2_VERSION}  \
	&& mv /usr/local/bin/helm /usr/local/bin/helm2  \
	&& ./get_helm.sh --version ${HELM3_VERSION}  \
	&& mv /usr/local/bin/helm /usr/local/bin/helm3  \
	&& ln -s /usr/local/bin/helm2 /usr/local/bin/helm  \
	&& rm -rf /tmp/helm*

ARG JQ_VERSION=1.6
RUN wget --no-check-certificate https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-linux64 -O /tmp/jq-linux64  \
	&& cp /tmp/jq-linux64 /usr/bin/jq  \
	&& chmod +x /usr/bin/jq  \
	&& rm -f /tmp/jq-linux64

ARG IBMCLOUD_VERSION=1.3.0
RUN wget --quiet -O /tmp/Bluemix_CLI.tar.gz https://download.clis.cloud.ibm.com/ibm-cloud-cli/${IBMCLOUD_VERSION}/IBM_Cloud_CLI_${IBMCLOUD_VERSION}_amd64.tar.gz  \
	&& tar -xzvf /tmp/Bluemix_CLI.tar.gz -C /tmp  \
	&& export PATH=/opt/IBM/cf/bin:$PATH  \
	&& /tmp/Bluemix_CLI/install_bluemix_cli  \
	&& rm -rf /tmp/Bluemix_CLI*  \
	&& ibmcloud config --check-version false  \
	&& mkdir -p /usr/local/Bluemix/bin/cfcli  \
	&& mkdir -p /usr/local/ibmcloud/bin/cfcli

RUN ibmcloud plugin install container-service -r Bluemix -v 1.0.208  \
	&& ibmcloud plugin install container-registry -r Bluemix -v 0.1.497  \
	&& ibmcloud plugin install cloud-functions -r Bluemix -v 1.0.49  \
	&& ibmcloud plugin install schematics -r Bluemix -v 1.4.25  \
	&& ibmcloud plugin install doi -r Bluemix -v 0.2.9  \
	&& ibmcloud plugin install cis -r Bluemix -v 1.11.0  \
	&& ibmcloud cf install -v 6.51.0 --force

RUN ln -s /usr/local/ibmcloud/bin/cfcli/cf /usr/local/Bluemix/bin/cfcli/cf  \
	&& ln -s /usr/local/ibmcloud/bin/cfcli/cf /usr/local/bin/cf  \
	&& ln -s /usr/local/ibmcloud/bin/ibmcloud /usr/local/bin/ic

ENV TMPDIR=/tmp
ENV HOME=/root
ENV PATH=/root/.nvm/versions/node/${NODE_VERSION}/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:/root/go/bin
ENV BASH_ENV=/root/.bashrc
