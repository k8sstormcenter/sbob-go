# Contributing

## Status

This module is work in progress. It is a prototype, built to work out the design of a portable behaviour document and the one matcher that gives it meaning, not a finished product. The grammar, the API and the package layout will still change; a change in matching semantics is a minor version at least, and until 1.0 there is no compatibility promise beyond that.

## Comments wanted

People who work on the eBPF and runtime-security side are explicitly invited to comment on issues: Falco, Tetragon, Inspektor Gadget, Kubescape, Tracee, Kubewarden, or any other engine that observes or enforces process, file and network behaviour. The questions where outside knowledge helps most:

- which patterns an in-kernel filter can carry exactly, and which have to be judged in user space;
- which events keep argument boundaries, resolved paths and peer identities, and on which kernels they can be lost;
- how a rule should be carried into an engine that has no profile functions, and when it is better to report it as not portable than to approximate it;
- whether a document's meaning is clear from the specification alone, without reading this code.

A short comment on an issue, or a new issue with one concrete case, is the most useful contribution at this stage.

## Opening an issue

Describe one case: the profile entry or pattern, the event (path, argument vector, address), the verdict you expected and the verdict you got or expect from another engine. The engine and kernel version help when the question is about what an event carries.

For a security problem, do not open a public issue; use the private vulnerability reporting of this repository.

## Changes

Open an issue before a pull request, so the design question is settled first. A pull request keeps the module on the Go standard library only and compiling with TinyGo; continuous integration checks both. Conformance tests live in a separate suite; the maintainers run it for every change and publish a summary with each release.

Contributions are accepted under the Apache License 2.0, the license of this repository.
