#### Author:  Joel Sheppard 2018 Synopsys, Inc

FROM centos:7
# Install all the things
RUN yum install -y bind-utils wget zip unzip git jq
RUN mkdir /tmp/test

COPY kube-prcptr-tests.sh /tmp/test
COPY ../common/perceptorTestNS.yml /tmp/test

RUN chmod -R 777 /tmp/test

ENTRYPOINT ["/tmp/test/kube-prcptr-tests.sh"]
