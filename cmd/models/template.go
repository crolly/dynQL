package models

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/gobuffalo/flect"
)

// TemplateConfig ...
type TemplateConfig struct {
	ProjectPath string                 `yaml:"-"`
	Transform   string                 `yaml:"Transform"`
	Globals     GlobalConfig           `yaml:"Globals"`
	Resources   map[string]SAMFunction `yaml:"Resources"`
}

// GlobalConfig ...
type GlobalConfig struct {
	Function SAMFnProp `yaml:"Function"`
}

// SAMFunction ...
type SAMFunction struct {
	Type       string    `yaml:"Type"`
	Properties SAMFnProp `yaml:"Properties"`
}

// SAMFnProp ...
type SAMFnProp struct {
	Runtime     string              `yaml:"Runtime,omitempty"`
	Handler     string              `yaml:"Handler,omitempty"`
	CodeURI     string              `yaml:"CodeUri,omitempty"`
	Events      map[string]SAMEvent `yaml:"Events,omitempty"`
	Environment FnEnvironment       `yaml:"Environment,omitempty"`
}

// SAMEvent ...
type SAMEvent struct {
	Type       string  `yaml:"Type"`
	Properties SAMProp `yaml:"Properties"`
}

// SAMProp ...
type SAMProp struct {
	Path   string `yaml:"Path"`
	Method string `yaml:"Method"`
}

// FnEnvironment ...
type FnEnvironment struct {
	Variables map[string]string `yaml:"Variables,omitempty"`
}

// NewTemplate returns a new TemplateConfig
func NewTemplate(c *DQLConfig) (*TemplateConfig, error) {
	// instantiate
	t := &TemplateConfig{
		ProjectPath: c.ProjectPath,
		Transform:   "AWS::Serverless-2016-10-31",
		Globals: GlobalConfig{
			Function: SAMFnProp{
				Environment: FnEnvironment{
					Variables: map[string]string{
						"LOCAL":    "TRUE",
						"ENDPOINT": "http://dynamodb:8000",
						"REGION":   c.Region,
					},
				},
			},
		},
	}

	s, err := ReadServerlessConfig(c.ProjectPath)
	if err != nil {
		return nil, err
	}

	t.addFunctions(s)
	t.addResourceEnvs(s)

	return t, nil
}

// addFunctions adds the functions from the ServerlessConfig
func (t *TemplateConfig) addFunctions(s *ServerlessConfig) {
	if len(t.Resources) == 0 {
		t.Resources = map[string]SAMFunction{}
	}

	for n, f := range s.Functions {
		fName := flect.New(n).Camelize().String() + "Function"
		// ensure to add only http event functions
		ev := f.Events[0].HTTP
		if ev != nil {
			t.Resources[fName] = SAMFunction{
				Type: "AWS::Serverless::Function",
				Properties: SAMFnProp{
					Runtime: "go1.x",
					Handler: strings.TrimPrefix(f.Handler, "bin/"),
					CodeURI: "debug",
					Events: map[string]SAMEvent{
						"http": SAMEvent{
							Type: "Api",
							Properties: SAMProp{
								Path:   "/" + ev.Path,
								Method: ev.Method,
							},
						},
					},
				},
			}
		}
	}
}

func (t *TemplateConfig) addResourceEnvs(s *ServerlessConfig) {
	mode := os.Getenv("GRAPH_DYNAMO_MODE")
	for n := range s.Resources.Resources {
		nIdent := flect.New(strings.TrimSuffix(n, "DynamoDbTable"))
		k := nIdent.Singularize().ToUpper().String() + "_TABLE_NAME"
		v := s.Service.Name + "-" + nIdent.Pluralize().Camelize().String() + "-" + mode
		t.setEnv(k, v)
	}
}

func (t *TemplateConfig) setEnv(key, value string) {
	t.Globals.Function.Environment.Variables[key] = value
}

// Write writes the TemplateConfig to template.yml
func (t *TemplateConfig) Write() error {
	fp := filepath.Join(t.ProjectPath, "template.yml")
	// make sure directory exists
	if _, err := os.Stat(t.ProjectPath); os.IsNotExist(err) {
		if err := os.MkdirAll(t.ProjectPath, 0755); err != nil {
			return err
		}
	}

	yml, err := yaml.Marshal(t)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fp, yml, 0644)
}
