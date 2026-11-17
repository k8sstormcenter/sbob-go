# Contributing

Thank you for your interest.

## Where this stands

This library is a prototype and a work in progress. It belongs to a proposal, the Software Bill of Behaviour, that tries to give users the same experience of runtime profiling whichever tool they use. Nothing here is finished or stable, and we do not yet know whether the approach is the right one. The grammar, the API and the package layout will change, and until 1.0 there is no compatibility promise.

## Your comments are the most valuable contribution

We would be very grateful for comments from people who build eBPF and runtime-security tools: Falco, Tetragon, Inspektor Gadget, Kubescape, Tracee, Kubewarden, or any other project that observes or enforces what processes, files and network connections do. You know your tools far better than we do, and we would much rather adapt the proposal than ask anyone to change their design.

Questions where your experience would help us most:

- Which patterns can an in-kernel filter carry exactly, and which would have to be checked in user space?
- Which events keep argument boundaries, resolved paths and peer identities, and on which kernels can they be lost?
- How could expected behaviour be handed to a tool that has no notion of a profile, and when is it better to say a rule cannot be carried than to approximate it?
- Is anything here unnecessary, or in conflict with how your project already works?

A short comment on an issue, or a new issue with one concrete case, already helps a great deal.

## Opening an issue

It helps most if an issue describes one case: the profile entry or pattern, the event (path, argument list, address), and what you expected to happen. When the question is about what an event carries, the tool and kernel version are useful too.

For a security problem, please do not open a public issue; use the private vulnerability reporting of this repository instead.

## Changes

Please open an issue before a pull request, so we can talk about the idea first. Changes keep the library on the Go standard library only and compiling with TinyGo; continuous integration checks both.

Contributions are accepted under the Apache License 2.0, the license of this repository.
