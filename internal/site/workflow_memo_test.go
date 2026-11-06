package site

import (
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

func TestWorkflowMemoPreviewCurrentness(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	TestPreviewFailureCurrentObservationAndBuildBinding(t)
}
