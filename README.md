# taki
Totally Awesome Kubernetes Imager

Taki is a tool for creating images of running Kubernetes containers for the purposes of incident response and digital forensics.

## Functionality Overview
Taki "images" are effectively a filesystem-diff between the base image the container was created from and the current
state of the running container. The client detects changes, collects new and modified files, and then downloads
a `tar` file of only those changed files. 

First your Kubernetes cluser must have access to the taki container, which will collect the information cluster-side.
The client runs `kubectl debug` to start the taki container, and communicates with it using `kubectl`'s stdio.

I considered other options for communication, but leveraging `kubectl` means that if the user can run `kubectl debug`
then `taki` will work correctly, no need for the user to debug communication issues. 
