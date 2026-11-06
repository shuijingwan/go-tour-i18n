package sitecontent

import "github.com/shuijingwan/go-tour-i18n/internal/languageview"

type semanticRole = languageview.Role

const (
	naturalLanguage     = languageview.NaturalLanguage
	technicalIdentifier = languageview.TechnicalIdentifier
	codeOrLiteral       = languageview.CodeOrLiteral
	structuralToken     = languageview.StructuralToken
	urlOrTarget         = languageview.URLOrTarget
	knownTechnicalAtom  = languageview.KnownTechnicalAtom
	ambiguousFailClosed = languageview.AmbiguousFailClosed
)

func classifyProseToken(w, b, a string) semanticRole {
	return languageview.ClassifyProseTokenInRole(w, b, a, true)
}
func classifyProseTokenInRole(w, b, a string, n bool) semanticRole {
	return languageview.ClassifyProseTokenInRole(w, b, a, n)
}
func standardGoIdentity(w string) bool   { return languageview.StandardGoIdentity(w) }
func legacyMachineWord(w, t string) bool { return languageview.LegacyMachineWord(w, t) }

var identifierPosition = languageview.IdentifierPosition
var enumPosition = languageview.EnumPosition
var enumFollowing = languageview.EnumFollowing
