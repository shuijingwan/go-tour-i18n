# kn-IN 首次 revision 独立 re-QC：13 个 Page

reviewer: kn-IN 长期独立 ChatGPT GPT-5.6 Sol High Reviewer；未参与任何 kn-IN Generation、revision 或 replacement
locale: kn-IN; rubric: translation-quality/v1; date: 2026-09-26
snapshot: 20260926-kn-IN-qc-002; predecessor: 20260926-kn-IN-qc-001
batch: chatgpt-kn-IN-004; real generation provenance: chatgpt / gpt-5.6-sol-high
Reviewer ZIP: /tmp/kn-IN-20260926-kn-IN-qc-002-reQC-13-reviewer.zip
ZIP SHA-256: bc23d8b7a82ae3240920ffb1aaa358e885914e979feb0154ddf72b49987b37ba
Snapshot manifest SHA-256: 0a313f4de01b18aafb629921ebde5576d611fd145bbfe1d616cfe73f43109961
Bundle identity: ad707e20bca5a3f5b37d2577f981ec98c1dfdc3267d42e8a129d3e573f020bf5
Input verification: all 36 declared ZIP files, paths, sizes and SHA-256; six authority docs, complete glossary, exact frozen 122-Unit Snapshot, batch manifest with all 13 previous ratings/findings and protected inputs, 13 complete English/candidate sections, and 13 matched passed validation artifacts.
Preflight before review: incremental; previous qc-001 effective A=109; 13 changed Page candidates; 0 additional changed units; no prior qc-002 records and no finalized predecessor.
Independent decision: 13/13 A; 0 B, 0 C, 0 D. All previous findings independently resolved; no replacement produced by Reviewer.
Renderer check: actual locked golang.org/x/tools/present.Style converts interior underscores to word spaces in italic spans; *ಇನ್ನೊಂದು*ಗಮನಿಸಿ:* renders a single bold Kannada phrase, not a literal raw marker.

## 独立逐 Unit 决定及完整英文和 Kannada 证据

### 17 basics/12 — A

Previous C finding: major: English source first introduces emphasized zero value; Kannada candidate retains rendered English zero value inside _zero_value_, violating mandatory zero value -> ಶೂನ್ಯ ಮೌಲ್ಯ. Generation must translate the visible emphasized natural-language span without breaking actual Go Present emphasis or protected structure.

Reviewer determination: Go Present renders _ಶೂನ್ಯ_ಮೌಲ್ಯ_ as italic ಶೂನ್ಯ ಮೌಲ್ಯ; numeric, boolean and string zero values retain original semantics.

source SHA-256: d9ad62501c05359e7b2ac445fa92d629e11d6d378c2701b767abab76012a558e
candidate SHA-256: 350a3442b01f5e0019db0e630b501d35d0c8e49a11d5dc6a14c898ee2e5a0926
validation SHA-256: ff4baeb4212b017d01145338a36f4b0875c7549082f829a45bf9dc4e6fee8579

**Complete English Section**

```text
* Zero values

Variables declared without an explicit initial value are given their
_zero_value_.

The zero value is:

- `0` for numeric types,
- `false` for the boolean type, and
- `""` (the empty string) for strings.

.play basics/zero.go
```

**Complete Kannada candidate**

```text
* ಶೂನ್ಯ ಮೌಲ್ಯಗಳು

ಸ್ಪಷ್ಟವಾದ ಆರಂಭಿಕ ಮೌಲ್ಯವಿಲ್ಲದೆ ಘೋಷಿಸಿದ ಚರಗಳಿಗೆ ಅವುಗಳ
_ಶೂನ್ಯ_ಮೌಲ್ಯ_ ನೀಡಲಾಗುತ್ತದೆ.

ಶೂನ್ಯ ಮೌಲ್ಯ ಹೀಗಿರುತ್ತದೆ:

- ಸಂಖ್ಯಾತ್ಮಕ ಟೈಪ್‌ಗಳಿಗೆ `0`,
- ಬೂಲಿಯನ್ ಟೈಪ್‌ಗೆ `false` ಮತ್ತು
- ಸ್ಟ್ರಿಂಗ್‌ಗಳಿಗೆ `""` (ಖಾಲಿ ಸ್ಟ್ರಿಂಗ್).

.play basics/zero.go
```

### 30 flowcontrol/8 — A

Previous C finding: major: English says division by 2z is the derivative of z²; Kannada says 2z ಎಂಬುದು z² ನ ಉತ್ಪನ್ನ, a product rather than a derivative. Generation must restore the calculus/ Newton-method derivative semantics without altering formulas or protected structures.

Reviewer determination: 2z is now explicitly the derivative (ವ್ಯುತ್ಪನ್ನ) of z²; Newton adjustment, formulas, hint and links remain faithful.

source SHA-256: d8bbee8455ff59212ef432a08312f7c7703360367325a80df3719157200316e9
candidate SHA-256: d4244fd3052c3cabb5556a990e7f5ff6afb373af07edafc3a9df7fd60e8c2157
validation SHA-256: db0b7128008c4b5aa0de291dab34e3031a6836a609f0d9cb34c35bb96bf18c96

**Complete English Section**

```text
* Exercise: Loops and Functions

As a way to play with functions and loops, let's implement a square root function: given a number x, we want to find the number z for which z² is most nearly x.

Computers typically compute the square root of x using a loop.
Starting with some guess z, we can adjust z based on how close z² is to x,
producing a better guess:

	z -= (z*z - x) / (2*z)

Repeating this adjustment makes the guess better and better
until we reach an answer that is as close to the actual square root as can be.

Implement this in the `func`Sqrt` provided.
A decent starting guess for z is 1, no matter what the input.
To begin with, repeat the calculation 10 times and print each z along the way.
See how close you get to the answer for various values of x (1, 2, 3, ...)
and how quickly the guess improves.

Hint: To declare and initialize a floating point value,
give it floating point syntax or use a conversion:

	z := 1.0
	z := float64(1)

Next, change the loop condition to stop once the value has stopped
changing (or only changes by a very small amount).
See if that's more or fewer than 10 iterations.
Try other initial guesses for z, like x, or x/2.
How close are your function's results to the [[/pkg/math/#Sqrt][math.Sqrt]] in the standard library?

(*Note:* If you are interested in the details of the algorithm, the z² − x above
is how far away z² is from where it needs to be (x), and the division by 2z is the derivative
of z², to scale how much we adjust z by how quickly z² is changing.
This general approach is called [[https://en.wikipedia.org/wiki/Newton%27s_method][Newton's method]].
It works well for many functions but especially well for square root.)

.play flowcontrol/exercise-loops-and-functions.go
```

**Complete Kannada candidate**

```text
* ಅಭ್ಯಾಸ: ಲೂಪ್‌ಗಳು ಮತ್ತು ಫಂಕ್ಷನ್‌ಗಳು

ಫಂಕ್ಷನ್‌ಗಳು ಮತ್ತು ಲೂಪ್‌ಗಳೊಂದಿಗೆ ಪ್ರಯೋಗಿಸಲು ವರ್ಗಮೂಲವನ್ನು ಕಂಡುಹಿಡಿಯುವ ಫಂಕ್ಷನ್ ಅನ್ನು ಅನುಷ್ಠಾನಗೊಳಿಸೋಣ: ಕೊಟ್ಟಿರುವ x ಸಂಖ್ಯೆಗೆ, z² ಮೌಲ್ಯವು x ಗೆ ಸಾಧ್ಯವಾದಷ್ಟು ಹತ್ತಿರವಾಗುವಂತಹ z ಸಂಖ್ಯೆಯನ್ನು ಕಂಡುಹಿಡಿಯಬೇಕು.

ಕಂಪ್ಯೂಟರ್‌ಗಳು ಸಾಮಾನ್ಯವಾಗಿ ಲೂಪ್ ಬಳಸಿ x ನ ವರ್ಗಮೂಲವನ್ನು ಲೆಕ್ಕಿಸುತ್ತವೆ.
ಮೊದಲಿಗೆ z ಗೆ ಒಂದು ಅಂದಾಜು ಮೌಲ್ಯವನ್ನು ತೆಗೆದುಕೊಂಡು, z² ಮೌಲ್ಯವು x ಗೆ ಎಷ್ಟು ಹತ್ತಿರದಲ್ಲಿದೆ ಎಂಬುದರ ಆಧಾರದ ಮೇಲೆ z ಅನ್ನು ತಿದ್ದುತ್ತಾ ಉತ್ತಮ ಅಂದಾಜು ಪಡೆಯಬಹುದು:

	z -= (z*z - x) / (2*z)

ಈ ತಿದ್ದುಪಡಿಯನ್ನು ಪುನರಾವರ್ತಿಸಿದಂತೆ ಅಂದಾಜು ಮತ್ತಷ್ಟು ನಿಖರವಾಗುತ್ತದೆ.
ಕೊನೆಗೆ ನಿಜವಾದ ವರ್ಗಮೂಲಕ್ಕೆ ಸಾಧ್ಯವಾದಷ್ಟು ಹತ್ತಿರವಾದ ಉತ್ತರ ಸಿಗುತ್ತದೆ.

ನೀಡಿರುವ `func`Sqrt` ಅನ್ನು ಈ ವಿಧಾನದಿಂದ ಪೂರ್ಣಗೊಳಿಸಿ.
ಇನ್‌ಪುಟ್ ಯಾವುದೇ ಆಗಿರಲಿ, z ಗೆ 1 ಉತ್ತಮ ಆರಂಭಿಕ ಅಂದಾಜು.
ಮೊದಲಿಗೆ ಲೆಕ್ಕಾಚಾರವನ್ನು 10 ಬಾರಿ ಪುನರಾವರ್ತಿಸಿ; ಪ್ರತಿಯೊಂದು ಹಂತದಲ್ಲಿಯೂ z ಅನ್ನು ಮುದ್ರಿಸಿ.
x ನ ವಿವಿಧ ಮೌಲ್ಯಗಳಿಗೆ (1, 2, 3, ...) ಉತ್ತರ ಎಷ್ಟು ಹತ್ತಿರ ಬರುತ್ತದೆ ಮತ್ತು ಅಂದಾಜು ಎಷ್ಟು ವೇಗವಾಗಿ ಸುಧಾರಿಸುತ್ತದೆ ಎಂಬುದನ್ನು ಗಮನಿಸಿ.

ಸುಳಿವು: ಫ್ಲೋಟಿಂಗ್ ಪಾಯಿಂಟ್ ಮೌಲ್ಯವನ್ನು ಘೋಷಿಸಿ ಆರಂಭಿಕೀಕರಿಸಲು,
ಫ್ಲೋಟಿಂಗ್ ಪಾಯಿಂಟ್ ಸಿಂಟ್ಯಾಕ್ಸ್ ಬಳಸಿ ಅಥವಾ ಟೈಪ್ ಪರಿವರ್ತನೆ ಮಾಡಿ:

	z := 1.0
	z := float64(1)

ಮುಂದೆ, ಮೌಲ್ಯ ಬದಲಾಗುವುದನ್ನು ನಿಲ್ಲಿಸಿದಾಗ (ಅಥವಾ ಅತಿ ಕಡಿಮೆ ಪ್ರಮಾಣದಲ್ಲಿ ಮಾತ್ರ ಬದಲಾದಾಗ) ಲೂಪ್ ನಿಲ್ಲುವಂತೆ ಅದರ ಷರತ್ತನ್ನು ಬದಲಿಸಿ.
ಹೀಗೆ ಮಾಡಿದರೆ 10 ಕ್ಕಿಂತ ಹೆಚ್ಚು ಅಥವಾ ಕಡಿಮೆ ಪುನರಾವರ್ತನೆಗಳು ಬೇಕಾಗುತ್ತವೆಯೇ ಎಂದು ನೋಡಿ.
z ಗೆ x ಅಥವಾ x/2 ನಂತಹ ಇತರ ಆರಂಭಿಕ ಅಂದಾಜುಗಳನ್ನೂ ಪ್ರಯತ್ನಿಸಿ.
ನಿಮ್ಮ ಫಂಕ್ಷನ್‌ನ ಫಲಿತಾಂಶಗಳು ಪ್ರಮಾಣಿತ ಲೈಬ್ರರಿಯ [[/pkg/math/#Sqrt][math.Sqrt]] ಫಲಿತಾಂಶಗಳಿಗೆ ಎಷ್ಟು ಹತ್ತಿರವಾಗಿವೆ?

(*ಗಮನಿಸಿ:* ಈ ಅಲ್ಗಾರಿದಮ್‌ನ ವಿವರಗಳಲ್ಲಿ ಆಸಕ್ತಿ ಇದ್ದರೆ, ಮೇಲಿನ z² − x ಎಂಬುದು z² ತನ್ನ ಗುರಿ ಮೌಲ್ಯವಾದ x ನಿಂದ ಎಷ್ಟು ದೂರದಲ್ಲಿದೆ ಎಂಬುದನ್ನು ಸೂಚಿಸುತ್ತದೆ. z² ನ ವ್ಯುತ್ಪನ್ನವು 2z ಆಗಿದೆ. ಆದ್ದರಿಂದ 2z ನಿಂದ ಭಾಗಿಸುವ ಮೂಲಕ, z² ಎಷ್ಟು ವೇಗವಾಗಿ ಬದಲಾಗುತ್ತಿದೆ ಎಂಬುದಕ್ಕೆ ಅನುಗುಣವಾಗಿ z ಗೆ ಮಾಡುವ ತಿದ್ದುಪಡಿಯನ್ನು ಸರಿಹೊಂದಿಸಲಾಗುತ್ತದೆ.
ಈ ಸಾಮಾನ್ಯ ವಿಧಾನವನ್ನು [[https://en.wikipedia.org/wiki/Newton%27s_method][ನ್ಯೂಟನ್ ವಿಧಾನ]] ಎನ್ನುತ್ತಾರೆ.
ಇದು ಅನೇಕ ಫಂಕ್ಷನ್‌ಗಳಿಗೆ, ವಿಶೇಷವಾಗಿ ವರ್ಗಮೂಲಕ್ಕೆ, ಚೆನ್ನಾಗಿ ಕೆಲಸ ಮಾಡುತ್ತದೆ.)

.play flowcontrol/exercise-loops-and-functions.go
```

### 34 flowcontrol/12 — A

Previous C finding: major: English deferred call does not execute until surrounding function returns; Kannada says ಸುತ್ತಲಿನ ಫಂಕ್ಷನ್ ಹಿಂತಿರುಗಿದ ನಂತರವೇ, implying after enclosing function has already returned. Generation must state correct Go defer timing: arguments evaluated immediately, deferred call executed at return before function returns to caller.

Reviewer determination: Deferred arguments evaluate immediately, and deferred call occurs at enclosing-function return before returning to caller; no reverse ordering.

source SHA-256: fa23a1d6ebf12f11843d38fe79c413a9c19bfcaf305ae64b75c360db24a436ec
candidate SHA-256: ae3ff6f429578f16b74a69c4900e25ae88dbe44abba28414fdfe8fdc1c307e5b
validation SHA-256: 1ec181b8cec985133b905392df47e6721255102279b81f227d7b3516e1fbf154

**Complete English Section**

```text
* Defer

A defer statement defers the execution of a function until the surrounding
function returns.

The deferred call's arguments are evaluated immediately, but the function call
is not executed until the surrounding function returns.

.play flowcontrol/defer.go
```

**Complete Kannada candidate**

```text
* Defer

defer ಹೇಳಿಕೆಯು ಸುತ್ತಲಿನ ಫಂಕ್ಷನ್ ಹಿಂತಿರುಗುವವರೆಗೆ ಮತ್ತೊಂದು ಫಂಕ್ಷನ್‌ನ ಕಾರ್ಯನಿರ್ವಹಣೆಯನ್ನು ಮುಂದೂಡುತ್ತದೆ.

ಮುಂದೂಡಿದ ಕರೆಯ ಆರ್ಗ್ಯುಮೆಂಟ್‌ಗಳನ್ನು ತಕ್ಷಣವೇ ಮೌಲ್ಯಮಾಪನ ಮಾಡಲಾಗುತ್ತದೆ; ಆದರೆ ಸುತ್ತಲಿನ ಫಂಕ್ಷನ್ ಹಿಂತಿರುಗಲು ಸಿದ್ಧವಾದಾಗ, ಅದು ತನ್ನನ್ನು ಕರೆದವರಿಗೆ ಹಿಂತಿರುಗುವ ಮೊದಲು ಮಾತ್ರ ಮುಂದೂಡಿದ ಫಂಕ್ಷನ್ ಕರೆಯು ಕಾರ್ಯಗತವಾಗುತ್ತದೆ.

.play flowcontrol/defer.go
```

### 37 moretypes/1 — A

Previous B finding: minor: in two preformatted pointer teaching comments, Kannada moves protected identifier p unnaturally to sentence end: ಪಾಯಿಂಟರ್ ಮೂಲಕ i ಅನ್ನು ಓದಿ p and ಪಾಯಿಂಟರ್ ಮೂಲಕ i ಗೆ ಮೌಲ್ಯ ನೀಡಿ p. Generation should improve Kannada word order while retaining code, i, p, comments and protection.

Reviewer determination: Both inline pointer comments naturally attach p to the Kannada pointer noun; read/write and i, p, &, * all correspond to the source.

source SHA-256: 7706eaf0d46acc79fdad3b7953e5571bf404551d71aaefafd8ba79f5761f9806
candidate SHA-256: d5599c5f169a0327ef79037fb874e9d5075201aed4760a7cd895bdc14981d0da
validation SHA-256: aebf6df45ef8975f131cedd693ca38231408b79b9753ff5760c67a667acf3491

**Complete English Section**

```text
* Pointers

Go has pointers.
A pointer holds the memory address of a value.

The type `*T` is a pointer to a `T` value. Its zero value is `nil`.

	var p *int

The `&` operator generates a pointer to its operand.

	i := 42
	p = &i

The `*` operator denotes the pointer's underlying value.

	fmt.Println(*p) // read i through the pointer p
	*p = 21         // set i through the pointer p

This is known as "dereferencing" or "indirecting".

Unlike C, Go has no pointer arithmetic.

.play moretypes/pointers.go
```

**Complete Kannada candidate**

```text
* ಪಾಯಿಂಟರ್‌ಗಳು

Go ಭಾಷೆಯಲ್ಲಿ ಪಾಯಿಂಟರ್‌ಗಳಿವೆ.
ಪಾಯಿಂಟರ್ ಒಂದು ಮೌಲ್ಯದ ಮೆಮೊರಿ ವಿಳಾಸವನ್ನು ಹೊಂದಿರುತ್ತದೆ.

`*T` ಟೈಪ್ `T` ಟೈಪ್‌ನ ಮೌಲ್ಯವನ್ನು ಸೂಚಿಸುವ ಪಾಯಿಂಟರ್. ಇದರ ಶೂನ್ಯ ಮೌಲ್ಯ `nil`.

	var p *int

`&` ಆಪರೇಟರ್ ತನ್ನ ಆಪರ್ಯಾಂಡ್ ಅನ್ನು ಸೂಚಿಸುವ ಪಾಯಿಂಟರ್ ಅನ್ನು ಉತ್ಪಾದಿಸುತ್ತದೆ.

	i := 42
	p = &i

`*` ಆಪರೇಟರ್ ಪಾಯಿಂಟರ್ ಸೂಚಿಸುವ ಆಧಾರಭೂತ ಮೌಲ್ಯವನ್ನು ಸೂಚಿಸುತ್ತದೆ.

	fmt.Println(*p) // i ಯ ಮೌಲ್ಯವನ್ನು ಓದಲು ಬಳಸುವ ಪಾಯಿಂಟರ್ p
	*p = 21         // i ಗೆ ಮೌಲ್ಯ ನೀಡಲು ಬಳಸುವ ಪಾಯಿಂಟರ್ p

ಇದನ್ನು "ಪಾಯಿಂಟರ್ ಸೂಚಿಸಿದ ಮೌಲ್ಯವನ್ನು ಪಡೆಯುವುದು" (dereferencing ಅಥವಾ indirecting) ಎಂದು ಕರೆಯುತ್ತಾರೆ.

C ಭಾಷೆಗಿಂತ ಭಿನ್ನವಾಗಿ, Go ಭಾಷೆಯಲ್ಲಿ ಪಾಯಿಂಟರ್ ಅಂಕಗಣಿತವಿಲ್ಲ.

.play moretypes/pointers.go
```

### 64 methods/1 — A

Previous C finding: English: A method is a function with a special _receiver_ argument.; Kannada: ಮೆಥಡ್ ಎಂದರೆ ವಿಶೇಷ _receiver_ ಆರ್ಗ್ಯುಮೆಂಟ್ ಹೊಂದಿರುವ ಫಂಕ್ಷನ್.; Major. Actual Go Present renders the introduction as italic English receiver instead of mandatory ರಿಸೀವರ್. Generation must localize this emphasized natural-language term while retaining correct emphasis and protected code.

Reviewer determination: Emphasized ರಿಸೀವರ್ now renders Kannada; special receiver argument, func/name placement and Vertex/v/Abs agree with source.

source SHA-256: 8a62096b755332ede134c87493abe49231a52ce60abebcdc107bb6a64434fc72
candidate SHA-256: 542ee9b25a2d153ad90380b78c06a8ae15cb1cb72d8ba38edebd8019e7bfa97b
validation SHA-256: f3c65e2494969dea46f73bc5707df14f5ab1808dba0e37daed3e44809f2d1497

**Complete English Section**

```text
* Methods

Go does not have classes.
However, you can define methods on types.

A method is a function with a special _receiver_ argument.

The receiver appears in its own argument list between the `func` keyword and
the method name.

In this example, the `Abs` method has a receiver of type `Vertex` named `v`.

.play methods/methods.go
```

**Complete Kannada candidate**

```text
* ಮೆಥಡ್‌ಗಳು

Go ಭಾಷೆಯಲ್ಲಿ ಕ್ಲಾಸ್‌ಗಳಿಲ್ಲ.
ಆದರೆ ಟೈಪ್‌ಗಳಿಗೆ ಮೆಥಡ್‌ಗಳನ್ನು ವ್ಯಾಖ್ಯಾನಿಸಬಹುದು.

ಮೆಥಡ್ ಎಂದರೆ ವಿಶೇಷ _ರಿಸೀವರ್_ ಆರ್ಗ್ಯುಮೆಂಟ್ ಹೊಂದಿರುವ ಫಂಕ್ಷನ್.

ರಿಸೀವರ್‌ಗೆ ಪ್ರತ್ಯೇಕ ಆರ್ಗ್ಯುಮೆಂಟ್ ಪಟ್ಟಿ ಇರುತ್ತದೆ; ಅದು `func` ಕೀವರ್ಡ್ ಮತ್ತು ಮೆಥಡ್‌ನ ಹೆಸರಿನ ನಡುವೆ ಬರುತ್ತದೆ.

ಈ ಉದಾಹರಣೆಯಲ್ಲಿ, `Abs` ಮೆಥಡ್‌ನ ರಿಸೀವರ್ `Vertex` ಟೈಪ್‌ನದ್ದಾಗಿದೆ; ಅದರ ಹೆಸರು `v`.

.play methods/methods.go
```

### 72 methods/9 — A

Previous C finding: English: An _interface_type_ is defined as a set of method signatures.; Kannada: _interface_type_ ಎಂದರೆ ಮೆಥಡ್ ಸಿಗ್ನೇಚರ್‌ಗಳ ಸಮೂಹದಿಂದ ವ್ಯಾಖ್ಯಾನಿಸಲಾದ ಟೈಪ್.; Major. Present displays italic English interface type instead of mandatory ಇಂಟರ್‌ಫೇಸ್ ಟೈಪ್ in the first defining sentence. Generation must localize the emphasized technical term and preserve the method-signature definition and markup.

Reviewer determination: Emphasized ಇಂಟರ್‌ಫೇಸ್ ಟೈಪ್ renders Kannada; method signatures and Vertex versus *Vertex implementing Abser are preserved.

source SHA-256: 39efc19ed01619130489d21e421f2c0d8947b20f408580f4ceaabce6a0feb099
candidate SHA-256: 0a25011fc8b8c52e47dba50391af85a40b60369ef71bf514159ce5d3661f6656
validation SHA-256: c427fc71da00b07f0df21a339b8ce981786f91a10961a11b9fdabfbdecb6434b

**Complete English Section**

```text
* Interfaces

An _interface_type_ is defined as a set of method signatures.

A value of interface type can hold any value that implements those methods.

*Note:* There is an error in the example code on line 22.
`Vertex` (the value type) doesn't implement `Abser` because
the `Abs` method is defined only on `*Vertex` (the pointer type).

.play methods/interfaces.go
```

**Complete Kannada candidate**

```text
* ಇಂಟರ್‌ಫೇಸ್‌ಗಳು

_ಇಂಟರ್‌ಫೇಸ್_ಟೈಪ್_ ಎಂದರೆ ಮೆಥಡ್ ಸಿಗ್ನೇಚರ್‌ಗಳ ಸಮೂಹದಿಂದ ವ್ಯಾಖ್ಯಾನಿಸಲಾದ ಟೈಪ್.

ಆ ಮೆಥಡ್‌ಗಳನ್ನು ಅನುಷ್ಠಾನಗೊಳಿಸುವ ಯಾವುದೇ ಟೈಪ್‌ನ ಮೌಲ್ಯವನ್ನು ಇಂಟರ್‌ಫೇಸ್ ಟೈಪ್‌ನ ಮೌಲ್ಯವು ಹೊಂದಿರಬಹುದು.

*ಗಮನಿಸಿ:* ಈ ಉದಾಹರಣೆಯ 22ನೇ ಸಾಲಿನ ಕೋಡ್‌ನಲ್ಲಿ ದೋಷವಿದೆ.
`Vertex` (ಮೌಲ್ಯ ಟೈಪ್) `Abser` ಅನ್ನು ಅನುಷ್ಠಾನಗೊಳಿಸುವುದಿಲ್ಲ; ಏಕೆಂದರೆ
`Abs` ಮೆಥಡ್ ಅನ್ನು `*Vertex` (ಪಾಯಿಂಟರ್ ಟೈಪ್) ಮೇಲಷ್ಟೇ ವ್ಯಾಖ್ಯಾನಿಸಲಾಗಿದೆ.

.play methods/interfaces.go
```

### 77 methods/14 — A

Previous C finding: English: The interface type that specifies zero methods is known as the _empty_interface_:; Kannada: ಯಾವುದೇ ಮೆಥಡ್ ಅನ್ನು ಸೂಚಿಸದ ಇಂಟರ್‌ಫೇಸ್ ಟೈಪ್ ಅನ್ನು _empty_interface_ ಎನ್ನುತ್ತಾರೆ:; Major. The defining emphasized span renders English empty interface rather than mandatory ಖಾಲಿ ಇಂಟರ್‌ಫೇಸ್, despite a translated heading. Generation must localize the defining span while preserving emphasis, interface{} and any equivalence.

Reviewer determination: Emphasized ಖಾಲಿ ಇಂಟರ್‌ಫೇಸ್ renders Kannada; zero methods, interface{}, any alias and fmt.Print remain exact.

source SHA-256: 7a09a7654c8b6ac6d96a1a8ddaafac368782f89ec2e6e61f323d40eb01cbf750
candidate SHA-256: a19db67c0882600a8a31255be032c618469711edd238b181b1623b3373376f24
validation SHA-256: 58370990da367fb4ce9fc09e3383eb50d8f193e50d9e76bc3812ccb286c6566d

**Complete English Section**

```text
* The empty interface

The interface type that specifies zero methods is known as the _empty_interface_:

	interface{}

An empty interface may hold values of any type.
(Every type implements at least zero methods.)

`any` is an alias for `interface{}`, and the two are completely
equivalent.

Empty interfaces are used by code that handles values of unknown type.
For example, `fmt.Print` takes any number of arguments of type `any`.

.play methods/empty-interface.go
```

**Complete Kannada candidate**

```text
* ಖಾಲಿ ಇಂಟರ್‌ಫೇಸ್

ಯಾವುದೇ ಮೆಥಡ್ ಅನ್ನು ಸೂಚಿಸದ ಇಂಟರ್‌ಫೇಸ್ ಟೈಪ್ ಅನ್ನು _ಖಾಲಿ_ಇಂಟರ್‌ಫೇಸ್_ ಎನ್ನುತ್ತಾರೆ:

	interface{}

ಖಾಲಿ ಇಂಟರ್‌ಫೇಸ್ ಯಾವುದೇ ಟೈಪ್‌ನ ಮೌಲ್ಯವನ್ನು ಹೊಂದಿರಬಹುದು.
(ಎಲ್ಲ ಟೈಪ್‌ಗಳೂ ಕನಿಷ್ಠ ಶೂನ್ಯ ಮೆಥಡ್‌ಗಳನ್ನು ಅನುಷ್ಠಾನಗೊಳಿಸುತ್ತವೆ.)

`any` ಎನ್ನುವುದು `interface{}` ಗೆ ಪರ್ಯಾಯ ಹೆಸರು; ಇವೆರಡೂ ಸಂಪೂರ್ಣವಾಗಿ ಸಮಾನ.

ಯಾವ ಟೈಪ್ ಎಂದು ಮುಂಚಿತವಾಗಿ ತಿಳಿಯದ ಮೌಲ್ಯಗಳನ್ನು ನಿರ್ವಹಿಸುವ ಕೋಡ್‌ನಲ್ಲಿ ಖಾಲಿ ಇಂಟರ್‌ಫೇಸ್‌ಗಳನ್ನು ಬಳಸಲಾಗುತ್ತದೆ.
ಉದಾಹರಣೆಗೆ, `fmt.Print` ಫಂಕ್ಷನ್ `any` ಟೈಪ್‌ನ ಬೇಕಾದಷ್ಟು ಆರ್ಗ್ಯುಮೆಂಟ್‌ಗಳನ್ನು ಸ್ವೀಕರಿಸುತ್ತದೆ.

.play methods/empty-interface.go
```

### 78 methods/15 — A

Previous C finding: English: A _type_assertion_ provides access to an interface value; Kannada: _type_assertion_ ಇಂಟರ್‌ಫೇಸ್ ಮೌಲ್ಯದೊಳಗಿನ ಆಧಾರಭೂತ ನಿರ್ದಿಷ್ಟ ಮೌಲ್ಯವನ್ನು ಪಡೆಯಲು ಅವಕಾಶ ನೀಡುತ್ತದೆ.; Major. The introductory emphasized span renders English type assertion rather than mandatory ಟೈಪ್ ಅಸರ್ಷನ್. Generation must localize the span while preserving valid emphasis and all protected i.(T), t, ok and panic semantics.

Reviewer determination: Emphasized ಟೈಪ್ ಅಸರ್ಷನ್ renders Kannada; i.(T) panic and t,ok non-panic paths, boolean and zero value are correct.

source SHA-256: c205b4eb940358edea6321cfa422c3c00a560e9a6a71b08c8b3fa3ab9e4926ae
candidate SHA-256: 108a6b853ca8d10c4880ea2f7d697bd2073ba02c0752dc9da1a26671f08967a1
validation SHA-256: 12d9687acc16b08abe917a379821f7eeb0a6d30280cd967b94c8391d2608b95b

**Complete English Section**

```text
* Type assertions

A _type_assertion_ provides access to an interface value's underlying concrete value.

	t := i.(T)

This statement asserts that the interface value `i` holds the concrete type `T`
and assigns the underlying `T` value to the variable `t`.

If `i` does not hold a `T`, the statement will trigger a panic.

To _test_ whether an interface value holds a specific type,
a type assertion can return two values: the underlying value
and a boolean value that reports whether the assertion succeeded.

	t, ok := i.(T)

If `i` holds a `T`, then `t` will be the underlying value and `ok` will be true.

If not, `ok` will be false and `t` will be the zero value of type `T`,
and no panic occurs.

Note the similarity between this syntax and that of reading from a map.

.play methods/type-assertions.go
```

**Complete Kannada candidate**

```text
* ಟೈಪ್ ಅಸರ್ಷನ್‌ಗಳು

_ಟೈಪ್_ಅಸರ್ಷನ್_ ಇಂಟರ್‌ಫೇಸ್ ಮೌಲ್ಯದೊಳಗಿನ ಆಧಾರಭೂತ ನಿರ್ದಿಷ್ಟ ಮೌಲ್ಯವನ್ನು ಪಡೆಯಲು ಅವಕಾಶ ನೀಡುತ್ತದೆ.

	t := i.(T)

ಈ ಹೇಳಿಕೆ ಇಂಟರ್‌ಫೇಸ್ ಮೌಲ್ಯವಾದ `i` ಯೊಳಗೆ `T` ಎಂಬ ನಿರ್ದಿಷ್ಟ ಟೈಪ್ ಇದೆ ಎಂದು ಅಸರ್ಷನ್ ಮಾಡಿ, ಆಧಾರಭೂತ `T` ಮೌಲ್ಯವನ್ನು `t` ಚರಕ್ಕೆ ನಿಯೋಜಿಸುತ್ತದೆ.

`i` ಯೊಳಗೆ `T` ಟೈಪ್‌ನ ಮೌಲ್ಯ ಇಲ್ಲದಿದ್ದರೆ, ಈ ಹೇಳಿಕೆ ಪ್ಯಾನಿಕ್ ಉಂಟುಮಾಡುತ್ತದೆ.

ಇಂಟರ್‌ಫೇಸ್ ಮೌಲ್ಯದಲ್ಲಿ ನಿರ್ದಿಷ್ಟ ಟೈಪ್ ಇದೆಯೇ ಎಂದು _ಪರಿಶೀಲಿಸಲು_ ಟೈಪ್ ಅಸರ್ಷನ್ ಎರಡು ಮೌಲ್ಯಗಳನ್ನು ಹಿಂತಿರುಗಿಸಬಹುದು: ಆಧಾರಭೂತ ಮೌಲ್ಯ ಮತ್ತು ಅಸರ್ಷನ್ ಯಶಸ್ವಿಯಾಯಿತೇ ಎಂಬುದನ್ನು ಸೂಚಿಸುವ ಬೂಲಿಯನ್ ಮೌಲ್ಯ.

	t, ok := i.(T)

`i` ಯಲ್ಲಿ `T` ಇದ್ದರೆ, `t` ಆಧಾರಭೂತ ಮೌಲ್ಯವನ್ನು ಹೊಂದಿರುತ್ತದೆ ಮತ್ತು `ok` true ಆಗಿರುತ್ತದೆ.

ಇಲ್ಲದಿದ್ದರೆ, `ok` false ಆಗಿದ್ದು, `t` ಯಲ್ಲಿ `T` ಟೈಪ್‌ನ ಶೂನ್ಯ ಮೌಲ್ಯ ಇರುತ್ತದೆ; ಪ್ಯಾನಿಕ್ ಸಂಭವಿಸುವುದಿಲ್ಲ.

ಈ ಸಿಂಟ್ಯಾಕ್ಸ್ ಮತ್ತು ಮ್ಯಾಪ್‌ನಿಂದ ಮೌಲ್ಯವನ್ನು ಓದುವ ಸಿಂಟ್ಯಾಕ್ಸ್ ನಡುವಿನ ಸಾಮ್ಯವನ್ನು ಗಮನಿಸಿ.

.play methods/type-assertions.go
```

### 79 methods/16 — A

Previous C finding: English: A _type_switch_ is a construct that permits several type assertions in series.; Kannada: _type_switch_ ಒಂದಾದ ನಂತರ ಒಂದರಂತೆ ಹಲವು ಟೈಪ್ ಅಸರ್ಷನ್‌ಗಳನ್ನು ಪರಿಶೀಲಿಸಲು ಅವಕಾಶ ನೀಡುವ ರಚನೆ.; Major. The introductory emphasized span renders English type switch rather than mandatory ಟೈಪ್ ಸ್ವಿಚ್. Generation must localize this span while preserving switch code, case/default behavior and Present markup.

Reviewer determination: Emphasized ಟೈಪ್ ಸ್ವಿಚ್ renders Kannada; case T/S and default retain distinct Go type semantics and permitted teaching comments.

source SHA-256: 26e4da09e80d30b06368691c76ee2940139b0f6fc40cad47bb6d1d2947933c27
candidate SHA-256: 87c8480a72a8bd4a5909cc9e619049a95a1ee0bea593d6faba07fe5f97fdc7a5
validation SHA-256: 909e82c33c2072c92deb8a490c6a31134d9a4aed7354bd417592fea63afaf416

**Complete English Section**

```text
* Type switches

A _type_switch_ is a construct that permits several type assertions in series.

A type switch is like a regular switch statement, but the cases in a type
switch specify types (not values), and those values are compared against
the type of the value held by the given interface value.

	switch v := i.(type) {
	case T:
		// here v has type T
	case S:
		// here v has type S
	default:
		// no match; here v has the same type as i
	}

The declaration in a type switch has the same syntax as a type assertion `i.(T)`,
but the specific type `T` is replaced with the keyword `type`.

This switch statement tests whether the interface value `i`
holds a value of type `T` or `S`.
In each of the `T` and `S` cases, the variable `v` will be of type
`T` or `S` respectively and hold the value held by `i`.
In the default case (where there is no match), the variable `v` is
of the same interface type and value as `i`.

.play methods/type-switches.go
```

**Complete Kannada candidate**

```text
* ಟೈಪ್ ಸ್ವಿಚ್‌ಗಳು

_ಟೈಪ್_ಸ್ವಿಚ್_ ಒಂದಾದ ನಂತರ ಒಂದರಂತೆ ಹಲವು ಟೈಪ್ ಅಸರ್ಷನ್‌ಗಳನ್ನು ಪರಿಶೀಲಿಸಲು ಅವಕಾಶ ನೀಡುವ ರಚನೆ.

ಟೈಪ್ ಸ್ವಿಚ್ ಸಾಮಾನ್ಯ switch ಹೇಳಿಕೆಯಂತೆಯೇ ಇರುತ್ತದೆ; ಆದರೆ ಅದರಲ್ಲಿ ಪ್ರತಿ case ಮೌಲ್ಯದ ಬದಲಿಗೆ ಟೈಪ್ ಅನ್ನು ಸೂಚಿಸುತ್ತದೆ. ಆ ಟೈಪ್ ಅನ್ನು ನೀಡಲಾದ ಇಂಟರ್‌ಫೇಸ್ ಮೌಲ್ಯದೊಳಗಿರುವ ಮೌಲ್ಯದ ಟೈಪ್‌ನೊಂದಿಗೆ ಹೋಲಿಸಲಾಗುತ್ತದೆ.

	switch v := i.(type) {
	case T:
		// ಇಲ್ಲಿ v ಯ ಟೈಪ್ T
	case S:
		// ಇಲ್ಲಿ v ಯ ಟೈಪ್ S
	default:
		// ಯಾವುದೂ ಹೊಂದಲಿಲ್ಲ; ಇಲ್ಲಿ v ಯ ಟೈಪ್ i ಯ ಟೈಪ್‌ನಂತೆಯೇ ಇದೆ
	}

ಟೈಪ್ ಸ್ವಿಚ್‌ನ ಘೋಷಣೆಯ ಸಿಂಟ್ಯಾಕ್ಸ್, `i.(T)` ಎಂಬ ಟೈಪ್ ಅಸರ್ಷನ್‌ನ ಸಿಂಟ್ಯಾಕ್ಸ್‌ನಂತೆಯೇ ಇರುತ್ತದೆ; ಆದರೆ ನಿರ್ದಿಷ್ಟ ಟೈಪ್ `T` ಬದಲಿಗೆ `type` ಕೀವರ್ಡ್ ಬಳಸಲಾಗುತ್ತದೆ.

ಈ ಸ್ವಿಚ್ ಹೇಳಿಕೆಯು ಇಂಟರ್‌ಫೇಸ್ ಮೌಲ್ಯ `i` ಯಲ್ಲಿ `T` ಅಥವಾ `S` ಟೈಪ್‌ನ ಮೌಲ್ಯ ಇದೆಯೇ ಎಂದು ಪರೀಕ್ಷಿಸುತ್ತದೆ.
`T` ಮತ್ತು `S` case ಗಳಲ್ಲಿ ಕ್ರಮವಾಗಿ `v` ಯ ಟೈಪ್ `T` ಅಥವಾ `S` ಆಗಿರುತ್ತದೆ; `i` ಯಲ್ಲಿ ಇರುವ ಮೌಲ್ಯವೇ ಅದರಲ್ಲಿ ಇರುತ್ತದೆ.
ಯಾವುದೇ ಟೈಪ್ ಹೊಂದದ default case ನಲ್ಲಿ `v` ಯ ಟೈಪ್ ಹಾಗೂ ಮೌಲ್ಯ ಎರಡೂ `i` ಯ ಇಂಟರ್‌ಫೇಸ್ ಟೈಪ್ ಮತ್ತು ಮೌಲ್ಯದಂತೆಯೇ ಇರುತ್ತವೆ.

.play methods/type-switches.go
```

### 95 concurrency/3 — A

Previous B finding: English: Channels can be _buffered_.  Provide the buffer length; Kannada: ಚಾನೆಲ್‌ಗಳು _buffered_ ಆಗಿರಬಹುದು, ಅಂದರೆ ಬಫರ್ ಹೊಂದಿರಬಹುದು.; Minor. Present renders English buffered immediately followed by equivalent Kannada, creating avoidable bilingual duplication; the rest of the page correctly explains buffered channels. Generation must localize the italic qualifier with valid markup without changing buffer blocking, length/capacity or code.

Reviewer determination: Emphasized ಬಫರ್ ಹೊಂದಿದ renders Kannada; buffered send blocks only full, receive only empty, buffer length preserved.

source SHA-256: caef0893a54c61f192be397f6b27a039d81dc007f98f2d203a5a287383c0682e
candidate SHA-256: 80feaa549c3d9dc9c2ffcf639069e87ac100c56951386317a338626a5a5836d7
validation SHA-256: 5f8fda7991fd7a5adbe7e76fc72ac2667b6080fd9b631f54f3c1c2b419f163e8

**Complete English Section**

```text
* Buffered Channels

Channels can be _buffered_.  Provide the buffer length as the second argument to `make` to initialize a buffered channel:

	ch := make(chan int, 100)

Sends to a buffered channel block only when the buffer is full. Receives block when the buffer is empty.

Modify the example to overfill the buffer and see what happens.

.play concurrency/buffered-channels.go
```

**Complete Kannada candidate**

```text
* ಬಫರ್ ಹೊಂದಿದ ಚಾನೆಲ್‌ಗಳು

ಚಾನೆಲ್‌ಗಳು _ಬಫರ್_ಹೊಂದಿದ_ ರೂಪದಲ್ಲಿರಬಹುದು. ಬಫರ್ ಹೊಂದಿದ ಚಾನೆಲ್ ಅನ್ನು ಆರಂಭಿಕೀಕರಿಸಲು `make` ಗೆ ಎರಡನೆಯ ಆರ್ಗ್ಯುಮೆಂಟ್ ಆಗಿ ಬಫರ್‌ನ ಉದ್ದವನ್ನು ನೀಡಿ:

	ch := make(chan int, 100)

ಬಫರ್ ತುಂಬಿದಾಗ ಮಾತ್ರ ಅದಕ್ಕೆ ಕಳುಹಿಸುವ ಕಾರ್ಯ ನಿರ್ಬಂಧಿತವಾಗುತ್ತದೆ. ಬಫರ್ ಖಾಲಿಯಾಗಿದ್ದಾಗ ಅದರಿಂದ ಸ್ವೀಕರಿಸುವ ಕಾರ್ಯ ನಿರ್ಬಂಧಿತವಾಗುತ್ತದೆ.

ಬಫರ್‌ನ ಸಾಮರ್ಥ್ಯಕ್ಕಿಂತ ಹೆಚ್ಚು ಅಂಶಗಳನ್ನು ಕಳುಹಿಸುವಂತೆ ಉದಾಹರಣೆಯನ್ನು ಬದಲಿಸಿ ಏನಾಗುತ್ತದೆ ಎಂದು ನೋಡಿ.

.play concurrency/buffered-channels.go
```

### 96 concurrency/4 — A

Previous B finding: English: *Note:* Only the sender should close a channel, never the receiver.; Kannada: *Note:* ಗಮನಿಸಿ — ಚಾನೆಲ್ ಅನ್ನು ಕಳುಹಿಸುವವರೇ ಮುಚ್ಚಬೇಕು; ಸ್ವೀಕರಿಸುವವರು ಎಂದಿಗೂ ಮುಚ್ಚಬಾರದು.; Minor. Present visibly renders English bold Note: followed by Kannada ಗಮನಿಸಿ; the second English Another note: is likewise duplicated by Kannada ಇನ್ನೊಂದು ಗಮನಿಸಿ. Generation must localize both bold labels once, preserving channel-close, panic and range-loop technical semantics.

Reviewer determination: Actual Present renders both Kannada Note labels bold; sender closes, receiver never closes; panic and range-close rules unchanged.

source SHA-256: 3e925c6745255a443c61f25cd8131d538dc8677938f1c5021ce798465020b42e
candidate SHA-256: 8476b41244a75638454b1f8b3ca8b6fb2eed0df3689a5710362a35efb004f80b
validation SHA-256: 68ef8d8bd6afeda54c329f902b470b94f509c479296feb9da6e50f1378607e3e

**Complete English Section**

```text
* Range and Close

A sender can `close` a channel to indicate that no more values will be sent. Receivers can test whether a channel has been closed by assigning a second parameter to the receive expression: after

	v, ok := <-ch

`ok` is `false` if there are no more values to receive and the channel is closed.

The loop `for`i`:=`range`c` receives values from the channel repeatedly until it is closed.

*Note:* Only the sender should close a channel, never the receiver. Sending on a closed channel will cause a panic.

*Another*note:* Channels aren't like files; you don't usually need to close them. Closing is only necessary when the receiver must be told there are no more values coming, such as to terminate a `range` loop.

.play concurrency/range-and-close.go
```

**Complete Kannada candidate**

```text
* Range ಮತ್ತು Close

ಇನ್ನು ಯಾವುದೇ ಮೌಲ್ಯಗಳನ್ನು ಕಳುಹಿಸುವುದಿಲ್ಲವೆಂದು ತಿಳಿಸಲು ಕಳುಹಿಸುವವರು ಚಾನೆಲ್ ಅನ್ನು `close` ಮೂಲಕ ಮುಚ್ಚಬಹುದು. ಚಾನೆಲ್ ಮುಚ್ಚಿದೆಯೇ ಎಂದು ತಿಳಿಯಲು ಸ್ವೀಕರಿಸುವವರು ಸ್ವೀಕರಣ ಅಭಿವ್ಯಕ್ತಿಯ ಎರಡನೆಯ ಮೌಲ್ಯವನ್ನೂ ನಿಯೋಜಿಸಬಹುದು. ಉದಾಹರಣೆಗೆ:

	v, ok := <-ch

ಸ್ವೀಕರಿಸಲು ಇನ್ನಾವುದೇ ಮೌಲ್ಯ ಇಲ್ಲದಿದ್ದರೆ ಮತ್ತು ಚಾನೆಲ್ ಮುಚ್ಚಿದ್ದರೆ `ok` ಮೌಲ್ಯ `false` ಆಗಿರುತ್ತದೆ.

`for`i`:=`range`c` ಲೂಪ್ ಚಾನೆಲ್ ಮುಚ್ಚುವವರೆಗೆ ಅದರಿಂದ ಪುನಃಪುನಃ ಮೌಲ್ಯಗಳನ್ನು ಸ್ವೀಕರಿಸುತ್ತದೆ.

*ಗಮನಿಸಿ:*  ಚಾನೆಲ್ ಅನ್ನು ಕಳುಹಿಸುವವರೇ ಮುಚ್ಚಬೇಕು; ಸ್ವೀಕರಿಸುವವರು ಎಂದಿಗೂ ಮುಚ್ಚಬಾರದು. ಮುಚ್ಚಿದ ಚಾನೆಲ್‌ಗೆ ಮೌಲ್ಯ ಕಳುಹಿಸಿದರೆ ಪ್ಯಾನಿಕ್ ಸಂಭವಿಸುತ್ತದೆ.

*ಇನ್ನೊಂದು*ಗಮನಿಸಿ:*  ಚಾನೆಲ್‌ಗಳು ಫೈಲ್‌ಗಳಂತಲ್ಲ; ಸಾಮಾನ್ಯವಾಗಿ ಅವನ್ನು ಮುಚ್ಚುವ ಅಗತ್ಯವಿಲ್ಲ. ಸ್ವೀಕರಿಸುವವರಿಗೆ ಇನ್ನು ಮೌಲ್ಯಗಳು ಬರುವುದಿಲ್ಲವೆಂದು ತಿಳಿಸಬೇಕಾದಾಗ ಮಾತ್ರ, ಉದಾಹರಣೆಗೆ `range` ಲೂಪ್ ನಿಲ್ಲಿಸಲು, ಚಾನೆಲ್ ಅನ್ನು ಮುಚ್ಚಬೇಕಾಗುತ್ತದೆ.

.play concurrency/range-and-close.go
```

### 101 concurrency/9 — A

Previous B finding: English: This concept is called _mutual_exclusion_, and the conventional name for the data structure that provides it is _mutex_.; Kannada: ಈ ಪರಿಕಲ್ಪನೆಯನ್ನು _mutual_exclusion_ (ಪರಸ್ಪರ ಹೊರಗಿಡುವಿಕೆ) ಎನ್ನುತ್ತಾರೆ; ಅದನ್ನು ಒದಗಿಸುವ ಡೇಟಾ ರಚನೆಯ ಸಾಂಪ್ರದಾಯಿಕ ಹೆಸರು _mutex_ (ಮ್ಯೂಟೆಕ್ಸ್).; Minor. Present emphasizes English mutual exclusion and mutex rather than glossary mandatory Kannada forms; correct Kannada glosses in parentheses preserve meaning but duplicate terminology. Generation must focus the two italic spans on mandatory ಪರಸ್ಪರ ಹೊರಗಿಡುವಿಕೆ and ಮ್ಯೂಟೆಕ್ಸ್, preserving sync.Mutex, Lock, Unlock and defer.

Reviewer determination: Present renders italic ಪರಸ್ಪರ ಹೊರಗಿಡುವಿಕೆ and ಮ್ಯೂಟೆಕ್ಸ್; sync.Mutex, Lock/Unlock and defer retain precise semantics.

source SHA-256: 49f727794604c2ef8a8dda9508d0f412d7015cea77754c675a02b166dd9bfc93
candidate SHA-256: 4428fe81f790859ad7f8ed58d17c7b85ed17cb074d9e11b7909bcb98e8bfce56
validation SHA-256: 3aa9e5017d25e3d11ae1fcdbee44dd2e14d217530d46dce6febe8c5a08086c26

**Complete English Section**

```text
* sync.Mutex

We've seen how channels are great for communication among goroutines.

But what if we don't need communication? What if we just want to make sure only
one goroutine can access a variable at a time to avoid conflicts?

This concept is called _mutual_exclusion_, and the conventional name for the data structure that provides it is _mutex_.

Go's standard library provides mutual exclusion with
[[/pkg/sync/#Mutex][`sync.Mutex`]] and its two methods:

- `Lock`
- `Unlock`

We can define a block of code to be executed in mutual exclusion by surrounding it
with a call to `Lock` and `Unlock` as shown on the `Inc` method.

We can also use `defer` to ensure the mutex will be unlocked as in the `Value` method.

.play concurrency/mutex-counter.go
```

**Complete Kannada candidate**

```text
* sync.Mutex

goroutine‌ಗಳ ನಡುವೆ ಸಂವಹನಕ್ಕೆ ಚಾನೆಲ್‌ಗಳು ಎಷ್ಟು ಉಪಯುಕ್ತವೆಂದು ನೋಡಿದ್ದೇವೆ.

ಆದರೆ ನಮಗೆ ಸಂವಹನದ ಅಗತ್ಯವೇ ಇಲ್ಲದಿದ್ದರೆ? ಸಂಘರ್ಷಗಳನ್ನು ತಪ್ಪಿಸಲು ಒಂದು ಸಮಯದಲ್ಲಿ ಒಂದೇ goroutine ಚರವನ್ನು ಪ್ರವೇಶಿಸಬಹುದೆಂದು ಖಚಿತಪಡಿಸಿಕೊಳ್ಳುವುದಷ್ಟೇ ಬೇಕಾದರೆ?

ಈ ಪರಿಕಲ್ಪನೆಯನ್ನು _ಪರಸ್ಪರ_ಹೊರಗಿಡುವಿಕೆ_ ಎನ್ನುತ್ತಾರೆ; ಅದನ್ನು ಒದಗಿಸುವ ಡೇಟಾ ರಚನೆಯ ಸಾಂಪ್ರದಾಯಿಕ ಹೆಸರು _ಮ್ಯೂಟೆಕ್ಸ್_.

Go ಭಾಷೆಯ ಪ್ರಮಾಣಿತ ಲೈಬ್ರರಿಯು [[/pkg/sync/#Mutex][`sync.Mutex`]] ಹಾಗೂ ಅದರ ಎರಡು ಮೆಥಡ್‌ಗಳ ಮೂಲಕ ಪರಸ್ಪರ ಹೊರಗಿಡುವಿಕೆಯನ್ನು ಒದಗಿಸುತ್ತದೆ:

- `Lock`
- `Unlock`

`Lock` ಮತ್ತು `Unlock` ಕರೆಗಳ ನಡುವೆ ಕೋಡ್ ಬ್ಲಾಕ್ ಅನ್ನು ಇರಿಸುವ ಮೂಲಕ, `Inc` ಮೆಥಡ್‌ನಲ್ಲಿ ತೋರಿಸಿದಂತೆ, ಅದನ್ನು ಪರಸ್ಪರ ಹೊರಗಿಡುವಿಕೆಯೊಂದಿಗೆ ಕಾರ್ಯಗತಗೊಳಿಸಬಹುದು.

`defer` ಬಳಸುವುದರಿಂದ `Value` ಮೆಥಡ್‌ನಂತೆಯೇ ಮ್ಯೂಟೆಕ್ಸ್‌ನ ಲಾಕ್ ಖಚಿತವಾಗಿ ಬಿಡುಗಡೆಯಾಗುವುದನ್ನೂ ಖಚಿತಪಡಿಸಬಹುದು.

.play concurrency/mutex-counter.go
```

### 102 concurrency/10 — A

Previous B finding: English: _Hint_: you can keep a cache of the URLs; Kannada: _Hint_: ಸುಳಿವು — ಈಗಾಗಲೇ ಪಡೆಯಲಾದ URL ಗಳನ್ನು ಮ್ಯಾಪ್‌ನಲ್ಲಿ ಕ್ಯಾಶ್ ಮಾಡಬಹುದು;; Minor. Present visibly renders English italic Hint followed by Kannada ಸುಳಿವು, an unnecessary double label. Generation must localize the hint label once inside valid italic markup and preserve URL caching and concurrent-map safety.

Reviewer determination: Actual Present renders italic Kannada ಸುಳಿವು; URL cache, no-duplicate fetching and concurrent map safety all preserved.

source SHA-256: bb686393225ef2335e627819a1664a84456f00b64af133af64920d43f0abdb94
candidate SHA-256: e96c03889fd1aa497d640fbf93b3eac601914440c7ff95903db5a87fbe47054b
validation SHA-256: c590af03cb9035894ded6585a8f9745c6a9b1f153cf98915806b988ced7ed020

**Complete English Section**

```text
* Exercise: Web Crawler

In this exercise you'll use Go's concurrency features to parallelize a web crawler.

Modify the `Crawl` function to fetch URLs in parallel without fetching the same URL twice.

_Hint_: you can keep a cache of the URLs that have been fetched on a map, but maps alone are not
safe for concurrent use!

.play concurrency/exercise-web-crawler.go
```

**Complete Kannada candidate**

```text
* ಅಭ್ಯಾಸ: ವೆಬ್ ಕ್ರಾಲರ್

ಈ ಅಭ್ಯಾಸದಲ್ಲಿ Go ಭಾಷೆಯ ಸಹವರ್ತಿ ಕಾರ್ಯನಿರ್ವಹಣೆಯ ವೈಶಿಷ್ಟ್ಯಗಳನ್ನು ಬಳಸಿ ವೆಬ್ ಕ್ರಾಲರ್‌ನ ಕೆಲಸವನ್ನು ಸಮಾನಾಂತರಗೊಳಿಸುತ್ತೀರಿ.

`Crawl` ಫಂಕ್ಷನ್ ಅನ್ನು ಬದಲಿಸಿ: ಅದು URL ಗಳನ್ನು ಸಮಾನಾಂತರವಾಗಿ ಪಡೆದುಕೊಳ್ಳಬೇಕು, ಆದರೆ ಅದೇ URL ಅನ್ನು ಎರಡು ಬಾರಿ ಪಡೆಯಬಾರದು.

_ಸುಳಿವು_:  ಈಗಾಗಲೇ ಪಡೆಯಲಾದ URL ಗಳನ್ನು ಮ್ಯಾಪ್‌ನಲ್ಲಿ ಕ್ಯಾಶ್ ಮಾಡಬಹುದು; ಆದರೆ ಮ್ಯಾಪ್‌ಗಳನ್ನು ಮಾತ್ರ ಬಳಸುವುದು ಸಹವರ್ತಿ ಪ್ರವೇಶದ ಸಂದರ್ಭದಲ್ಲಿ ಸುರಕ್ಷಿತವಲ್ಲ!

.play concurrency/exercise-web-crawler.go
```

## 完整 scope 与边界

本次唯一待审 working set 为上述 13 个 Page。旧 Snapshot 的 109 个有效 A 仅按 CLI exact source/candidate/validation/attempt identity carry-forward，不重新审核、不生成 replacement；全部 19 个 Example 不重新审核。此前所有 B/C 已实际查看新版完整候选，不根据自动 validation 直接给 A。
只有在正式 reviewer-bundle-check CURRENT、preflight predecessor 精确匹配后，才使用显式 --previous-snapshot-id 完成第一次 record；全 A、B/C/D=0 和 pending=0 时由正式 quality-check finalize 生成独立 finalization.json。此 Markdown 不是机器 finalization authority。
