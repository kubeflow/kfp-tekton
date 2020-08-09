FROM fluent/fluentd:v1.11-1
MAINTAINER fenglixa <fenglixa@cn.ibm.com>

USER root

RUN apk add --no-cache --update --virtual .build-deps \
        sudo build-base ruby-dev \
        && sudo gem install fluent-plugin-s3 \
        && sudo gem install fluent-plugin-kubernetes_metadata_filter \
        && sudo gem install fluent-plugin-rewrite-tag-filter \
        && sudo gem sources --clear-all \
        && apk del .build-deps \
        && rm -rf /home/fluent/.gem/ruby/2.5.0/cache/*.gem

COPY fluent.conf /fluentd/etc/
COPY entrypoint.sh /bin/

# USER fluent
