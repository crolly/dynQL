package helpers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/gobuffalo/flect"
	"github.com/gobuffalo/packr/v2"
	"github.com/spf13/cobra"
)

var (
	// SchemaBox is the packr box containing the resource file templates
	SchemaBox = packr.New("schema", "../../templates/schema")
	// ResourceBox is the packr box containing the resource file templates
	ResourceBox = packr.New("resource", "../../templates/resource")
	// FunctionBox is the packr box containing the function file templates
	FunctionBox = packr.New("function", "../../templates/function")
	// MakeBox is the packr box containing the Makefile template
	MakeBox = packr.New("make", "../../templates/make")
	// RemoveBox is the packr box containing the schema template to be rerender when a resource is removed
	RemoveBox = packr.New("make", "../../templates/remove")
)

// GetWorkingDir get the directory the current command is run out of
func GetWorkingDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return wd, err
}

// AwsType returns the AWS datatype for a given golang type
func AwsType(s string) string {
	switch strings.ToLower(s) {
	case "string", "time.Time", "*time.Time", "uuid.UUID":
		return "S"
	case "[]string":
		return "SS"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"byte", "rune",
		"float32", "float64", "complex64", "complex128":
		return "N"
	case "[]int", "[]int8", "[]int16", "[]int32", "[]int64",
		"[]uint", "[]uint8", "[]uint16", "[]uint32", "[]uint64",
		"[]rune", "[]float32", "[]float64", "[]complex64", "[]complex128":
		return "NS"
	case "map[string]string", "map[string]int", "map[string]interface{}":
		return "M"
	case "[]byte":
		return "B"
	case "[][]byte":
		return "BS"
	case "bool":
		return "BOOL"

	default:
		return s
	}
}

// LoadTemplateFromBox loads a *text/template.Template from a packr.Box
func LoadTemplateFromBox(b *packr.Box, file string) (*template.Template, error) {
	// load string from template
	ts, err := b.FindString(file)
	if err != nil {
		return nil, err
	}

	// add FuncMap to remove bin/ for debug target
	funcMap := template.FuncMap{
		"TrimBinPrefix": func(s string) string {
			return strings.TrimPrefix(s, "bin/")
		},
		"Underscore": func(s string) string {
			return flect.Underscore(s)
		},
		"UnderscoreList": func(as interface{}) string {
			s := ""
			v := reflect.ValueOf(as)
			iter := v.MapRange()
			for iter.Next() {
				ident := iter.Value().FieldByName("Ident").Interface().(flect.Ident)
				s += "\"" + ident.Underscore().String() + "\"" + ","
			}
			return strings.TrimSuffix(s, ",")
		},
		"Camelize": func(s string) string {
			return flect.Camelize(s)
		},
		"First": func(s flect.Ident) string {
			return string(s.String()[0])
		},
		"Pascalize": func(s string) string {
			return flect.Pascalize(s)
		},
	}

	// create new template with string
	t, err := template.New(file).Funcs(funcMap).Parse(ts)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// AppendStringIfMissing appends an element to a slice, if it doesn't contain the element already
func AppendStringIfMissing(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

// RunCmd will run an OS command with the given arguments
func RunCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)

	return execCmd(cmd)
}

// RunCmdWithEnv will run an OS command with the given arguments and an environment
func RunCmdWithEnv(envs []string, name string, args ...string) error {
	cmdEnv := append(os.Environ(), envs...)
	cmd := exec.Command(name, args...)
	cmd.Env = cmdEnv

	return execCmd(cmd)
}

func execCmd(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ReadDataFromFile reads the contents of a file at the given path
func ReadDataFromFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// GetFuncName returns the generated function name for a given resource/ function group name and a functionName
func GetFuncName(resourceName, functionName string) string {
	ident := flect.New(resourceName)
	if ident.String() == "_" {
		return functionName
	}

	return functionName + "_" + ident.Singularize().String()

}

// Contains checks whether a string slice contains a given string
func Contains(s []string, v string) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}
	return false
}

// GetList returns the list of deployable/ debugable resources/ function groups
func GetList(projectPath, wish string) ([]string, error) {
	var available []string
	// list of all resources and function groups available in project
	info, err := ioutil.ReadDir(filepath.Join(projectPath, "functions"))
	if err != nil {
		return nil, err
	}

	for _, f := range info {
		if f.IsDir() {
			available = append(available, f.Name())
		}
	}

	var list []string
	if wish == "all" {
		list = available
	} else {
		// split list of resources/ function groups
		wishList := strings.Split(wish, ",")
		for _, r := range available {
			if Contains(wishList, r) {
				list = append(list, r)
			}
		}
	}

	return list, nil
}

// CreateLambdaNetwork spins up a lambda network for dynamodb and AWS SAM to interact with one another
func CreateLambdaNetwork() error {
	// check if network exists
	out, err := exec.Command("docker", "network", "ls", "--filter", "name=^lambda-local$", "--format", "{{.Name}}").Output()
	if err != nil {
		return err
	}
	// create network if it doesn't exist
	if len(out) == 0 {
		log.Println("Creating lambda-local docker network")
		RunCmd("docker", "network", "create", "lambda-local")
	} else {
		log.Println("Docker network lambda-local already exists, skipping creation...")
	}

	return nil
}

// StartLocalDynamoDB spins up the dynamodb-local docker image
func StartLocalDynamoDB() error {
	// check if container exists
	out, err := exec.Command("docker", "ps", "-a", "--filter", "network=lambda-local", "--filter", "ancestor=amazon/dynamodb-local", "--filter", "name=dynamodb", "--format", "{{.Status}}").Output()
	if err != nil {
		return err
	}

	if strings.HasPrefix(string(out), "Exited") {
		log.Println("Restarting dynamodb-local container...")
		RunCmd("docker", "restart", "dynamodb")
	}

	// create container if it doesn't exist already
	if len(out) == 0 {
		log.Println("Starting dynamodb-local...")
		wd, err := GetWorkingDir()
		if err != nil {
			return err
		}
		RunCmd("docker", "run", "-v", fmt.Sprintf("%s:/dynamodb_local_db", wd), "-p", "8000:8000", "--net=lambda-local", "--name", "dynamodb", "-d", "amazon/dynamodb-local")
	}

	log.Println("dynamodb-local running.")
	return nil
}

// RenderFile renders a template to the file at folder/fName with the given data
func RenderFile(box *packr.Box, fName, tPath, folder string, data map[string]interface{}) error {
	f, err := os.Create(filepath.Join(folder, fName))
	if err != nil {
		return err
	}
	defer f.Close()

	// load template
	tmpl, err := LoadTemplateFromBox(box, tPath)
	if err != nil {
		return err
	}

	err = tmpl.Execute(f, data)
	if err != nil {
		return err
	}

	return nil
}

// ExecuteCommand executes a command from anywhere
func ExecuteCommand(cmd *cobra.Command, args ...string) (output string, err error) {
	_, output, err = executeCommandC(cmd, args...)
	return output, err
}

func executeCommandC(cmd *cobra.Command, args ...string) (c *cobra.Command, output string, err error) {
	buf := new(bytes.Buffer)
	cmd.SetOutput(buf)
	cmd.SetArgs(args)
	c, err = cmd.ExecuteC()
	return c, buf.String(), err
}
