package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/crolly/dynQL/cmd/helpers"

	"gopkg.in/yaml.v2"

	"github.com/gobuffalo/flect"
)

// DQLConfig ...
type DQLConfig struct {
	ProjectName string
	ProjectPath string
	Region      string
	Schemas     map[string]*Schema
	Resources   map[string]*Resource
}

// Schema ...
type Schema struct {
	Name string
	Path string
}

// Resource ...
type Resource struct {
	Ident      flect.Ident
	Attributes map[string]AttributeDefinition
}

// AttributeDefinition represents the definition of a resource's attribute
type AttributeDefinition struct {
	Ident   flect.Ident `json:"ident"`
	AwsType string      `json:"aws_type"`
}

// ReadDQLConfig ...
func ReadDQLConfig() (*DQLConfig, error) {
	wd, err := helpers.GetWorkingDir()
	if err != nil {
		return nil, err
	}
	data, err := helpers.ReadDataFromFile(filepath.Join(wd, "dql.conf.json"))
	if err != nil {
		return nil, err
	}

	var config DQLConfig
	json.Unmarshal(data, &config)

	// make sure map exists
	if len(config.Resources) == 0 {
		config.Resources = make(map[string]*Resource)
	}

	return &config, nil
}

// Write write the DQLConfig to dql.config.json in the project path
func (c DQLConfig) Write() error {
	f := filepath.Join(helpers.GetProjectPath(c.ProjectPath), "dql.conf.json")

	json, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(f, json, 0644)
}

// newServerlessConfig return a new ServerlessConfig with the attributes from the DQLConfig
func (c DQLConfig) newServerlessConfig() ServerlessConfig {
	s := newDefaultServerlessConfig()
	s.Service = Service{Name: c.ProjectName}
	s.Provider.Region = c.Region
	s.ProjectPath = helpers.GetProjectPath(c.ProjectPath)

	return s
}

// ReadServerlessConfig reads the ServerlessConfig from serverless.yml in the resource or function group directory.
// If a serverless.yml file does not exist, a new default ServerlessConfig is returned
func (c DQLConfig) ReadServerlessConfig() (*ServerlessConfig, error) {
	var sc ServerlessConfig
	data, err := helpers.ReadDataFromFile(filepath.Join(helpers.GetProjectPath(c.ProjectPath), "serverless.yml"))
	if err == nil {
		if err := yaml.Unmarshal(data, &sc); err != nil {
			return nil, err
		}
	} else if os.IsNotExist(err) {
		// file doesn't exist return default ServerlessConfig
		sc = c.newServerlessConfig()

	}

	return &sc, nil
}

// AddSchema adds a new instance of Schema to the Config
func (c *DQLConfig) AddSchema(schemaName, path string) error {
	if len(c.Schemas) == 0 {
		c.Schemas = map[string]*Schema{}
	}

	c.Schemas[schemaName] = &Schema{
		Name: schemaName,
		Path: path,
	}

	// add schema to ServerelessConfig
	s, err := c.ReadServerlessConfig()
	if err != nil {
		return err
	}

	return s.AddSchema(schemaName, path).Write()
}

// Remove removes the schema/ function from the Config
func (c *DQLConfig) Remove(name string) error {
	// remove from DQLConfig
	delete(c.Schemas, name)

	// remove from ServerlessConfig
	s, err := c.ReadServerlessConfig()
	if err != nil {
		return err
	}

	return s.RemoveFunction(name).Write()
}

// RemoveResource removes a given resource from the DQLConfig and ServerlessConfig
func (c *DQLConfig) RemoveResource(resourceName string, deleteTable bool) error {
	// remove from DQLConfig
	delete(c.Resources, resourceName)

	// delete table
	if deleteTable {
		svc := c.connectDB()
		result, err := svc.ListTables(&dynamodb.ListTablesInput{})
		if err != nil {
			return err
		}
		for _, t := range result.TableNames {
			if strings.HasPrefix(*t, c.ProjectName+"-"+flect.New(resourceName).Pluralize().Camelize().String()) {
				c.deleteTable(svc, *t)
			}
		}
	}

	// remove from ServerlessConfig
	s, err := c.ReadServerlessConfig()
	if err != nil {
		return err
	}

	return s.removeResource(resourceName).Write()
}

func (c DQLConfig) connectDB() *dynamodb.DynamoDB {
	// create service to dynamodb
	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint: aws.String("http://localhost:8000"),
		Region:   aws.String(c.Region),
	}))
	return dynamodb.New(sess)
}

// CreateResourceTables creates the tables in the local DynamoDB named by the given mode
func (c DQLConfig) CreateResourceTables(overwrite bool) error {
	svc := c.connectDB()
	// get list of tables
	result, err := svc.ListTables(&dynamodb.ListTablesInput{})
	if err != nil {
		return fmt.Errorf("Error during creation of resource tables: %s", err)
	}

	tables := make(map[string]bool)
	for _, t := range result.TableNames {
		tables[*t] = true
	}

	// iterate over resources
	s, err := c.ReadServerlessConfig()
	if err != nil {
		return err
	}
	for _, r := range c.Resources {
		mode := os.Getenv("GRAPH_DYNAMO_MODE")
		tableName := c.ProjectName + "-" + r.Ident.Pluralize().Camelize().String() + "-" + mode

		rName := r.Ident.Pascalize().String() + "DynamoDbTable"
		res := s.Resources.Resources[rName]
		if res == nil {
			return fmt.Errorf("Resource %s not valid. Please check your serverless.yml", rName)
		}
		props := res.Properties

		if tables[tableName] {
			if overwrite {
				c.deleteTable(svc, tableName)
				createTableForResource(svc, tableName, props)
			} else {
				log.Printf("Table %s already exists, skipping creation...", tableName)
			}
		} else {
			createTableForResource(svc, tableName, props)
		}
	}

	return nil
}

func createTableForResource(svc *dynamodb.DynamoDB, tableName string, props Properties) error {
	// create the table input for the resource
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
	}

	// get keySchema
	keySchema := []*dynamodb.KeySchemaElement{}
	for _, k := range props.KeySchema {
		keySchema = append(keySchema, &dynamodb.KeySchemaElement{
			AttributeName: aws.String(flect.New(k.AttributeName).Underscore().String()),
			KeyType:       aws.String(k.KeyType),
		})
	}

	// get attributes
	attributes := []*dynamodb.AttributeDefinition{}
	for _, a := range props.AttributeDefinitions {
		attributes = append(attributes, &dynamodb.AttributeDefinition{
			AttributeName: aws.String(flect.New(a.AttributeName).Underscore().String()),
			AttributeType: aws.String(a.AttributeType),
		})
	}

	// get throughput
	throughput := &dynamodb.ProvisionedThroughput{
		ReadCapacityUnits:  aws.Int64(1),
		WriteCapacityUnits: aws.Int64(1),
	}
	if props.ProvisionedThroughput != nil {
		throughput = &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(props.ProvisionedThroughput.ReadCapacityUnits),
			WriteCapacityUnits: aws.Int64(props.ProvisionedThroughput.WriteCapacityUnits),
		}
	}

	// get the local secondary indexex
	// TODO: support for non index attributes
	lsi := []*dynamodb.LocalSecondaryIndex{}
	for _, i := range props.LocalSecondaryIndexes {
		keySchema := []*dynamodb.KeySchemaElement{}
		for _, k := range i.KeySchema {
			keySchema = append(keySchema, &dynamodb.KeySchemaElement{
				AttributeName: aws.String(flect.New(k.AttributeName).Underscore().String()),
				KeyType:       aws.String(k.KeyType),
			})
		}
		lsi = append(lsi, &dynamodb.LocalSecondaryIndex{
			IndexName: aws.String(i.IndexName),
			KeySchema: keySchema,
			Projection: &dynamodb.Projection{
				ProjectionType: aws.String(i.Projection.ProjectionType),
			},
		})
	}

	gsi := []*dynamodb.GlobalSecondaryIndex{}
	for _, i := range props.GlobalSecondaryIndexes {
		keySchema := []*dynamodb.KeySchemaElement{}
		for _, k := range i.KeySchema {
			keySchema = append(keySchema, &dynamodb.KeySchemaElement{
				AttributeName: aws.String(flect.New(k.AttributeName).Underscore().String()),
				KeyType:       aws.String(k.KeyType),
			})
		}
		idx := &dynamodb.GlobalSecondaryIndex{
			IndexName: aws.String(i.IndexName),
			KeySchema: keySchema,
			Projection: &dynamodb.Projection{
				ProjectionType: aws.String(i.Projection.ProjectionType),
			},
		}
		if i.ProvisionedThroughput != nil {
			idx.ProvisionedThroughput = &dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(i.ProvisionedThroughput.ReadCapacityUnits),
				WriteCapacityUnits: aws.Int64(i.ProvisionedThroughput.WriteCapacityUnits),
			}
		} else {
			idx.ProvisionedThroughput = &dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(1),
				WriteCapacityUnits: aws.Int64(1),
			}
		}
		gsi = append(gsi, idx)
	}

	// append properties to input
	if len(keySchema) > 0 {
		input.KeySchema = keySchema
	} else {
		return errors.New("KeySchema has to be provided")
	}
	if len(attributes) == len(keySchema)+len(lsi)+len(gsi) {
		input.AttributeDefinitions = attributes
	} else {
		return errors.New("Number of attributes defined invalid. Did you add your Local Secondary Index to the Attribute Definition?")
	}
	if len(lsi) > 0 {
		input.LocalSecondaryIndexes = lsi
	}
	if len(gsi) > 0 {
		input.GlobalSecondaryIndexes = gsi
	}
	if throughput != nil {
		input.ProvisionedThroughput = throughput
	}

	out, err := svc.CreateTable(input)
	if err != nil {
		return fmt.Errorf("Error creating table %s: %s", tableName, err)
	}

	log.Printf("Table %s created: %s", tableName, out)
	return nil
}

func (c DQLConfig) deleteTable(svc *dynamodb.DynamoDB, tableName string) error {
	_, err := svc.DeleteTable(&dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})

	if err != nil {
		return fmt.Errorf("Error deleting table %s: %s", tableName, err)
	}
	return nil
}

func (c DQLConfig) renderMakefile(t *template.Template) error {
	// open file and execute template
	f, err := os.Create(filepath.Join(helpers.GetProjectPath(c.ProjectPath), "Makefile"))
	if err != nil {
		return err
	}
	defer f.Close()

	s, err := c.ReadServerlessConfig()
	if err != nil {
		return err
	}
	// execute template and save to file
	data := map[string]interface{}{
		"Functions": s.Functions,
	}

	err = t.Execute(f, data)
	if err != nil {
		return err
	}
	log.Println("Makefile generated.")
	return nil
}

func (c DQLConfig) make(path, target string, test bool) error {
	// load Makefile template
	t, err := helpers.LoadTemplateFromBox(helpers.MakeBox, "Makefile.tmpl")
	if err != nil {
		return err
	}

	// clear the debug binaries
	os.RemoveAll(filepath.Join(helpers.GetProjectPath(c.ProjectPath), "debug"))
	// render for each resource/ function group
	c.renderMakefile(t)
	// run test if flag indicates so
	if test {
		log.Println("Run tests")
		helpers.RunCmd("make", "test")
	}
	// and run the build
	helpers.RunCmd("make", target)
	return nil
}

// MakeDebug renders the Makefile and builds the debug binaries
func (c DQLConfig) MakeDebug() {
	c.make("debug", "debug", false)
}

// MakeBuild renders the Makefile and builds the binaries
func (c DQLConfig) MakeBuild(test bool) {
	c.make("bin", "build", test)
}

// RemoveFiles ...
func (c DQLConfig) RemoveFiles(name string) error {
	// function folder
	folder := filepath.Join(helpers.GetProjectPath(c.ProjectPath), "handler", name)

	return os.RemoveAll(folder)
}

// RemoveResourceFiles ...
func (c DQLConfig) RemoveResourceFiles(schemaName, resourceName string) error {
	f := resourceName + ".go"
	t := resourceName + "_test.go"
	projPath := helpers.GetProjectPath(c.ProjectPath)
	files := []string{
		filepath.Join(projPath, "models", f),
		filepath.Join(projPath, "models", t),
		filepath.Join(projPath, "services", f),
		filepath.Join(projPath, "services", t),
	}

	if len(schemaName) > 0 {
		files = append(files, filepath.Join(projPath, "handler", schemaName, "schema", f),
			filepath.Join(projPath, "handler", schemaName, "schema", t))
	}

	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			return err
		}
	}

	return nil
}
