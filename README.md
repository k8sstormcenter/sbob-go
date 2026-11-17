# sbob-go

The reference implementation of the Software Bill of Behaviour matching grammar and collapse rules, as a Go module with no dependencies beyond the standard library.
Current goal: kubescape-node-agent produces profiles, cli uses this library to generalize them into sbobs -> transpile into a few known other runtime-stacks so they can read sbobs, too. 
Not stable, very much WIP

| package | contents |
|---|---|
| `match` | tokens, path and argument matching, the exec-name rule, the any-sentinel, IP, CIDR and DNS matching and validation, subtract evaluation, regular-expression rendering of patterns |
| `collapse` | the path analyzer and its thresholds, exec, open and endpoint collapse over plain values, consolidation, collapse settings |
| `identity` | the workload subject, its string and hashed forms, the storage name of a learned profile, workload identifier build and parse |

The specification this module implements, its conventions and schemas are maintained in the SBoB specification repository. Conformance is proven against a separate harness; a green summary is published with each release.

`cmd/wasmcheck` exists only so the module can be compiled with TinyGo for WebAssembly in continuous integration.

## Performance

An exec entry constrains its arguments either with the argument grammar (`args`) or with a POSIX regular expression over the space-joined argument vector (`argsRegex`). Expressions are compiled once per profile and a check allocates nothing; a literal prefix or a fully literal expression is decided without running the regular-expression engine.

![Nanoseconds per exec event on three large runner profiles, grammar versus argsRegex](docs/exec-args-large-profiles.svg)

The three largest profiles of a recorded set, container profiles of self-hosted CI runners with 1,806 to 3,040 exec entries. Every grammar entry was rewritten as an expression, and the same generated events, matching and near-miss argument lists for each entry, ran through both forms. On the two largest profiles the expression form is faster (0.67× and 0.78×); on the third it costs 1.35×. The grammar allocates up to 1.4 million times per pass over a profile, the expressions at most once. Both forms reach the same verdict on 91,248 of 91,254 events; the six others carry a space inside a single argument, which the joined subject cannot tell apart from two arguments.

Measured with v0.6.0 and Go 1.24 on an x86-64 Xeon at 2.8 GHz, one pinned core, median of three runs.

## Versions

The module version is semantic. A change in matching or collapse semantics is a minor version at least; images built against this module carry the module version as a label.
