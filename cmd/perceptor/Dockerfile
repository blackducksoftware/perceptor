FROM centos:centos7


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
COPY ./perceptor ./perceptor


# Ports
EXPOSE 3001


# Run perceptor
CMD ["./perceptor"]
