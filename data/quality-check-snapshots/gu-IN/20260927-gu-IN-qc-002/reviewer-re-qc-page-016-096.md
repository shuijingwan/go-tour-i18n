# gu-IN 定向 TranslationUnit re-QC：6 个 Page

- Locale: gu-IN; rubric: translation-quality/v1; independent Reviewer: original gu-IN ChatGPT GPT-5.6 Sol High (no Generation/revision/replacement).
- Previous full Snapshot: 20260927-gu-IN-qc-001.
- Current full Snapshot: 20260927-gu-IN-qc-002.
- Page revision batch: chatgpt-gu-IN-004; validation: 6/6 passed.
- Reviewer ZIP: /tmp/gu-IN-20260927-gu-IN-qc-002-re-qc-page-6.zip.
- ZIP SHA-256: ec206c07fd868b386c882d26c87da969caf599042f4888cc17287c6cb153f5c4
- Bundle identity: 97edb5444fd4a4ebe244c5c9a6bb39da96d5d8a0f8601a2d6ec4b4958aa12bd9
- Snapshot manifest SHA-256: 746f1ee72a7a9635e31f7ce1d03df15e6979ab31b117087353c7439fdca9b236
- Full glossary SHA-256: 9e9698f4281fb2be23f5532d3e6804f85eec5dbc25759044ce8b233d7e85c377
- Complete bundle inventory: all 22 declared SHA-256 and sizes PASS; 23 ZIP entries including manifest; ZIP CRC PASS.
- Repository formal reviewer-bundle-check: CURRENT.
- Formal preflight: incremental, predecessor=20260927-gu-IN-qc-001, 114 old A carried forward, 8 pending, first record not started.
- Reviewed scope: only revised Page indexes 16, 37, 51, 73, 83, 96. Historical A (114) not reassessed; Example 107/108 not reviewed.
- Reviewer conclusion: six complete Page Units independently checked against complete English source, updated candidate, glossary, batch previous finding and validation; all A. No new actionable issue.

## Six individual ratings

| Stable index | Unit ID | Previous rating | Current rating | Validation |
| ---: | --- | :---: | :---: | :---: |
| 16 | basics/11 | C | **A** | passed |
| 37 | moretypes/1 | C | **A** | passed |
| 51 | moretypes/15 | B | **A** | passed |
| 73 | methods/10 | C | **A** | passed |
| 83 | methods/20 | B | **A** | passed |
| 96 | concurrency/4 | B | **A** | passed |

## Each complete Page, previous finding and independent re-review

### #16 basics/11 — A (previous C)

- Previous finding: Both teaching comments invert the type-alias relationship: byte/uint8 and rune/int32. Correct Gujarati alias direction while preserving protected code and comment structure.
- Source SHA-256: 836ab1dd659798dcf5e90f283c4ee794aed1d2cf0766c2ccc268fa91684e9d05
- Candidate SHA-256: 0c79bab9261cbbc2a04a6e980e749c0e88ed6c6c40717c8c9d5874c46710e8cb
- Validation SHA-256: 1a8f9fb820cefaf50e3751de8da810dffad64a31b15b142eeadfea6be1e2c46b
- Revision input SHA-256: c2557646ac0084dfa79046a92ee6c3878d0207f6c105bafc8f9013085ebc7aa8
- Batch/attempt: chatgpt-gu-IN-004/1
- Independent review: A: byte is explicitly the alias for uint8; rune is explicitly the alias for int32. Unicode code-point explanation, widths and int advice preserved. Code and comment markers unchanged.

**Complete English Page:**

```text
* Basic types

Go's basic types are

	bool

	string

	int  int8  int16  int32  int64
	uint uint8 uint16 uint32 uint64 uintptr

	byte // alias for uint8

	rune // alias for int32
	     // represents a Unicode code point

	float32 float64

	complex64 complex128

The example shows variables of several types,
and also that variable declarations may be "factored" into blocks,
as with import statements.

The `int`, `uint`, and `uintptr` types are usually 32 bits wide on 32-bit systems and 64 bits wide on 64-bit systems.
When you need an integer value you should use `int` unless you have a specific reason to use a sized or unsigned integer type.

.play basics/basic-types.go
```

**Complete revised Gujarati Page:**

```text
* મૂળભૂત ટાઇપ

Go ની મૂળભૂત ટાઇપ નીચે મુજબ છે:

	bool

	string

	int  int8  int16  int32  int64
	uint uint8 uint16 uint32 uint64 uintptr

	byte // uint8 ટાઇપનું ઉપનામ છે

	rune // int32 ટાઇપનું ઉપનામ છે
	     // યુનિકોડ કોડ પોઇન્ટ દર્શાવે છે

	float32 float64

	complex64 complex128

આ ઉદાહરણમાં અનેક ટાઇપના વેરિએબલ દર્શાવ્યા છે.
ઇમ્પોર્ટ સ્ટેટમેન્ટની જેમ વેરિએબલની ઘોષણાઓને પણ બ્લોકમાં જૂથબદ્ધ કરી શકાય છે.

32-બિટ સિસ્ટમમાં `int`, `uint` અને `uintptr` ટાઇપ સામાન્ય રીતે 32 બિટની હોય છે; 64-બિટ સિસ્ટમમાં તે સામાન્ય રીતે 64 બિટની હોય છે.
તમને પૂર્ણાંક મૂલ્યની જરૂર હોય તો નિશ્ચિત કદની અથવા unsigned પૂર્ણાંક ટાઇપ વાપરવાનું કોઈ ખાસ કારણ ન હોય ત્યાં સુધી `int` વાપરો.

.play basics/basic-types.go
```

### #37 moretypes/1 — A (previous C)

- Previous finding: Both pointer teaching comments glue identifier p to Gujarati verbs, producing વાંચોp and આપોp. Restore the explicit relation between pointer p and reading/writing i without changing protected code.
- Source SHA-256: 7706eaf0d46acc79fdad3b7953e5571bf404551d71aaefafd8ba79f5761f9806
- Candidate SHA-256: f36f322c0224093601d4bc974e98f4cb11d0ac37c52337dfc650b0ecaed817e9
- Validation SHA-256: a0a3af842ec62ab87518766f9ac754d135f4792842661af08e601c08cd56ad43
- Revision input SHA-256: b6b916daff7c87dfb9553a2c0e739f4ce3004aa40023a62d15e19ad5a693e67c
- Batch/attempt: chatgpt-gu-IN-004/1
- Independent review: A: Both p comments clearly say pointer p reads i and assigns a value to i; natural Gujarati word order. Pointer address, nil, dereference and Go no-pointer-arithmetic explanations preserved.

**Complete English Page:**

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

**Complete revised Gujarati Page:**

```text
* પોઇન્ટર

Go માં પોઇન્ટર ઉપલબ્ધ છે.
પોઇન્ટર કોઈ મૂલ્યનું મેમરી સરનામું રાખે છે.

`*T` ટાઇપ એ `T` ટાઇપના મૂલ્યનો પોઇન્ટર છે. તેનું શૂન્ય મૂલ્ય `nil` છે.

	var p *int

`&` ઑપરેટર તેના ઑપરેન્ડનો પોઇન્ટર બનાવે છે.

	i := 42
	p = &i

`*` ઑપરેટર પોઇન્ટર જે મૂલ્ય તરફ નિર્દેશ કરે છે તે દર્શાવે છે.

	fmt.Println(*p) // પોઇન્ટર p મારફતે i વાંચો
	*p = 21         // પોઇન્ટર p મારફતે i ને મૂલ્ય સોંપો

આને "ડિરેફરન્સિંગ" અથવા "ઇનડાયરેક્ટિંગ" કહેવામાં આવે છે.

C થી વિપરીત, Go માં પોઇન્ટર અંકગણિતની સુવિધા નથી.

.play moretypes/pointers.go
```

### #51 moretypes/15 — A (previous B)

- Previous finding: The variadic values supplied to append are described as multiple formal parameters. Clarify parameter versus argument semantics for vs ...T.
- Source SHA-256: f2fc257e7c5f118321acc3b75e946053601532d411e8540534280d7fba795289
- Candidate SHA-256: 7c752fa0a71746d9b678a753b45b20a48808663eba94d1bb0fb626283ea86d22
- Validation SHA-256: 9ea9c9febd1537762659b70cb41ee886cad34b01cb2fbc27acc1efa331b03c32
- Revision input SHA-256: 6088b687ef801ed9c0eb38c3537cfb2d2e57037de2160d81b0f4a9aec96b8dbe
- Batch/attempt: chatgpt-gu-IN-004/1
- Independent review: A: first formal parameter s (પેરામીટર) and remaining supplied values as arguments (આર્ગ્યુમેન્ટ) distinguished. Exact append signature, allocation semantics and hyperlinks preserved.

**Complete English Page:**

```text
* Appending to a slice

It is common to append new elements to a slice, and so Go provides a built-in
`append` function. The [[/pkg/builtin/#append][documentation]]
of the built-in package describes `append`.

	func append(s []T, vs ...T) []T

The first parameter `s` of `append` is a slice of type `T`, and the rest are
`T` values to append to the slice.

The resulting value of `append` is a slice containing all the elements of the
original slice plus the provided values.

If the backing array of `s` is too small to fit all the given values a bigger
array will be allocated. The returned slice will point to the newly allocated
array.

(To learn more about slices, read the [[/blog/go-slices-usage-and-internals][Slices: usage and internals]] article.)

.play moretypes/append.go
```

**Complete revised Gujarati Page:**

```text
* સ્લાઇસમાં નવા ઘટકો ઉમેરવા

સ્લાઇસમાં નવા ઘટકો ઉમેરવાની ઘણી વાર જરૂર પડે છે. તેથી Go બિલ્ટ-ઇન `append` ફંક્શન આપે છે. બિલ્ટ-ઇન પેકેજનું [[/pkg/builtin/#append][દસ્તાવેજીકરણ]] `append` વિશે સમજાવે છે.

	func append(s []T, vs ...T) []T

`s` એ `append` નો પ્રથમ પેરામીટર છે અને `T` ટાઇપના ઘટકો ધરાવતી સ્લાઇસ છે. બાકીનાં આર્ગ્યુમેન્ટ તરીકે સ્લાઇસમાં ઉમેરવાનાં `T` ટાઇપનાં મૂલ્યો આપવામાં આવે છે.

`append` નું પરિણામ મૂળ સ્લાઇસના બધા ઘટકો તથા આપેલા વધારાના મૂલ્યો ધરાવતી સ્લાઇસ છે.

જો `s` નું આધારરૂપ એરે આપેલા બધા મૂલ્યો સમાવવા માટે નાનું હોય, તો વધુ મોટું એરે ફાળવવામાં આવે છે. પરત મળેલી સ્લાઇસ નવા ફાળવેલા એરે તરફ નિર્દેશ કરે છે.

(સ્લાઇસ વિશે વધુ જાણવા [[/blog/go-slices-usage-and-internals][સ્લાઇસ: ઉપયોગ અને આંતરિક રચના]] લેખ વાંચો.)

.play moretypes/append.go
```

### #73 methods/10 — A (previous C)

- Previous finding: Go has no implements keyword and no explicit declaration of interface implementation. The candidate only says the keyword need not be used, implying optional syntax. Explicitly state that the keyword does not exist while preserving implicit interface implementation.
- Source SHA-256: 11ca99e6e83d93827daf02b177156fcb6f3c94ac8c202491dfffc5840ce4e39f
- Candidate SHA-256: fb1d635b9b64903bef73579e228f1d60145e187e2158b2924692970636265124
- Validation SHA-256: 03f5aa5c42b44b5059c69ae5f9b533026a7fcfd119bc80becb916d3badf6faed
- Revision input SHA-256: fa48cb3504311f34589bfc2fc2cae535e37f0968803a164275445a77dbb6e5e8
- Batch/attempt: chatgpt-gu-IN-004/1
- Independent review: A: explicitly says no implements keyword exists in Go, while preserving implicit interface satisfaction and independence of definition from implementation.

**Complete English Page:**

```text
* Interfaces are implemented implicitly

A type implements an interface by implementing its methods.
There is no explicit declaration of intent, no "implements" keyword.

Implicit interfaces decouple the definition of an interface from its
implementation, which could then appear in any package without prearrangement.

.play methods/interfaces-are-satisfied-implicitly.go
```

**Complete revised Gujarati Page:**

```text
* ઇન્ટરફેસનું અમલીકરણ સ્પષ્ટ જાહેર કરવું પડતું નથી

કોઈ ટાઇપ ઇન્ટરફેસની મેથડનું અમલીકરણ કરીને તે ઇન્ટરફેસનું અમલીકરણ કરે છે.
આ માટે ઇરાદો દર્શાવતી કોઈ સ્પષ્ટ ઘોષણા હોતી નથી; Go માં "implements" નામનો કીવર્ડ પણ નથી.

આ રીતે ઇન્ટરફેસનું સ્પષ્ટ ઘોષણા વગરનું અમલીકરણ તેની વ્યાખ્યાને
અમલીકરણથી સ્વતંત્ર રાખે છે; અમલીકરણ અગાઉથી કોઈ ગોઠવણ કર્યા વિના
કોઈપણ પેકેજમાં લખી શકાય છે.

.play methods/interfaces-are-satisfied-implicitly.go
```

### #83 methods/20 — A (previous B)

- Previous finding: In ordinary prose for Sqrt returning a non-nil error value, the candidate leaves error untranslated against mandatory glossary ભૂલનું મૂલ્ય. Use the glossary term while retaining protected Go identifiers and all code.
- Source SHA-256: 41f1f73320fde60ee5ff30d5927a19ff22d6da6a336bb776e15f2499e4f421d8
- Candidate SHA-256: d17993b6010666dc624541a1f0a8ebd798e2a3ad3bf1305a9e3d69444ff209da
- Validation SHA-256: 85702a109540d18dc7fcfcee7bf9e8179e93abc4e9536ccb2587c5acd4d9691d
- Revision input SHA-256: 79601176789b6547ba267bb636ca302742ba87da73a5763ea8cc678f810c8e9c
- Batch/attempt: chatgpt-gu-IN-004/1
- Independent review: A: ordinary prose now uses mandatory glossary form ભૂલનું મૂલ્ય; protected Go type error is unchanged. Non-nil negative result, ErrNegativeSqrt method, infinite recursion warning and float64 conversion are accurate.

**Complete English Page:**

```text
* Exercise: Errors

Copy your `Sqrt` function from the [[/tour/flowcontrol/8][earlier exercise]] and modify it to return an `error` value.

`Sqrt` should return a non-nil error value when given a negative number, as it doesn't support complex numbers.

Create a new type

	type ErrNegativeSqrt float64

and make it an `error` by giving it a

	func (e ErrNegativeSqrt) Error() string

method such that `ErrNegativeSqrt(-2).Error()` returns `"cannot`Sqrt`negative`number:`-2"`.

*Note:* A call to `fmt.Sprint(e)` inside the `Error` method will send the program into an infinite loop. You can avoid this by converting `e` first: `fmt.Sprint(float64(e))`. Why?

Change your `Sqrt` function to return an `ErrNegativeSqrt` value when given a negative number.

.play methods/exercise-errors.go
```

**Complete revised Gujarati Page:**

```text
* અભ્યાસ: ભૂલો

તમારું `Sqrt` ફંક્શન [[/tour/flowcontrol/8][અગાઉના અભ્યાસ]] માંથી કૉપિ કરો અને
તે `error` મૂલ્ય પણ પરત કરે તે રીતે બદલો.

`Sqrt` ને નકારાત્મક સંખ્યા આપવામાં આવે ત્યારે તેણે nil ન હોય એવું ભૂલનું મૂલ્ય
પરત કરવું જોઈએ, કારણ કે તે કોમ્પ્લેક્સ સંખ્યાઓને સપોર્ટ કરતું નથી.

એક નવી ટાઇપ બનાવો:

	type ErrNegativeSqrt float64

અને નીચેની મેથડ આપીને તેને `error` બનાવો:

	func (e ErrNegativeSqrt) Error() string

જેથી `ErrNegativeSqrt(-2).Error()` `"cannot`Sqrt`negative`number:`-2"` પરત કરે.

*નોંધ:* `fmt.Sprint(e)` ને `Error` મેથડની અંદર કૉલ કરવાથી
પ્રોગ્રામ અનંત લૂપમાં ફસાઈ જશે. પહેલાં `e` નું રૂપાંતરણ કરીને
`fmt.Sprint(float64(e))` વાપરવાથી તે ટાળી શકાય છે. શા માટે?

નકારાત્મક સંખ્યા આપવામાં આવે ત્યારે તમારું `Sqrt` ફંક્શન
`ErrNegativeSqrt` મૂલ્ય પરત કરે તે રીતે બદલો.

.play methods/exercise-errors.go
```

### #96 concurrency/4 — A (previous B)

- Previous finding: Go Present renders the bold candidate બીજી_નોંધ: with a visible internal underscore, whereas the English bold source renders Another note: with a word space. Restore natural spacing in the bold text while preserving the emphasis and protected structure.
- Source SHA-256: 3e925c6745255a443c61f25cd8131d538dc8677938f1c5021ce798465020b42e
- Candidate SHA-256: 72d315c1d3f139c470c22b8c179071c92fc012d16ca1f10a6d307e883cf7da37
- Validation SHA-256: adb1697d1fee86f46770a179b777b144956481e538ba713a8ea8f63326871855
- Revision input SHA-256: f68dc9ef06de3cb5d48846ebe7786e6cc06b77ea5c05bc6d56b80f667f5dad10
- Batch/attempt: chatgpt-gu-IN-004/1
- Independent review: A: real repository-pinned present.Style renders revised *બીજી*નોંધ:* as bold બીજી નોંધ: with a space and no visible underscore. Channel closure, receive ok, panic and receiver termination remain accurate.

**Complete English Page:**

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

**Complete revised Gujarati Page:**

```text
* Range અને Close

હવે વધુ મૂલ્યો મોકલવામાં નહીં આવે તે દર્શાવવા મોકલનાર ચેનલને `close` કરી શકે છે.
મેળવનાર receive અભિવ્યક્તિનું બીજું મૂલ્ય લઈને ચેનલ બંધ થઈ છે કે નહીં
તે તપાસી શકે છે: નીચેના સ્ટેટમેન્ટ પછી

	v, ok := <-ch

જો મેળવવા માટે કોઈ મૂલ્ય બાકી ન હોય અને ચેનલ બંધ હોય, તો `ok` `false` હોય છે.

`for`i`:=`range`c` લૂપ ચેનલ બંધ ન થાય ત્યાં સુધી તેમાંથી વારંવાર મૂલ્યો મેળવે છે.

*નોંધ:* ચેનલ માત્ર મોકલનારે જ બંધ કરવી જોઈએ, મેળવનારે ક્યારેય નહીં.
બંધ ચેનલ પર મૂલ્ય મોકલવાથી panic સર્જાશે.

*બીજી*નોંધ:* ચેનલ ફાઇલ જેવી નથી; સામાન્ય રીતે તેને બંધ કરવાની જરૂર હોતી નથી.
જ્યારે મેળવનારને આગળ કોઈ મૂલ્ય નહીં આવે તેની જાણ કરવી જરૂરી હોય,
જેમ કે `range` લૂપ સમાપ્ત કરવા, ત્યારે જ ચેનલ બંધ કરવી જરૂરી છે.

.play concurrency/range-and-close.go
```

## Locked Go Present renderer test

- Old raw markup: *બીજી_નોંધ:* -> rendered HTML contains visible underscore: <b>બીજી_નોંધ:</b>.
- New raw markup: *બીજી*નોંધ:* -> rendered HTML: <b>બીજી નોંધ:</b>.
- English raw markup: *Another*note:* -> rendered HTML: <b>Another note:</b>.
- These results were checked by actual present.Style under the current repository; no parser guess or false-positive underscore finding.

## Formal handoff and stop

- A=6, B=0, C=0, D=0 for Page re-QC. No translation or replacement produced; formal quality-check result NOT recorded by this Reviewer.
- Original gu-IN Generation role or maintainer Local terminal must first run current-check and incremental preflight, then record exactly these 6 A on current Snapshot; the FIRST record MUST pass --previous-snapshot-id 20260927-gu-IN-qc-001.
- After successful record, export current two-Example Reviewer ZIP 107 and 108 for separate independent re-QC; do not finalize until all 122 effective A and pending=0.
