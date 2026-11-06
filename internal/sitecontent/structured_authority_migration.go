package sitecontent

// Audited legacy language contract before the Call-Zero-First execution-only
// transition. This exact, locale-independent bridge approves no unknown docs
// version and no future semantic authority drift. New profiled Bundles compare
// their actual semantic authority bytes instead of this migration baseline.
var legacyStructuredAuthoritySHA = map[string]string{
	"AGENTS.md":                           "bd6fce072f4b4685e79b5856cafdaf0c84fdf4422cece07d677c3602106b4057",
	"docs/CHATGPT_LANGUAGE_GENERATION.md": "cffc8bc6d5d69ce817d3de25b215f781bf6c7bbc5a66f2ee87887f550129a4b0",
	"docs/CODEX_TRANSLATION.md":           "df0b60a26a1e7dc5444d57c342d40852fea55fadcb93ebd3c2c805795b3e5e9b",
	"docs/GLOSSARY_COMPATIBILITY.md":      "3c741bfe6548ffabd03886f0fe42b6f01ea449725875e02a35ab6158e5e61b76",
	"docs/GLOSSARY_REVIEW.md":             "3f9d8ffe9c9cc68bba4b271ca6967b5aa8ea348f54a1b19ecd6e5801be3b2a93",
	"docs/SITE_V2_ARCHITECTURE.md":        "a5f9d1aa33d28cf7d30d233727fa8a50a52a17cfdb0c9e77f84f244bba435e3c",
	"docs/SITE_V2_WORKFLOW.md":            "6ddd6e8785d337493e5d5d7e7edf0be66808b2b050d44ab5cd34c17b5d293d1b",
	"docs/TRANSLATION_TASK_SPEC.md":       "23354845e628e7ea8d9b26909a8e901202ccce4be8f1df0b42d5aae423f33023",
	"docs/TRANSLATION_WORKFLOW.md":        "8b639ad902332c87b11a91130c8163a2112c8f1b53109c69e5a986e8c7a02f56",
	"docs/WORKFLOW_HANDOFF.md":            "dcb7e757f4c3f575b691c416814edcc407d9e39e9f81b828482d798a21bd9752",
}

var legacyStructuredSemanticSHA = map[string]string{
	"docs/STRUCTURED_GENERATION_SEMANTICS.md": "edc465477ade76ee8cca46a97cc929df5223fec8e79ff6e39fbe461b6de75c7d",
	"docs/TRANSLATION_TERMINOLOGY.md":         "658ce72428d12aa910a75b898b851fc642d5233a6c3cb60c61115891d909982d",
	"docs/TERMINOLOGY_GUIDE.md":               "ea8789136105e27c05075c04dd644e882459510e7ce6e24ed0679dd864ff0d5a",
}
