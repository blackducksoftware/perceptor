# Perceptor

Perceptor is a cloud-native program for which detects the pods and images running in a cluster,
scans those images using the hub, and informs the user of policy violations, risks, and vulnerabilities
based on what's currently running in their cluster.

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
