package clipboard

import (
	"testing"
)

func TestCopyAndPaste(t *testing.T) {
	testText := "test.dataset.table"

	err := Copy(testText)
	if err != nil {
		t.Skipf("Clipboard not available in test environment: %v", err)
		return
	}

	result, err := Paste()
	if err != nil {
		t.Fatalf("Failed to paste from clipboard: %v", err)
	}

	if result != testText {
		t.Errorf("Expected '%s', got '%s'", testText, result)
	}
	
	t.Logf("Successfully copied and pasted: %s", testText)
}
