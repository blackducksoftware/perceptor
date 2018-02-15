#### Authors:  Senthil & Jay Vyas 2018 Synopsys

FROM centos:7
# install all the things we need
RUN yum install -y bind-utils wget zip unzip git jq
RUN mkdir /tmp/test

ADD imageStream /tmp/test/imageStream
ADD pod /tmp/test/pod

COPY perceptor-e2e.sh /tmp/test

RUN chmod -R 777 /tmp/test

ENTRYPOINT ["/tmp/test/perceptor-e2e.sh"]
