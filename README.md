# sbob-go

The reference implementation of the Software Bill of Behaviour matching grammar and collapse rules, as a Go module with no dependencies beyond the standard library.

| package | contents |
|---|---|
| `match` | tokens, path and argument matching, the exec-name rule, the any-sentinel, IP, CIDR and DNS matching and validation, subtract evaluation, regular-expression rendering of patterns |
| `collapse` | the path analyzer and its thresholds, exec, open and endpoint collapse over plain values, consolidation, collapse settings |

The specification this module implements, its conventions and schemas are maintained in the SBoB specification repository. Conformance is proven against a separate harness; a green summary is published with each release.

`cmd/wasmcheck` exists only so the module can be compiled with TinyGo for WebAssembly in continuous integration.

## Versions

The module version is semantic. A change in matching or collapse semantics is a minor version at least; images built against this module carry the module version as a label.
