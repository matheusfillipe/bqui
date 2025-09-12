package clipboard

import (
	"github.com/atotto/clipboard"
)

func Copy(text string) error {
	return clipboard.WriteAll(text)
}

func Paste() (string, error) {
	return clipboard.ReadAll()
}
