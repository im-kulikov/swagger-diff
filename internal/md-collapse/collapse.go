package collapse

import (
	"bytes"
	"html/template"
)

func init() {
	var err error
	tpl, err = template.
		New("collapse").
		Parse(collapseTpl)

	if err != nil {
		panic(err)
	}
}

func New(name, content string) ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := tpl.Execute(buf, struct {
		Name    string
		Content string
	}{
		Name:    name,
		Content: content,
	}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

var tpl *template.Template

const collapseTpl = `
<details>
 
	<summary>{{ .Name }}</summary>

	{{ .Content }}

</details>
`
