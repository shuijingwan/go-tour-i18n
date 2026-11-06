package i18n

import "testing"

func TestTranslationInvocationSeparatesInitialPolicy(t *testing.T) {
	c := retranslationTestCatalog(61)
	ids := []string{}
	for i, p := range c.Pages {
		u, err := c.Unit(p.ID)
		if err != nil {
			t.Fatal(err)
		}
		group := "first"
		if i >= 30 {
			group = "second"
		}
		c.Adapted = append(c.Adapted, AdaptedTranslationUnit{Unit: *u, WorkingSet: group, WorkingSetLimit: 30})
		ids = append(ids, p.ID)
	}
	if err := c.RequireInitialTranslationWorkingSet([]string{ids[0], ids[30]}); err == nil {
		t.Fatal("initial groups merged")
	}
	if err := c.RequireInitialTranslationWorkingSet(ids[30:]); err == nil {
		t.Fatal("initial per-group limit ignored")
	}
	if err := c.RequireTranslationInvocation(ids[:60]); err != nil {
		t.Fatal("shared invocation inherited initial partition/limit", err)
	}
	if err := c.RequireTranslationInvocation(ids); err == nil {
		t.Fatal("shared invocation exceeded 60")
	}
	e := retranslationTestExample("example:lesson/1.go", "_content/tour/lesson/1.go", "package main\n// Teaching comment.\n")
	c.Examples = append(c.Examples, e)
	if err := c.RequireTranslationInvocation([]string{ids[0], e.ID}); err == nil {
		t.Fatal("mixed Page/Example invocation")
	}
}

// Registering another surface makes old/new IDs readable without expanding
// the default Tour scheduling inventory or rewriting its historical batch.
func TestSurfaceRegistrationPreservesDefaultTourScheduling(t *testing.T) {
	root, c, _, _, _ := generationRecoveryFixture(t, 1)
	u, err := c.Unit(c.Pages[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	other := *u
	other.ID, other.SourcePath = "other-surface-page", "_content/doc/synthetic.md"
	c.Adapted = append(c.Adapted, AdaptedTranslationUnit{Unit: other})
	if _, err := c.Unit(other.ID); err != nil {
		t.Fatal("registered historical ID unreadable", err)
	}
	r, err := ExportRetranslationBatch(root, c, RetranslationExportOptions{Locale: "zh-CN", Generator: RetranslationGeneratorChatGPT})
	if err != nil || !r.AllExported {
		t.Fatalf("default Tour scope expanded: %+v %v", r, err)
	}
}
