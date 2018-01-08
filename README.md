# canary
Perceptor is a ???TODO??? .

# Building

Run the dockerfile; it will build the binary:

```
docker build -t bdsengineering/perceptor:1.0
```

`build.sh` is just a hacky convenience script for local development; use at your own risk.

# Running

Run the replication controller :)

TODO: Need to update this with perceptor and perceiver YAML.

# Development Policy
Perceptor embraces the traditional values of open source projects in the Apache and CNCF communities, and embraces ideas and community over the code itself.  Please create an issue -- or, better yet, submit a pull request if you have any suggestions around metrics or checks that you think will be generically useful to organizations that ship code which is meant to run in a microservice environment.

# Golang Standards
We follow the same standards for golang as are followed in the moby project, the kubernetes project, and other major golang projects.  We embrace modern golang idioms including usage of viper (for config), glide (for dependencies), and aim to stay on the 'bleeding edge', since, after all, we aim to always deploy inside of containers.
