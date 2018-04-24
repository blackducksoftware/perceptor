[![Build Status](https://travis-ci.com/blackducksoftware/perceptor.svg?branch=master)](https://travis-ci.com/blackducksoftware/perceptor)

# Perceptor

Perceptor is an API server and event handler for consuming, storing, and queueing various workloads associated with responding to events that occur in distributed orchestration systems.  Canonically, it manages information related to container events that happen in cloud native orchestration systems (i.e. openshift, kubernetes, ...).  It is meant to live in a decoupled state from its companion containers, which are called perceivers, described in the next section of this README.

The Perceptor REST API is documented [here](./api/perceptor-swagger-spec.json), and can be consumed from any programming language.

# Perceptor core

This maintains a model which is essentially a join of the pods and images currently running in the system,
and the information relating to those images from the hub.

It contains business logic for deciding when and what to scan, and provides a REST API for perceivers
and scanners to communicate with it.


## REST API

 - [docs](./api/perceptor-swagger-spec.json) -- check out [this online viewer](http://editor2.swagger.io/#!/?import=https://raw.githubusercontent/blackducksoftware/perceptor/master/api/perceptor-swagger-spec.json) to get a nice UI

Modifications to the REST API or data model should be done with great care in order to maintain backward compatibility. REST API change checklist:

 - The [swagger specification](./api/perceptor-swagger-spec.json) must be modified
 - The updated swagger specification must be present as part of the PR containing the modifications to the server

**TODO**: Automatically generate server stubs from the swagger specification.
 
# Perceivers

Perceivers are workers that notify Perceptor of events, and respond to information that Perceptor acquires about those events.

Perceivers are the canonical extension point to a Perceptor-based deployment to support new platforms and orchestration systems.  If you want to build one for your own platform, or customize the way cluster events are processed, check out [our Perceivers repo](https://github.com/blackducksoftware/perceivers) to learn more about them!

Perceivers are responsible for interacting with the platform machinery, finding events and information of interest, and forwarding those to Perceptor.  Perceivers are also responsible for taking action based on the scan results available in Perceptor -- whether creating alerts, or writing information into Kubernetes objects.

By splitting perceivers into a separate pod, we gain two things:
 - platform independence of the Perceptor core.  Perceivers require a relatively small amount of code,
   and are the only component that needs to be changed in order to support a new platform.
 - on openshift, perceivers require special permissions in order to be able to talk to the APIServer
   and watch pod and image events

# Scanners

Scanners are responsible for performing scan jobs by pulling from the Perceptor scan queue.  They do this by using a Black Duck Hub scan client and a running Black Duck Hub.  While perceptor scanners can be scaled, the hub itself remains a bottleneck.

# Development Environment Setup

1. Install gimme:

```
curl -sL -o ~/bin/gimme https://raw.githubusercontent.com/travis-ci/gimme/master/gimme
chmod +x ~/bin/gimme
export PATH=$PATH:~/bin/
gimme 1.9
```

2. If necessary, set up your `GOPATH` and `GOROOT` environment variables.

3. Create a directory based off your GOPATH for Perceptor:

```
cd $GOPATH
mkdir -p src/github.com/blackducksoftware
cd go/src/github.com/blackducksoftware
```

4. Clone the Perceptor repo:

```
git clone https://github.com/blackducksoftware/perceptor.git
```

5. if using Atom, install the go-plus Atom package

# Building

Check out [the makefile](./Makefile) -- from the root directory, run:

    make

# Continuous Integration

We build images, per commit, using cloud build files.  We're open to changing our build artifacts over time; take a look at [cloudbuild.yaml](./cloudbuild.yaml).

# Running

Check out [Protoform](https://github.com/blackducksoftware/perceptor-protoform/)!

# Development Policy

Perceptor embraces the traditional values of open source projects in the Apache and CNCF communities, and embraces ideas and community over the code itself.

## See a place to improve things?

Please create an issue -- better yet, accompanied with a pull request-- if you have any ideas for metrics, features, tests, or anything else related to Perceptor.

## Sticking with golang Standards

We follow the same standards for golang as are followed in the moby project, the kubernetes project, and other major golang projects.  

We embrace modern golang idioms including usage of viper for configuration, glide for dependencies, and aim to stay on the 'bleeding edge', since, after all, we aim to always deploy inside of containers.

## Testing your patches

We enable travis-ci for builds, which runs all the unit tests associated with your patches.  Make sure you submit code with unit tests when possible and verify your tests pass in your pull request.    If there are any issues with travis, file an issue and assign it to Jay (jayunit100) and Senthil (msenmurgan).
