FROM centos:centos7


# Get Java
RUN yum update -y && yum install -y -q java-1.8.0-openjdk && yum clean all
#RUN BDS_JAVA_HOME=/usr/lib/jvm/java-1.8.0-openjdk-1.8.0.151-5.b12.el7_4.x86_64/jre/
ENV BDS_JAVA_HOME=/usr/lib/jvm/java-1.8.0-openjdk-1.8.0.151-5.b12.el7_4.x86_64/jre/


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
EXPOSE 3003


# Run perceptor-scanner
CMD ["./dependencies/perceptor-scanner"]
