package evaluation

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRenderFile(t *testing.T) {
	values := Values{
		"VAR1":      "value1",
		"VAR2":      "value2",
		"TRUE_VAR":  "true",
		"EMPTY_VAR": "",
	}

	fixtures := []struct {
		Name     string
		Input    string
		Expected string
		Error    string
	}{
		{
			Name:     "plain text",
			Input:    "abc\ndef",
			Expected: "abc\ndef",
		},
		{
			Name:     "variable with whitespace trimming",
			Input:    "abc\n{{- .VAR1 -}}\ndef",
			Expected: "abcvalue1def",
		},
		{
			Name:     "if true",
			Input:    "abc\n{{if .TRUE_VAR}}def\n{{end}}ghi",
			Expected: "abc\ndef\nghi",
		},
		{
			Name:     "if false",
			Input:    "abc\n{{if .UNDEFINED_VAR}}def\n{{end}}ghi",
			Expected: "abc\nghi",
		},
		{
			Name:     "with sprig func",
			Input:    "{{ .VAR1 | upper }}",
			Expected: "VALUE1",
		},
	}

	for _, f := range fixtures {
		t.Run(f.Name, func(t *testing.T) {
			inputFile := writeTempFile(f.Input)
			outputFile := getTempFile()
			defer deleteFile(inputFile)
			defer deleteFile(outputFile)
			err := renderFile(values, inputFile, outputFile)
			actual := readFile(outputFile)

			if f.Error != "" {
				assert.NotNil(t, err)
				assert.Equal(t, f.Error, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, f.Expected, actual)
			}
		})
	}
}
