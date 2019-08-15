package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/crolly/dynQL/cmd/helpers"
	"github.com/gobuffalo/flect"
)

// Model represents a resource model object
type Model struct {
	Name          string               `json:"name"`
	Type          string               `json:"type"`
	Ident         flect.Ident          `json:"ident"`
	Attributes    map[string]Attribute `json:"attributes"`
	Nested        []*Model             `json:"nested"`
	Imports       []string             `json:"imports"`
	KeySchema     map[string]string    `json:"key_schema"`
	GeneratedID   bool                 `json:"generated_id"`
	CompositeKey  bool                 `json:"composite_key"`
	BillingMode   string               `json:"billing_mode"`
	CapacityUnits map[string]int64     `json:"capacity_units"`
}

// Attribute represents a resource model's attribute
type Attribute struct {
	Name    string      `json:"name"`
	Ident   flect.Ident `json:"ident"`
	GoType  string      `json:"go_type"`
	AwsType string      `json:"aws_type"`
}

// New returns a new model object
func New(name string, slice bool, attributes string, options map[string]interface{}) (*Model, error) {
	ident := flect.New(name)
	m := &Model{
		Name:  ident.Camelize().String(),
		Ident: ident,
	}

	if slice {
		m.Type = fmt.Sprintf("[]%s", m.Ident.Pascalize())
	} else {
		m.Type = m.Ident.Pascalize().String()
	}

	// parse nested models
	attributes = m.parseNested(attributes)
	m.parseAttributes(attributes)

	// handle all option values
	var keySchema *string
	billing := "provisioned"
	capacity := map[string]int64{
		"read":  1,
		"write": 1,
	}
	if options != nil {
		if k, ok := options["keySchema"].(string); ok {
			keySchema = &k
		}
		if b, ok := options["billing"].(string); ok {
			billing = b
		}
		if c, ok := options["capacity"].(map[string]int64); ok {
			capacity = c
		}
	}

	err := m.parseKeySchema(keySchema)
	if err != nil {
		return nil, err
	}

	m.BillingMode = strings.ToLower(billing)

	if m.BillingMode == "provisioned" {
		m.CapacityUnits = capacity
	}

	return m, nil
}

// parseNested parses the attributes string for nested models
func (m *Model) parseNested(attributes string) string {
	var (
		cob    []int        // curly opening bracket slice to remember position
		cbc    = 0          // closing curly bracket counter
		sob    []int        // square opening bracket slice to remember position
		sbc    = 0          // closing square bracket counter
		rm     []string     // string slice with nested parts to remove
		clAttr = attributes // cleared attribute string without nested parts
	)
	for pos, char := range attributes {
		if char == '{' {
			// opening bracket
			cob = append(cob, pos)
		}
		if char == '}' {
			// closing bracket
			cbc++
		}
		if char == '[' {
			sob = append(sob, pos)
		}
		if char == ']' {
			sbc++
		}

		if len(cob) > 0 && len(cob) == cbc { // found single nested
			cI := m.addNested(cob, pos, attributes, false)

			// append nested part to rm slice
			rm = append(rm, attributes[cI:pos+1])

			cob = nil
			cbc = 0
		}

		if len(sob) > 0 && len(sob) == sbc { // found slice nested
			cI := m.addNested(sob, pos, attributes, true)

			// append nested part to rm slice
			rm = append(rm, attributes[cI:pos+1])

			sob = nil
			sbc = 0
		}
	}

	for _, np := range rm {
		clAttr = strings.Replace(clAttr, np, "", 1)
	}

	return clAttr
}

// addNested adds a nested model to the resource model
func (m *Model) addNested(b []int, pos int, attributes string, slice bool) int {
	// opening bracket index
	bI := b[0]
	// comma index
	cI := strings.LastIndex(attributes[0:bI-1], ",")
	if cI < 0 {
		cI = 0
	}

	// new model name ensured to not have a comma or spaces
	nmn := strings.Replace(strings.TrimSpace(attributes[cI:bI-1]), ",", "", 1)
	attr := attributes[bI+1 : pos]
	n, _ := New(nmn, slice, attr, nil)

	m.Nested = append(m.Nested, n)

	return cI
}

// parseAttributes parses all the attributes attached to a resource model
func (m *Model) parseAttributes(attrs string) {
	for _, a := range strings.Split(attrs, ",") {
		inputs := strings.Split(a, ":")
		name := inputs[0]

		// handle optional inputs
		var (
			goType = "string"
		)

		if len(inputs) > 1 {
			goType = inputs[1]
		}

		attr := Attribute{
			Name:    name,
			Ident:   flect.New(name),
			GoType:  goType,
			AwsType: helpers.AwsType(goType),
		}

		m.addImport(goType)

		m.addAttribute(attr)
	}
}

// addImport will add an import directive if the given type requires it
func (m *Model) addImport(goType string) {
	switch goType {
	case "time.Time", "*time.Time":
		m.Imports = helpers.AppendStringIfMissing(m.Imports, "time")
	case "uuid.UUID":
		m.Imports = helpers.AppendStringIfMissing(m.Imports, "github.com/gofrs/uuid")
	case "json.RawMessage":
		m.Imports = helpers.AppendStringIfMissing(m.Imports, "encoding/json")
	}
}

// GetImports recursively iterates through all import slices and adds the import to the root model
func (m *Model) GetImports() []string {
	var imports []string
	if len(m.Nested) > 0 {
		for _, n := range m.Nested {
			// get all imports of the nested model
			nI := n.GetImports()

			// iterate over imports and append new ones to imports slice
			for _, i := range nI {
				imports = helpers.AppendStringIfMissing(imports, i)
			}
		}
	}

	for _, i := range m.Imports {
		imports = helpers.AppendStringIfMissing(imports, i)
	}

	return imports
}

// addAttribute adds an attribute to a resource model
func (m *Model) addAttribute(a Attribute) {
	// make sure all attributes have names
	if a.Name != "" {
		if m.Attributes == nil {
			m.Attributes = map[string]Attribute{
				a.Name: a,
			}
		}
		m.Attributes[a.Name] = a
	}

}

// parseKeySchema parses a given keySchema and add it to the model
func (m *Model) parseKeySchema(schema *string) error {
	if schema != nil {
		for _, k := range strings.Split(*schema, ",") {
			key := strings.Split(k, ":")
			if m.KeySchema == nil {
				m.KeySchema = map[string]string{
					strings.ToUpper(key[1]): key[0],
				}
			} else {
				m.KeySchema[strings.ToUpper(key[1])] = key[0]
			}
		}

		if c, err := m.checkKeys(); !c {
			return err
		}
	}

	return nil
}

// checkKeys checks the Key Schema of the model against its attributes
func (m *Model) checkKeys() (bool, error) {
	check := map[string]byte{
		"hash":  0,
		"range": 0,
	}

	hashKey := m.KeySchema["HASH"]
	rangeKey := m.KeySchema["RANGE"]

	for _, a := range m.Attributes {
		if a.Name == hashKey {
			check["hash"]++
		}
		if a.Name == rangeKey {
			check["range"]++
		}
	}

	if check["hash"] == 0 {
		return false, fmt.Errorf("No Hash Key defined for %s. Cannot identify ID Attribute", m.Name)
	}

	if check["hash"] == 1 {
		if check["range"] >= 1 {
			m.CompositeKey = true
		}

		return true, nil
	}

	return false, fmt.Errorf("Too many keys defined in Key Schema")

}

// GetConfig returns the updated DQLConfig with the information from this Model
func (m Model) GetConfig() (*DQLConfig, error) {
	attributeDefinitions := map[string]AttributeDefinition{}
	for _, k := range m.KeySchema {
		a := m.Attributes[k]
		if len(a.Name) > 0 {
			attributeDefinitions[a.Name] = AttributeDefinition{
				Ident:   a.Ident,
				AwsType: a.AwsType,
			}
		}
	}

	// update mug.config.json
	r := &Resource{
		Ident:      flect.New(m.Name),
		Attributes: attributeDefinitions,
	}
	c, err := ReadDQLConfig()
	if err != nil {
		return nil, err
	}
	c.Resources[m.Name] = r

	return c, nil
}

// Write write the Model definition to the modelName.json
func (m Model) Write(path string) error {
	json, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(path, "functions", m.Name, fmt.Sprintf("%s.json", m.Name)), json, 0644)
}

// String prints a representation of a model
func (m Model) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("// %s defines the %s model\n", m.Ident.Pascalize(), m.Ident.Pascalize()))
	sb.WriteString(fmt.Sprintf("type %s struct {\n", m.Ident.Pascalize()))
	for _, a := range m.Attributes {
		keys := make([]string, len(m.KeySchema))
		for _, k := range m.KeySchema {
			keys = append(keys, flect.Camelize(k))
		}
		isKey := helpers.Contains(keys, flect.Camelize(a.Name))
		sb.WriteString(fmt.Sprintf("%s\n", a.String(isKey)))
	}
	if len(m.Nested) > 0 {
		sb.WriteString("\n")
		for _, n := range m.Nested {
			sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s,omitempty\" dynamo:\"%s,omitempty\"`\n", n.Ident.Pascalize(), n.Type, n.Ident.Underscore(), n.Ident.Underscore()))
		}
		sb.WriteString("}\n")
		sb.WriteString("\n")
		for _, n := range m.Nested {
			sb.WriteString(n.String())
			sb.WriteString("\n")
		}

	} else {
		sb.WriteString("}\n")
	}

	return sb.String()
}

// String returns the string representation of an attribute
func (a Attribute) String(isKey bool) string {
	omitempty := ",omitempty"
	if isKey {
		omitempty = ""
	}
	return fmt.Sprintf("\t%s %s `json:\"%s%s\" dynamo:\"%s%s\"`", a.Ident.Pascalize(), a.GoType, a.Ident.Underscore(), omitempty, a.Ident.Underscore(), omitempty)
}
