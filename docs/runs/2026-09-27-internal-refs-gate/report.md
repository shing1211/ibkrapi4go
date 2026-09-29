# Report: a gate that is decidable, and two bugs in writing it

- **Date**: 2026-09-27
- **Mode**: BUILD
- **Baseline**: `2d11db4` (`v1.1.22`)
- **Outcome**: complete, uncommitted pending approval

## The rule

> Every package under `internal/` appears in at least one import somewhere in the
> module.

No thresholds, no pattern matching, no judgement about intent. That matters here
specifically: the `check_money.py` gate proposed two runs ago was withdrawn after
measurement - 27 false positives in one formulation, 16 false negatives in the other -
on the grounds that a gate with judgement calls in it gets ignored the first time it
is wrong. This one can be run against any tree and the answer is the same every time.

Package paths are read from `go.mod` so the comparison against import paths is
**exact**. A prefix match would be wrong: `internal/a` and `internal/a/deeper` are
different Go packages and only the former can be imported as the latter's prefix, so
prefix matching would silently pass a package nobody imports.

A test-only importer counts. A test-only consumer of an internal package is a
legitimate consumer, and treating it as dead code would make the rule wrong rather
than strict.

## What it found

`internal/fake` - the package the previous run documented as orphaned. It is on
`ALLOWED` with the reason recorded, so the gate is green today and the exception is
printed on every run:

```
internal-refs-check OK (2 packages referenced; 1 tolerated:
  github.com/shing1211/ibkrapi4go/internal/fake)
```

An exception that is invisible is how the previous gate's problems started, so this
one is noisy about its own exemptions.

## Verification, and the two bugs it found in the checker

A gate only ever run against a passing tree is the kind of test this sequence has
learned not to trust. So it was run against the real tree with a synthetic package,
and that found two genuine bugs in the checker itself - both of which would have made
it wrong in the dangerous direction.

**1. The import regex missed imports.** The first version was a single multiline
regex:

```python
re.compile(r'^\s*(?:import\s+)?(?:[.\w]+\s+)?"([^"]+)"', re.M)
```

It found `"testing"` in a grouped import block and missed the module's own import on
the next line, because the blank line between the stdlib and third-party groups
defeated `^` with `re.M` and a greedy `\s*` that may span newlines. That is the shape
a gofmt'd file has, so it would have missed imports in real code.

**2. The alias group was greedy in a way that ate the path.** The obvious fix -
`^\s*(?:[.\w]+\s+)?_?"([^"]+)"` - is also wrong. The optional alias group matches the
path's own leading word characters, then cannot satisfy its trailing `\s+`, and then
fails because a quote was expected where a letter is. The working form needs two
alternatives rather than an optional group:

```python
re.compile(r'^\s*(?:import\s+)?(?:"([^"]+)"|[.\w]+\s+"([^"]+)")')
```

matched per line. Both bugs were only visible against a real file: a synthetic test
string without a blank line, or with a short path like `m/internal/a`, passed.

**A third failure was the probe's, not the checker's.** The end-to-end verification
reported the gate still failing after adding a test-only importer, which looked like a
third checker bug. It was not: the probe had written an import line with an opening
quote and **no closing quote**. That is not an import, and the gate was right to
ignore it. The probe now asserts its own import line is well formed, with a comment
saying why - a probe that can emit invalid input makes a failing check meaningless.

That is the same lesson as the four vacuous tests earlier in this sequence, arriving
from the other direction: the thing that looked like a defect in the check was a
defect in the check.

## The end-to-end cases

| Case | Result |
|---|---|
| new package under `internal/`, imported by nothing | exit 1, names it |
| same package, imported by a `_test.go` file | exit 0 |
| package removed again | exit 0 |

The second case is what makes this a rule about dead code rather than a rule against
new packages.

## Wiring

- `make internal-refs-check`, and added to `make check`.
- A CI step beside the money-types check.
- AGENTS.md rule 8, renumbering the README rule to 9. A rule nobody is told about is
  one that gets reintroduced; the entry also records *why* `unused` cannot catch this,
  since that is the part a future reader would otherwise have to rediscover.

## Verification

All gates green: `gofmt -l`, `go build ./...`, `go vet ./...`, `go test ./...`,
both lint profiles, `check_money.py`, `check_design`, `check_links.py`,
`check_i18n.py`. Coverage 66.7%, unchanged - this adds a checker, not SDK code.

## Production code

None. A new script, the Makefile, one CI step, and AGENTS.md.
