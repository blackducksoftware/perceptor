# Perceptor

Perceptor is a cloud-native program for which detects the pods and images running in a cluster,
scans those images using the hub, and informs the user of policy violations, risks, and vulnerabilities
based on what's currently running in their cluster.

## Perceivers

Perceivers are responsible for interacting with the cluster manager -- whether kubernetes, openshift,
docker swarm, or docker compose.  Perceivers watch for pod and image events -- create, update, delete --
and forward those on to perceptor core.

By splitting perceivers into a separate pod, we gain two things:
 - platform independence of the perceptor core.  Perceivers require a relatively small amount of code,
   and are the only component that needs to be changed in order to support a new platform.
 - on openshift, perceivers require special permissions in order to be able to talk to the APIServer
   and watch pod and image events

## Perceptor core

This maintains a model which is essentially a join of the pods and images currently running in the system,
and the information relating to those images from the hub.

It contains business logic for deciding when and what to scan, and provides a REST API for perceivers
and scanners to communicate with it.

## Scanners

A replication controller.  Each pod is responsible for grabbing the tar file of a docker image,
and running the scan client against the tar file.

Scanners can be scaled, however, the hub itself remains a bottleneck.  Therefore, care should be exercised
when increasing the number of scanner pods, so that the hub is not overloaded.

### TODO

Split off the portion of code responsible for grabbing a docker image from the node's docker daemon.
This code requires special permissions in openshift.  By implementing this as a sidecar container, 
we minimize the amount of code which requires special permissions.

# Building

TODO

# Running

TODO

# Development Policy

Perceptor embraces the traditional values of open source projects in the Apache and CNCF communities, and embraces ideas and community over the code itself.  
Please create an issue -- or, better yet, submit a pull request -- if you have any ideas for metrics, features, tests, or anything else related to Perceptor.

# Golang Standards

We follow the same standards for golang as are followed in the moby project, the kubernetes project, and other major golang projects.  
We embrace modern golang idioms including usage of viper (for config), glide (for dependencies), and aim to stay on the 'bleeding edge', since, after all, we aim to always deploy inside of containers.
