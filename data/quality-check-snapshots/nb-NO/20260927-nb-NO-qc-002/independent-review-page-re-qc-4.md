# nb-NO independent revision re-QC: 20260927-nb-NO-qc-002

- Reviewer: independent ChatGPT GPT-5.6 Sol High; no nb-NO candidate generation or replacement.
- Rubric: translation-quality/v1.
- Snapshot 20260927-nb-NO-qc-002; predecessor 20260927-nb-NO-qc-001.
- Revision batch chatgpt-nb-NO-004; four Page units (indexes 17, 58, 61, 93).
- Bundle SHA-256: 88ae71a5078b050c21a0d6fee8ace596aa5bb4127b005faed53dee60eaee8365.
- Bundle identity: 61dc5ba6816efc0bd257708f2e1ce4f8f57ebfb6f28121f8e94410219a35ccc8.
- Exact 18 manifest files; all listed sizes and SHA-256 verified. reviewer-bundle-check CURRENT.
- Preflight incremental, results_started=false, carry_forward=118, pending=4.
- Complete glossary, English sources, candidate and historical findings reviewed; all four validation passed, independently checked for language and technical fidelity.

## Individual ratings

| Stable index | Unit ID | Previous | Re-QC |
|---:|---|:---:|:---:|
| 17 | basics/12 | C | A |
| 58 | moretypes/22 | B | A |
| 61 | moretypes/25 | B | A |
| 93 | concurrency/1 | B | A |

## Independent findings resolution

### basics/12: A
The zero value terminology is now nullverdi, including the legal one-word Go Present emphasis form _nullverdi_. The zero values for numeric, boolean and string types, Go code, and play directive retain their source meaning.

### moretypes/22: A
The revised two-value assignment terminology does not incorrectly refer to function return values. Map insertion/retrieval/deletion, elem and ok presence checks, absence zero value and short declaration stay technically accurate. Protected code and play directive are retained.

### moretypes/25: A
The heading Closure follows the approved glossary function closure mapping, avoiding the previous unnatural compound. The body correctly explains a function value referencing variables outside its body, reading and assigning the captured variables, and distinct sum bindings per closure.

### concurrency/1: A
The two definite forms den nåværende goroutinen and den nye goroutinen are grammatical Bokmål. Arguments f/x/y/z are evaluated in the current goroutine and f runs in the new goroutine. Same address space, shared-memory synchronization, the sync link, code, Present emphasis and play directive remain correct.

## Full English sources and full current Norwegian candidates


### 17 basics/12

Source SHA-256: d9ad62501c05359e7b2ac445fa92d629e11d6d378c2701b767abab76012a558e
~~~text
* Zero values

Variables declared without an explicit initial value are given their
_zero_value_.

The zero value is:

- `0` for numeric types,
- `false` for the boolean type, and
- `""` (the empty string) for strings.

.play basics/zero.go
~~~

Candidate SHA-256: 41638d4f0b9eb5cdfc2a8500b83ffa590169df39ba2ac8007bdf3700a6798656
~~~text
* Nullverdier

Variabler som deklareres uten en eksplisitt startverdi, får sin
_nullverdi_.

Nullverdien er:

- `0` for numeriske typer,
- `false` for boolske typer og
- `""` (den tomme strengen) for strenger.

.play basics/zero.go
~~~

### 58 moretypes/22

Source SHA-256: f6963bc620184c7fbea7b04c309ccf64d508a4e835500fa7dd4188e60204019f
~~~text
* Mutating Maps

Insert or update an element in map `m`:

	m[key] = elem

Retrieve an element:

	elem = m[key]

Delete an element:

	delete(m, key)

Test that a key is present with a two-value assignment:

	elem, ok = m[key]

If `key` is in `m`, `ok` is `true`. If not, `ok` is `false`.

If `key` is not in the map, then `elem` is the zero value for the map's element type.

*Note:* If `elem` or `ok` have not yet been declared you could use a short declaration form:

	elem, ok := m[key]

.play moretypes/mutating-maps.go
~~~

Candidate SHA-256: ef61a6abb32994173b776cda33b5e3fe10322161bd797aeae9af7e93c53bdb7d
~~~text
* Endre map-er

Sett inn eller oppdater et element i map-en `m`:

	m[key] = elem

Hent et element:

	elem = m[key]

Slett et element:

	delete(m, key)

Sjekk om en nøkkel finnes, ved hjelp av en tilordning med to verdier:

	elem, ok = m[key]

Hvis `key` finnes i `m`, er `ok` lik `true`. Hvis ikke, er `ok` lik `false`.

Hvis `key` ikke finnes i map-en, er `elem` nullverdien for map-ens elementtype.

*Merk:* Hvis `elem` eller `ok` ikke er deklarert ennå, kan du bruke den korte deklarasjonsformen:

	elem, ok := m[key]

.play moretypes/mutating-maps.go
~~~

### 61 moretypes/25

Source SHA-256: 570e535e9c2b2d1978534eb3b42f226102b11da36a3f9e61620dc5eafdb6be5f
~~~text
* Function closures

Go functions may be closures. A closure is a function value that references variables from outside its body. The function may access and assign to the referenced variables; in this sense the function is "bound" to the variables.

For example, the `adder` function returns a closure. Each closure is bound to its own `sum` variable.

.play moretypes/function-closures.go
~~~

Candidate SHA-256: 57a1f64bda99530ee7a488a391fe2f72cdfd5630f542f94d3c109caa5c0192f0
~~~text
* Closure

En funksjon i Go kan være en closure. En closure er en funksjonsverdi som refererer til variabler utenfor sin egen funksjonskropp. Funksjonen kan lese og tilordne verdier til disse variablene; slik er den «bundet» til variablene.

Funksjonen `adder` returnerer for eksempel en closure. Hver closure er bundet til sin egen `sum`-variabel.

.play moretypes/function-closures.go
~~~

### 93 concurrency/1

Source SHA-256: d0a9c45078c2255540cfffe48744a07fa04ab5c2113e8880068b5a400ae0ebdc
~~~text
* Goroutines

A _goroutine_ is a lightweight thread managed by the Go runtime.

	go f(x, y, z)

starts a new goroutine running

	f(x, y, z)

The evaluation of `f`, `x`, `y`, and `z` happens in the current goroutine and the execution of `f` happens in the new goroutine.

Goroutines run in the same address space, so access to shared memory must be synchronized. The [[/pkg/sync/][`sync`]] package provides useful primitives, although you won't need them much in Go as there are other primitives. (See the next slide.)

.play concurrency/goroutines.go
~~~

Candidate SHA-256: 7f4871671e0b154f8fbc3594ad094fd994fd57eec6773dcd0e7a74e6ffd12f73
~~~text
* Goroutiner

En _goroutine_ er en lettvektstråd som styres av Go-kjøresystemet.

	go f(x, y, z)

starter en ny goroutine som kjører

	f(x, y, z)

Uttrykkene `f`, `x`, `y` og `z` evalueres i den nåværende goroutinen, mens `f` kjøres i den nye goroutinen.

Goroutiner kjører i samme adresseområde, så tilgang til delt minne må synkroniseres. Pakken [[/pkg/sync/][`sync`]] inneholder nyttige primitiver, men i Go trenger du dem sjelden, siden språket også har andre primitiver. (Se neste side.)

.play concurrency/goroutines.go
~~~

## Scope expected after record
118 carried-forward A + 4 newly-reviewed A = 122 A; B/C/D=0 and pending=0, subject to actual CLI scope checks.
No candidate files were modified by this independent reviewer.
