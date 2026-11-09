package termtheme

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKittyFamily(t *testing.T) {
	tests := map[string]string{
		"Fira Code":   "Fira Code",
		`"Fira Code"`: "Fira Code",
		"monospace":   "",
		`family="JetBrainsMono Nerd Font" style=Regular`:          "JetBrainsMono Nerd Font",
		`family='Liberation Mono' postscript_name=LiberationMono`: "Liberation Mono",
		`postscript_name=LiberationMono family='Liberation Mono'`: "Liberation Mono",
		"family=Hack style=Bold":                                  "Hack",
		"family=Hack\tstyle=Bold":                                 "Hack",
		"family=Hack   style=Bold":                                "Hack",
		"postscript_name=Foo Bar family=Hack":                     "Hack",
		`style="Bold Italic" family=Hack`:                         "Hack",
		`family="A=B"`:                                            "A=B",
		`family="Unterminated Name`:                               "Unterminated Name",
		"family=auto":                                             "",
		"postscript_name=Hack":                                    "",
	}
	for text, want := range tests {
		assert.Equal(t, want, kittyFamily(text), text)
	}
}
