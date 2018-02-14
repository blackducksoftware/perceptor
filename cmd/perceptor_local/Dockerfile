FROM centos:centos7

# Make the openshift cli executable available under the name `oc`
RUN yum -y install wget
RUN wget https://github.com/openshift/origin/releases/download/v1.5.1/openshift-origin-client-tools-v1.5.1-7b451fc-linux-64bit.tar.gz
RUN tar -xvzf openshift-origin-client-tools-v1.5.1-7b451fc-linux-64bit.tar.gz
# TODO there's got to be a better way to do this
RUN cp openshift-origin-client-tools-v1.5.1-7b451fc-linux-64bit/oc /bin/oc


# TODO need master URL ... maybe?


# Get Java
RUN yum install -y -q java-1.8.0-openjdk
#RUN BDS_JAVA_HOME=/usr/lib/jvm/java-1.8.0-openjdk-1.8.0.151-5.b12.el7_4.x86_64/jre/
ENV BDS_JAVA_HOME=/usr/lib/jvm/java-1.8.0-openjdk-1.8.0.151-5.b12.el7_4.x86_64/jre/


# Set up environment
# TODO where should this password come from?
ENV BD_HUB_PASSWORD=blackduck


# Set up non-root user
# TODO not really sure what exactly this all does, nor whether it should be done
# RUN useradd --create-home --shell /bin/bash --gid root mynewuser
# WORKDIR /home/mynewuser
# # tmp gets the docker tarballs
# RUN mkdir tmp
# RUN chmod -R 775 /home/mynewuser
# RUN chown -R mynewuser:root /home/mynewuser
# USER mynewuser


# Bring in other dependencies
COPY ./dependencies/ ./dependencies/


# Provide access to host docker daemon
# TODO is this necessary within openshift?
# VOLUME ["/var/run/docker.sock"]


# Ports
EXPOSE 3001


# Run perceptor_local
CMD ["./dependencies/perceptor_local", "./dependencies/kubeconfig"]
