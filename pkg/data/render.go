package data

type TemplateDefinition struct {
	ApiVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	Metadata   Metadata   `yaml:"metadata"`
	Config     []Category `yaml:"config"`
}

type Category struct {
	Category string  `yaml:"category"`
	Routes   []Route `yaml:"routes"`
	Vars     []Var   `yaml:"vars"`
}

type Var struct {
	Name  string      `yaml:"name"`
	Value interface{} `yaml:"value"`
}

type Route struct {
	Path          string  `yaml:"path"`
	Category      string  `yaml:"-"`
	InlineCss     bool    `yaml:"inlineCss"`
	TemplatePath  string  `yaml:"templatePath"`
	MimeType      string  `yaml:"mimetype"`
	QueryPath     *string `yaml:"queryPath"`
	Vars          map[string]interface{}
	ResolvedQuery *string
}

type Metadata struct {
	Name   string  `yaml:"name"`
	Secret *string `yaml:"secret"`
}
