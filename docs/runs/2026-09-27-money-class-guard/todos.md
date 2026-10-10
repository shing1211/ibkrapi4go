# Todos: close the float32 money class

- [x] Measure how many `float32` fields the generated client holds, and how many
      sound monetary
- [x] Establish whether any of them actually reach a caller as money
      (answer: no — `rawToString` uses `json.Number`, and `float32ToStr` has
      zero production callers)
- [x] Identify the gate's blind spot: `check_money.py` inspects declarations,
      not function bodies
- [x] Add a second rule rejecting `strconv.FormatFloat` in production
      `pkg/ibkr` code
- [x] Add the indirect vector: a helper taking a binary float and returning a
      string
- [x] Add `ALLOWED_FLOAT_FORMATS` as a reasoned escape hatch; leave it empty
- [x] Extend the script's `_self_test` to six cases covering both vectors and the
      sanctioned `json.Number` path
- [x] Confirm both vectors are caught by reintroducing each
- [x] Delete the dead `float32ToStr` and `rest_float32_test.go`
- [x] Re-verify `checkMoneyNumberFields` still catches a money field regaining
      `float32`
- [x] Correct `07-money-and-numbers.md`, whose description of the gate was wrong
      in two ways
- [x] Full verification and this run's artifacts

## Deliberately not done

- [ ] Retype the 198 latent `float32` money fields in the generated client. A
      large spec change needing per-field knowledge of whether the gateway quotes
      the value; out of scope for a patch.
- [ ] Audit the 21 remaining `time.Sleep` sites. The 250 ms ones in
      `session_test.go` are the most suspicious.
- [ ] Close the general `unused` blind spot for production code kept alive only
      by a test, beyond the money shape.
