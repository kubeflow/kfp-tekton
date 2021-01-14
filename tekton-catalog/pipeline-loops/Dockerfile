FROM ubuntu
ARG bin_name
ENV BIN ${bin_name}

WORKDIR /

COPY _output/bin/${BIN} /usr/local/bin

ENTRYPOINT []
CMD ${BIN}
