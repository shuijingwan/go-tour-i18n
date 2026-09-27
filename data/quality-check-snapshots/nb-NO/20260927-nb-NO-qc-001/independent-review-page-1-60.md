# nb-NO Independent TranslationUnit QC — Page 1–60

- Review: first independent QC; rubric: translation-quality/v1
- Snapshot: 20260927-nb-NO-qc-001
- Predecessor: none (initial QC)
- Bundle SHA-256: 82a2a042aac0a6f0f9a3990a5c2355c6fab34374ee2d63d03b5336ee73cdb3b1
- Bundle identity: d26c86af4a7dcf6f534727cb9111fbbac213a48c0f36d782c7a509db93a3ceaf
- Coverage: complete English source and current candidate of 60 full Page units, glossary, six authority files and validation evidence
- Ratings: A=58, B=1, C=1, D=0

## Individual results

| Stable index | Unit ID | Rating |
|---:|---|:---:|
| 1 | `welcome/1` | A |
| 2 | `welcome/2` | A |
| 3 | `welcome/4` | A |
| 4 | `welcome/5` | A |
| 5 | `welcome/3` | A |
| 6 | `basics/1` | A |
| 7 | `basics/2` | A |
| 8 | `basics/3` | A |
| 9 | `basics/4` | A |
| 10 | `basics/5` | A |
| 11 | `basics/6` | A |
| 12 | `basics/7` | A |
| 13 | `basics/8` | A |
| 14 | `basics/9` | A |
| 15 | `basics/10` | A |
| 16 | `basics/11` | A |
| 17 | `basics/12` | C |
| 18 | `basics/13` | A |
| 19 | `basics/14` | A |
| 20 | `basics/15` | A |
| 21 | `basics/16` | A |
| 22 | `basics/17` | A |
| 23 | `flowcontrol/1` | A |
| 24 | `flowcontrol/2` | A |
| 25 | `flowcontrol/3` | A |
| 26 | `flowcontrol/4` | A |
| 27 | `flowcontrol/5` | A |
| 28 | `flowcontrol/6` | A |
| 29 | `flowcontrol/7` | A |
| 30 | `flowcontrol/8` | A |
| 31 | `flowcontrol/9` | A |
| 32 | `flowcontrol/10` | A |
| 33 | `flowcontrol/11` | A |
| 34 | `flowcontrol/12` | A |
| 35 | `flowcontrol/13` | A |
| 36 | `flowcontrol/14` | A |
| 37 | `moretypes/1` | A |
| 38 | `moretypes/2` | A |
| 39 | `moretypes/3` | A |
| 40 | `moretypes/4` | A |
| 41 | `moretypes/5` | A |
| 42 | `moretypes/6` | A |
| 43 | `moretypes/7` | A |
| 44 | `moretypes/8` | A |
| 45 | `moretypes/9` | A |
| 46 | `moretypes/10` | A |
| 47 | `moretypes/11` | A |
| 48 | `moretypes/12` | A |
| 49 | `moretypes/13` | A |
| 50 | `moretypes/14` | A |
| 51 | `moretypes/15` | A |
| 52 | `moretypes/16` | A |
| 53 | `moretypes/17` | A |
| 54 | `moretypes/18` | A |
| 55 | `moretypes/19` | A |
| 56 | `moretypes/20` | A |
| 57 | `moretypes/21` | A |
| 58 | `moretypes/22` | B |
| 59 | `moretypes/23` | A |
| 60 | `moretypes/24` | A |

## Findings

### C — index 17, basics/12 — mandatory zero value untranslated

**Source:** Variables declared without an explicit initial value are given their `_zero_value_`.

**Candidate:** Variabler som deklareres uten en eksplisitt startverdi, får sin `_zero_value_`.

The Go Present emphasis marker renders English “zero value” as visible text in the Norwegian sentence. The approved mandatory term is “nullverdi”. This page introduces that core concept; revise its translatable visible wording while preserving valid Present emphasis.

### B — index 58, moretypes/22 — two-value assignment terminology

**Source:** Test that a key is present with a two-value assignment:

**Candidate:** Sjekk om en nøkkel finnes, ved hjelp av en tilordning med to returverdier:

A map-index expression in a two-value assignment is not a function returning two values. Avoid “returverdier”, use precise Norwegian for a two-value assignment, preserve all Go code and the distinction between the value and presence bool.

## Scope after recording

Recorded 60/60 with 58 A, 1 B, 1 C; 62 other units await initial review. The two non-A units remain revision_required; do not start revision until all three initial groups are independently reviewed.
