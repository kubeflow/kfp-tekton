FROM gcr.io/cloud-builders/kubectl

ARG bin_dir=_output/bin

WORKDIR /

COPY ${bin_dir}/kubeclient /usr/local/bin

ENTRYPOINT []
CMD ["kubeclient"]
