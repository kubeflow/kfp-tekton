FROM ubuntu
ARG bin_dir=_output/bin
ARG bin_name
ENV BIN ${bin_name}

WORKDIR /

COPY ${bin_dir}/${BIN} /usr/local/bin

ENTRYPOINT []
CMD ${BIN}
