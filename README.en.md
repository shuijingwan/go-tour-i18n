# go-tour-i18n

[简体中文](README.md) | **English**

## About the project

`go-tour-i18n` is a community-maintained, unofficial project for translating, validating, synchronizing, and publishing a multilingual [A Tour of Go](https://go.dev/tour/).

The project is not maintained by Google, the Go team, or go.dev, and no affiliation, endorsement, or sponsorship is implied. Its source code and original Tour content come from the official Go project upstream.

## Live translations

Community-maintained translations currently in production:

<!-- live-locales:start -->
- [Arabic — العربية](https://ar-go-dev.shuijingwanwq.com/)
- [Bengali — বাংলা](https://bn-go-dev.shuijingwanwq.com/)
- [Brazilian Portuguese — Português (Brasil)](https://pt-go-dev.shuijingwanwq.com/)
- [Czech — Čeština](https://cs-go-dev.shuijingwanwq.com/)
- [Dutch — Nederlands](https://nl-go-dev.shuijingwanwq.com/)
- [French — Français](https://fr-go-dev.shuijingwanwq.com/)
- [German — Deutsch](https://de-go-dev.shuijingwanwq.com/)
- [Hindi — हिन्दी](https://hi-go-dev.shuijingwanwq.com/)
- [Indonesian — Bahasa Indonesia](https://id-go-dev.shuijingwanwq.com/)
- [Italian — Italiano](https://it-go-dev.shuijingwanwq.com/)
- [Japanese — 日本語](https://ja-go-dev.shuijingwanwq.com/)
- [Korean — 한국어](https://ko-go-dev.shuijingwanwq.com/)
- [Polish — Polski](https://pl-go-dev.shuijingwanwq.com/)
- [Romanian — Română](https://ro-go-dev.shuijingwanwq.com/)
- [Simplified Chinese — 简体中文](https://go-dev.shuijingwanwq.com/)
- [Spanish — Español](https://es-go-dev.shuijingwanwq.com/)
- [Swedish — Svenska](https://sv-go-dev.shuijingwanwq.com/)
- [Tamil — தமிழ்](https://ta-go-dev.shuijingwanwq.com/)
- [Telugu — తెలుగు](https://te-go-dev.shuijingwanwq.com/)
- [Thai — ไทย](https://th-go-dev.shuijingwanwq.com/)
- [Traditional Chinese — 繁體中文（台灣）](https://zh-tw-go-dev.shuijingwanwq.com/)
- [Turkish — Türkçe](https://tr-go-dev.shuijingwanwq.com/)
- [Ukrainian — Українська](https://uk-go-dev.shuijingwanwq.com/)
- [Urdu — اردو](https://ur-go-dev.shuijingwanwq.com/)
- [Vietnamese — Tiếng Việt](https://vi-go-dev.shuijingwanwq.com/)
<!-- live-locales:end -->

- Official English version: [A Tour of Go](https://go.dev/tour/)
- Bug reports and feedback: [GitHub Issues](https://github.com/shuijingwan/go-tour-i18n/issues)
- Project development blog (primarily in Chinese): [A Tour of Go multilingual translation project](https://www.shuijingwanwq.com/series/go-tour-chinese-edition-development-series/)

The current production lifecycle and public URL for every community locale are defined by [`production/identity.json`](production/identity.json), the sole machine authority for production identity.

## How the project works

The repository maintains a fixed upstream Tour baseline and a repeatable path from source to each language site:

1. synchronize and audit source changes from the official Go repository;
2. translate complete, versioned TranslationUnits using locale-specific terminology;
3. validate structure, protected content, metadata, UI text, and runnable examples;
4. perform per-TranslationUnit quality review before promotion;
5. build, verify, and publish a complete locale only after its required content is ready; and
6. continue multilingual maintenance as upstream content and locale coverage evolve.

For maintainer-level details, see the [translation workflow](docs/TRANSLATION_WORKFLOW.md), [new-locale runbook](docs/NEW_LOCALE_RUNBOOK.md), [locale surface review](docs/LOCALE_SURFACE_REVIEW.md), and [production runbook](docs/PRODUCTION_RUNBOOK.md).

## Project status and maintenance

The project is actively maintained, and its language coverage continues to expand. The live translation list above is generated from the current production identity and language registry; detailed lifecycle state remains in [`production/identity.json`](production/identity.json) rather than being duplicated here. Broader implementation and maintenance context is available in [PROJECT_STATE.md](docs/PROJECT_STATE.md).

## Upstream and third-party components

The Tour baseline is synchronized from the official [`golang/website`](https://github.com/golang/website) repository. Source provenance and synchronization policy are documented in [UPSTREAM.md](UPSTREAM.md) and [UPSTREAM_MANIFEST.tsv](UPSTREAM_MANIFEST.tsv).

Some historical frontend components are imported unchanged. Their versions and license information are recorded in [THIRD_PARTY.md](THIRD_PARTY.md).

## Support the project

If you find this project useful and would like to help keep it maintained, you can support the ongoing hosting, CDN, bandwidth, and maintenance costs.

### USDC

- Network: `Base`
- Address: `0x225f14d54683b1f5bc153bc8a678cad0277096d3`
- Minimum deposit: `0.01 USDC`

Send USDC only via the Base network.

### USDT

- Network: `Tron (TRC20)`
- Address: `TF2bM817pLQeN1Ykt3GEecRbTjuSsWtGdK`
- Minimum deposit: `0.1 USDT`

Send USDT only via the Tron (TRC20) network. Sending through the wrong asset or network may result in loss of funds.

### Exchange internal transfer

- Binance — UID: `1055351242`
- OKX — UID: `231321605530361856`

Users on the same exchange may use its internal transfer feature with the UID where available. Availability and fees depend on the exchange.

### For users in China

#### WeChat Pay

<img src="_content/images/support/wechat.png" alt="WeChat Pay QR code" width="300">

#### Alipay

<img src="_content/images/support/alipay.png" alt="Alipay QR code" width="300">

### Support reference

`go-dev-project`

If the payment service supports a note, reference, or memo field, this value can be used.

## License

The upstream source code and Tour content use the BSD-style [LICENSE](LICENSE) included in this repository. Unless stated otherwise, translations, tools, and documentation created for this project are provided under the same license. [PATENTS](PATENTS) contains the upstream patent grant, while third-party code and assets remain subject to their own licenses as documented in [THIRD_PARTY.md](THIRD_PARTY.md).
