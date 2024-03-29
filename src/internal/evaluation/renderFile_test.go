package evaluation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderFile(t *testing.T) {
	context := context{
		vars: varMap{
			"VAR1":      "value1",
			"VAR2":      "value2",
			"TRUE_VAR":  "true",
			"EMPTY_VAR": "",
		},
		placeholders: map[string]string{
			"projekt": "myproject",
			"PROJEKT": "MYPROJECT",
		},
	}

	fixtures := []struct {
		Name     string
		Input    string
		Mode     RenderMode
		Expected string
		Error    string
	}{
		{
			Name:     "plain text",
			Mode:     TemplateMode,
			Input:    "abc\ndef",
			Expected: "abc\ndef",
		},
		{
			Name:     "variable with whitespace trimming",
			Mode:     TemplateMode,
			Input:    "abc\n{{- .VAR1 -}}\ndef",
			Expected: "abcvalue1def",
		},
		{
			Name:     "if true",
			Mode:     TemplateMode,
			Input:    "abc\n{{if .TRUE_VAR}}def\n{{end}}ghi",
			Expected: "abc\ndef\nghi",
		},
		{
			Name:     "if false",
			Mode:     TemplateMode,
			Input:    "abc\n{{if .UNDEFINED_VAR}}def\n{{end}}ghi",
			Expected: "abc\nghi",
		},
		{
			Name:     "with sprig func",
			Mode:     TemplateMode,
			Input:    "{{.VAR1 | upper}}",
			Expected: "VALUE1",
		},
		{
			Name:     "replacements",
			Mode:     TemplateMode,
			Input:    "abcprojektdef {{.VAR1}} ABC_PROJEKT_DEF",
			Expected: "abcmyprojectdef value1 ABC_MYPROJECT_DEF",
		},
		{
			Name:     "variable without rendering",
			Mode:     CopyMode,
			Input:    "abc\n{{- .UNDEFINED_VAR -}}\ndef",
			Expected: "abc\n{{- .UNDEFINED_VAR -}}\ndef",
		},
		{
			Name:     "replacements without rendering",
			Mode:     CopyMode,
			Input:    "abcprojektdef ABC_PROJEKT_DEF",
			Expected: "abcprojektdef ABC_PROJEKT_DEF",
		},
	}

	for _, f := range fixtures {
		t.Run(f.Name, func(t *testing.T) {
			inputFile := writeTempFile(f.Input)
			outputFile := getTempFile()
			defer deleteFile(inputFile)
			defer deleteFile(outputFile)
			err := renderFile(context, inputFile, outputFile, f.Mode)
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
