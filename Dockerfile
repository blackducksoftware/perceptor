FROM centos:centos7

# TODO need: oc
RUN yum -y install wget

#RUN yum -y install openshift-cli
RUN wget https://github.com/openshift/origin/releases/download/v1.5.1/openshift-origin-client-tools-v1.5.1-7b451fc-linux-64bit.tar.gz
RUN tar -xvzf openshift-origin-client-tools-v1.5.1-7b451fc-linux-64bit.tar.gz

# TODO there's got to be a better way to make an executable available under the name `oc`
RUN cp openshift-origin-client-tools-v1.5.1-7b451fc-linux-64bit/oc /bin/oc
# RUN PATH="/openshift-origin-client-tools-v1.5.1-7b451fc-linux-64bit:$PATH"

# TODO need kubeconfig, master URL
# COPY /dependencies/kubeconfig /.kube/config

RUN yum install -y -q java-1.8.0-openjdk

#RUN update-alternatives --config java
#RUN update-alternatives --config javac

#RUN BDS_JAVA_HOME=/usr/lib/jvm/java-1.8.0-openjdk-1.8.0.151-5.b12.el7_4.x86_64/jre/
ENV BDS_JAVA_HOME=/usr/lib/jvm/java-1.8.0-openjdk-1.8.0.151-5.b12.el7_4.x86_64/jre/

# TODO where should this password come from?
ENV BD_HUB_PASSWORD=blackduck

# install docker client ?

ADD ./dependencies/ ./dependencies/
CMD ["./dependencies/perceptor", "./dependencies/kubeconfig", "./dependencies/scan.cli/bin/scan.cli.sh"]
