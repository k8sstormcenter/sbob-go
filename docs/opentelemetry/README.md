# Toward semantic rules: borrowing meaning from OpenTelemetry

Status: a draft for discussion. Nothing described here is implemented, and every part of it may change or be dropped after feedback.

## Why we are thinking about this

Today the rules that accompany a behaviour document are written against the event fields and rule identifiers of one engine, the Kubescape node-agent: `event.exepath`, `event.args`, `event.path`, `event.dstAddr`, rule `R0001`, and so on. A rule therefore means what that engine's event model means, and another tool has to translate every field before it can even read the rule.

OpenTelemetry already gives shared names to many of the same things: a process's executable path, its arguments, the file it touched, the address it connected to, the container and pod it ran in. The semantic conventions also have a small namespace for describing a security rule. If the rules were written against those names, a rule could mean the same thing in every tool that emits or maps to OpenTelemetry, and a finding could travel through the OpenTelemetry pipelines people already run.

We would like to explore that direction carefully and in the open, and we would be grateful for corrections from people who work on the semantic conventions and on the engines.

## What we would borrow

The names below were read from the OpenTelemetry semantic-conventions attribute registry on 2026-11-17. Their stability as listed there is given in the last column: Stable, Release Candidate (RC) or Development. Several of the names we would need are still in development, and the registry itself says that it is a work in progress.

### Names for what an event is about

| concept | today (Kubescape node-agent event) | OpenTelemetry attribute | status |
|---|---|---|---|
| executable that ran | `event.exepath` | `process.executable.path` | RC |
| process name | `event.comm` | `process.executable.name` | RC |
| argument vector | `event.args` | `process.command_args` | RC |
| process id, parent id | `event.pid`, parent | `process.pid`, `process.parent_pid` | RC |
| effective user | user information | `process.user.id`, `process.user.name` | RC |
| interactive session | (exec with a terminal) | `process.interactive`, `process.session_leader.pid` | RC |
| container | `event.containerId` | `container.id` | Stable |
| container in the pod | `event.containerName` | `k8s.container.name` | Stable |
| pod and namespace | pod name, namespace | `k8s.pod.name`, `k8s.pod.uid`, `k8s.namespace.name` | Stable |
| node, cluster | node name | `k8s.node.name`, `k8s.cluster.name` | Stable |
| image | image name and digest | `container.image.name`, `container.image.id` | Stable, Development |
| file opened | `event.path` | `file.path` | Development |
| symbolic link created | `event.oldPath`, `event.newPath` | `file.symbolic_link.target_path`, `file.path` | Development |
| connection peer | `event.dstAddr`, `event.dstPort` | `destination.address`, `destination.port` | Development |
| transport | `event.proto` | `network.transport` | Stable |
| DNS question and answers | `event.name`, `event.addresses` | `dns.question.name`, `dns.answers` | Development |
| HTTP request | method, path | `http.request.method`, `url.path` | Stable |

Two notes from the registry that matter for behaviour profiles:

- `process.command_args` and `process.command_line` carry the guidance that they "SHOULD NOT be collected by default unless there is sanitization that excludes sensitive data". Argument profiles need the arguments, so a semantic rule set would have to say how it meets that guidance, for example by matching on the node and never exporting raw arguments.
- `process.executable.name` and the kernel's task name (`comm`, at most 15 characters) are not always the same string. A rule that relies on the task name would need to say so.

### Names for the rule and the finding

The registry has a `security_rule` namespace, all in development: `security_rule.name`, `security_rule.uuid`, `security_rule.category`, `security_rule.description`, `security_rule.reference`, `security_rule.ruleset.name`, `security_rule.version` and `security_rule.license`.

A finding could then be an OpenTelemetry event: a log record whose EventName names the kind of finding (the `event.name` attribute is deprecated in favour of that field), carrying

- the `security_rule.*` attributes of the rule that matched, for example `security_rule.name` "Unexpected process launched", a stable `security_rule.uuid` in place of today's `R0001`, the rule set in `security_rule.ruleset.name`, and a link to the rule's clause in `security_rule.reference`;
- the event's own attributes from the table above;
- the container, pod and node as resource attributes, and the severity as the log record's severity.

How a finding should be named, and whether `security_rule` is the right home for "behaviour outside its profile", are open questions below.

### Rules written against these names

A sketch of what two of today's rules might look like. The function names are placeholders.

| rule | today | written against semantic names |
|---|---|---|
| R0001, unexpected process | `!cp.was_executed(event.containerId, parse.get_exec_path(event.args, event.comm, event.exepath))` | `!profile.allows_exec(container.id, process.executable.path, process.command_args)` |
| R0010, sensitive file access | `event.path.startsWith('/etc/shadow') && !cp.was_path_opened(event.containerId, event.path)` | `file.path.startsWith('/etc/shadow') && !profile.allows_open(container.id, file.path)` |

Each engine would keep its own event model and map it onto these names once. As far as we understand their models, that would mean `event.exepath` in the Kubescape node-agent, `proc.exepath` in Falco and `process.binary` in Tetragon all becoming `process.executable.path`. Those mappings are ours to check with each project and are not settled.

## Where there is no name yet

Several things today's rules look at have no attribute in the registry as we read it:

- the flags an open was made with (read, write, create, truncate);
- hard links (symbolic links have `file.symbolic_link.target_path`; hard links have nothing comparable);
- Linux capabilities;
- system-call names;
- kernel modules loaded;
- the parent process's executable;
- the verdict itself, that is, whether the document allowed what happened.

For each we see two paths: propose an addition to the semantic conventions through their usual process, or, until such a name exists, use a clearly marked and temporary extension namespace (for example `sbob.*`) that is retired as soon as an OpenTelemetry name exists. We would much prefer the first, and we would only use the second for as long as the first takes.

## What it might change in this library

Only a sketch, to show the size of the idea:

- a table of the concepts above and their OpenTelemetry names, kept in `conventions` like the other names it already holds;
- profile functions that take semantic values (a path, an argument vector, an address), which they largely do already; the change is in how the rule inputs are named;
- the portability classes in `rules` applied to rules written against semantic names;
- a description of a finding as an OpenTelemetry event.

None of this exists yet, and none of it should be built before the open questions have had answers.

## Open questions

- Is `security_rule.*` the right place for a finding that says "this behaviour was outside its profile", or would the semantic-conventions community prefer a separate namespace for behaviour profiles?
- How should a rule set that needs argument vectors meet the collection guidance for `process.command_args`?
- Which of the missing names above would be welcome as proposals, and in which group?
- Should the identifier of a rule be a UUID, a name, or both, and who would hand them out?
- Would engine maintainers find a shared mapping of their fields onto these names useful, or would they rather keep it in their own projects?

Comments on any of these are very welcome, see [CONTRIBUTING.md](../../CONTRIBUTING.md).
