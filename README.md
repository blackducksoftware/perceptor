[![Build Status](https://travis-ci.com/blackducksoftware/perceptor.svg?branch=master)](https://travis-ci.com/blackducksoftware/perceptor)

# Perceptor

Perceptor is an API server and event handler for consuming, storing, and queueing various workloads associated with responding to events that occur in distributed orchestration systems.  Canonically, it manages information related to container events that happen in cloud native orchestration systems (i.e. openshift, kubernetes, ...).  It is meant to live in a decoupled state from its companion containers, which are called perceivers, described in the next section of this README.

The Perceptor API is currently managed in [a swagger document](./api/perceptor-swagger-spec.json), and can, in principle, be consumed from any programming language.

At the moment, the golang API objects maintained in this directory are manually curated, and are currently the only APIs that we support and test (contributions welcome!).

# Perceivers

Perceivers are the canonical extension point to a Perceptor-based deployment to support new platforms and orchestration systems.

Perceivers are workers that notify Perceptor of events, and respond to information that Perceptor acquires about those events.  If you want to build one for your own platform, or customize the way cluster events are processed, check out [our Perceivers repo](https://github.com/blackducksoftware/perceivers) to learn more about them!

Perceivers are responsible for interacting with the cluster manager -- whether kubernetes, openshift,
docker swarm, or docker compose.  Perceivers watch for pod and image events -- create, update, delete --
and forward those on to Perceptor core.

By splitting perceivers into a separate pod, we gain two things:
 - platform independence of the Perceptor core.  Perceivers require a relatively small amount of code,
   and are the only component that needs to be changed in order to support a new platform.
 - on openshift, perceivers require special permissions in order to be able to talk to the APIServer
   and watch pod and image events

Perceivers:

 - openshift3.6
 - openshift3.7
 - kubernetes
 - GKE (TODO)
 - compose (TODO)
 - swarm (TODO)


# Perceptor core

This maintains a model which is essentially a join of the pods and images currently running in the system,
and the information relating to those images from the hub.

It contains business logic for deciding when and what to scan, and provides a REST API for perceivers
and scanners to communicate with it.

## REST API

 - [docs](./api/perceptor-swagger-spec.json) -- check out [this online viewer](http://editor2.swagger.io/#!/?import=https://raw.githubusercontent/blackducksoftware/perceptor/master/api/perceptor-swagger-spec.json) to get a nice UI

Modifications to the REST API or data model should be done with great care to not break backward compatibility. If REST API or data model changes are performed, the following must occur:

 - [ ] The [swagger specification](./api/perceptor-swagger-spec.json) file must be modified
 - [ ] The updated swagger specification must be present as part of the PR containing the modifications to the server

 Going forward it will be beneficial to automatically generate server stubs from the swagger specification. This would further canonize the swagger specification as the *single source of truth*.


# Scanners

A replication controller.  Each pod is responsible for grabbing the tar file of a docker image,
and running the scan client against the tar file.

Scanners can be scaled, however, the hub itself remains a bottleneck.  Therefore, care should be exercised
when increasing the number of scanner pods, so that the hub is not overloaded.

# Development Environment Setup

Install gimme, run it to compile perceptor:

```
curl -sL -o ~/bin/gimme https://raw.githubusercontent.com/travis-ci/gimme/master/gimme
chmod +x ~/bin/gimme
export PATH=$PATH:~/bin/
gimme 1.9
```

Getting work done:

Create a GO Project for Perceptor:

```
cd <to_your_favorite_directory>
mkdir go/
mkdir go/src/
mkdir go/src/github.com/blackducksoftware/perceptor
mkdir -p go/src/github.com/blackducksoftware/perceptor
cd go/src/github.com/blackducksoftware/perceptor
install go-plus Atom package
```

Clone Perceptor:

```
git clone https://github.com/blackducksoftware/perceptor.git
```

Set up your GOPATH:

```
export GOPATH=/Users/jsmith/workspace-perceptor/go/
```

# Building

Check out [the makefile](./Makefile) -- from the root directory, run:

    make

# Continuous Integration

We build images, per commit, using cloud build files.  We're open to changing our build artifacts over time, check out the [cloudbuild.yaml](./cloudbuild.yaml).  Note that post build hooks are currently disabled due to internal infrastructure changes.

# Running

Check out [Protoform](https://github.com/blackducksoftware/perceptor-protoform/)!

# Development Policy

Perceptor embraces the traditional values of open source projects in the Apache and CNCF communities, and embraces ideas and community over the code itself.

## Pardon our dust

Although Perceptor is stable and has been heavily tested at large scales, it's relatively new as an upstream project, and we're working on building the community.  If you have suggestions on how we could do a better job, let us know.

## See a place to improve things?

Please create an issue -- better yet, accompanied with a pull request-- if you have any ideas for metrics, features, tests, or anything else related to Perceptor.

## Sticking with golang Standards

We follow the same standards for golang as are followed in the moby project, the kubernetes project, and other major golang projects.  

We embrace modern golang idioms including usage of viper for configuration, glide for dependencies, and aim to stay on the 'bleeding edge', since, after all, we aim to always deploy inside of containers.

## Testing your patches

We enable travis-ci for builds, which runs all the unit tests associated with your patches.  Make sure you submit code with unit tests when possible and verify your tests pass in your pull request.    If there are any issues with travis, file an issue and assign it to Jay (jayunit100) and Senthil (msenmurgan).
