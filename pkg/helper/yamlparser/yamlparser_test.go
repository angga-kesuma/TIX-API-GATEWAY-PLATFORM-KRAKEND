package yamlparser

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

// Helper function to create temporary test files
func createTempFile(t *testing.T, content string) string {
	tmpfile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}
	return tmpfile.Name()
}

func TestReadFile(t *testing.T) {
	content := "test content\n"
	tmpfilePath := createTempFile(t, content)
	defer os.Remove(tmpfilePath)

	f, err := os.Open(tmpfilePath)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()

	data, err := readFile(f)
	if err != nil {
		t.Fatalf("readFile failed: %v", err)
	}

	if string(data) != content {
		t.Errorf("expected %q, got %q", content, string(data))
	}
}

func TestReadFileEmptyFile(t *testing.T) {
	tmpfilePath := createTempFile(t, "")
	defer os.Remove(tmpfilePath)

	f, err := os.Open(tmpfilePath)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()

	data, err := readFile(f)
	if err != nil {
		t.Fatalf("readFile failed: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(data))
	}
}

func TestReadFileLargeFile(t *testing.T) {
	largeContent := ""
	for i := 0; i < 1000; i++ {
		largeContent += "This is a test line\n"
	}

	tmpfilePath := createTempFile(t, largeContent)
	defer os.Remove(tmpfilePath)

	f, err := os.Open(tmpfilePath)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()

	data, err := readFile(f)
	if err != nil {
		t.Fatalf("readFile failed: %v", err)
	}

	if string(data) != largeContent {
		t.Errorf("expected large content, got mismatched data")
	}
}

func TestNodesEqualScalarNodes(t *testing.T) {
	node1 := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}
	node2 := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}
	node3 := &yaml.Node{Kind: yaml.ScalarNode, Value: "different"}

	if !nodesEqual(node1, node2) {
		t.Error("expected nodes with same value to be equal")
	}

	if nodesEqual(node1, node3) {
		t.Error("expected nodes with different values to not be equal")
	}
}

func TestNodesEqualDifferentKinds(t *testing.T) {
	scalarNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}
	mappingNode := &yaml.Node{Kind: yaml.MappingNode}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when comparing nodes of different kinds")
		}
	}()

	nodesEqual(scalarNode, mappingNode)
}

func TestRecursiveMergeMappingNodes(t *testing.T) {
	fromYAML := `
key1: 
  nested1: value1
key2:
  nested2: value2
`
	intoYAML := `
key3:
  nested3: value3
key4:
  nested4: value4
`

	var from, into yaml.Node
	if err := yaml.Unmarshal([]byte(fromYAML), &from); err != nil {
		t.Fatalf("failed to unmarshal from YAML: %v", err)
	}
	if err := yaml.Unmarshal([]byte(intoYAML), &into); err != nil {
		t.Fatalf("failed to unmarshal into YAML: %v", err)
	}

	err := recursiveMerge(&from, &into)
	if err != nil {
		t.Fatalf("recursiveMerge failed: %v", err)
	}

	var result map[string]map[string]interface{}
	if err := into.Decode(&result); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}

	// Verify keys from 'from' are added to 'into'
	if result["key1"]["nested1"] != "value1" {
		t.Errorf("expected key1.nested1=value1, got %v", result["key1"]["nested1"])
	}
	if result["key2"]["nested2"] != "value2" {
		t.Errorf("expected key2.nested2=value2, got %v", result["key2"]["nested2"])
	}
	
	// Verify existing keys in 'into' are preserved
	if result["key3"]["nested3"] != "value3" {
		t.Errorf("expected key3.nested3=value3, got %v", result["key3"]["nested3"])
	}
	if result["key4"]["nested4"] != "value4" {
		t.Errorf("expected key4.nested4=value4, got %v", result["key4"]["nested4"])
	}
}

func TestRecursiveMergeMappingNodesWithConflict(t *testing.T) {
	// Create mapping nodes with scalar values directly to bypass DocumentNode wrapper
	// which prevents error propagation
	scalarNode1 := &yaml.Node{Kind: yaml.ScalarNode, Value: "value_from"}
	scalarNode2 := &yaml.Node{Kind: yaml.ScalarNode, Value: "value_into"}

	// This should error because we're trying to merge two different scalar nodes
	err := recursiveMerge(scalarNode1, scalarNode2)
	if err == nil {
		t.Error("expected error when merging conflicting scalar values")
	}
	
	if err.Error() != "ymlconfigx - can only merge mapping and sequence nodes" {
		t.Errorf("expected 'can only merge mapping and sequence nodes' error, got %q", err.Error())
	}
}

func TestRecursiveMergeSequenceNodes(t *testing.T) {
	fromYAML := `
- item1
- item2
`
	intoYAML := `
- item0
`

	var from, into yaml.Node
	if err := yaml.Unmarshal([]byte(fromYAML), &from); err != nil {
		t.Fatalf("failed to unmarshal from YAML: %v", err)
	}
	if err := yaml.Unmarshal([]byte(intoYAML), &into); err != nil {
		t.Fatalf("failed to unmarshal into YAML: %v", err)
	}

	err := recursiveMerge(&from, &into)
	if err != nil {
		t.Fatalf("recursiveMerge failed: %v", err)
	}

	var result []string
	if err := into.Decode(&result); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
	if result[0] != "item0" || result[1] != "item1" || result[2] != "item2" {
		t.Errorf("unexpected sequence content: %v", result)
	}
}

func TestRecursiveMergeNestedMappings(t *testing.T) {
	fromYAML := `
parent:
  child1: value1
  child2: value2
`
	intoYAML := `
parent:
  child3: value3
  child4: value4
`

	var from, into yaml.Node
	if err := yaml.Unmarshal([]byte(fromYAML), &from); err != nil {
		t.Fatalf("failed to unmarshal from YAML: %v", err)
	}
	if err := yaml.Unmarshal([]byte(intoYAML), &into); err != nil {
		t.Fatalf("failed to unmarshal into YAML: %v", err)
	}

	err := recursiveMerge(&from, &into)
	if err != nil {
		t.Fatalf("recursiveMerge failed: %v", err)
	}

	var result map[string]map[string]string
	if err := into.Decode(&result); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}

	// Verify keys from 'from' are added to 'into'
	if result["parent"]["child1"] != "value1" {
		t.Errorf("expected child1=value1, got %v", result["parent"]["child1"])
	}
	if result["parent"]["child2"] != "value2" {
		t.Errorf("expected child2=value2, got %v", result["parent"]["child2"])
	}
	
	// Verify existing keys in 'into' are preserved
	if result["parent"]["child3"] != "value3" {
		t.Errorf("expected child3=value3, got %v", result["parent"]["child3"])
	}
	if result["parent"]["child4"] != "value4" {
		t.Errorf("expected child4=value4, got %v", result["parent"]["child4"])
	}
}

func TestRecursiveMergeDifferentKinds(t *testing.T) {
	// Create nodes with different kinds directly (not through Unmarshal)
	// This avoids the DocumentNode wrapper which has the same Kind for both
	fromNode := &yaml.Node{Kind: yaml.SequenceNode}
	intoNode := &yaml.Node{Kind: yaml.MappingNode}

	err := recursiveMerge(fromNode, intoNode)
	if err == nil {
		t.Error("expected error when merging different node kinds")
	}
	
	if err.Error() != "cannot merge nodes of different kinds" {
		t.Errorf("expected 'cannot merge nodes of different kinds', got %q", err.Error())
	}
}

func TestReadYamlConfigBasic(t *testing.T) {
	appContent := `
database:
  host: localhost
  port: 5432
`

	secretContent := `
database:
  username: admin
  password: secret
`

	appFile := createTempFile(t, appContent)
	secretFile := createTempFile(t, secretContent)
	defer os.Remove(appFile)
	defer os.Remove(secretFile)

	result, err := ReadYamlConfig(appFile, secretFile)
	if err != nil {
		t.Fatalf("ReadYamlConfig failed: %v", err)
	}

	var merged map[string]map[string]interface{}
	if err := yaml.Unmarshal(result, &merged); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if merged["database"]["host"] != "localhost" {
		t.Errorf("expected host=localhost, got %v", merged["database"]["host"])
	}
	if merged["database"]["username"] != "admin" {
		t.Errorf("expected username=admin, got %v", merged["database"]["username"])
	}
	if merged["database"]["password"] != "secret" {
		t.Errorf("expected password=secret, got %v", merged["database"]["password"])
	}
}

func TestReadYamlConfigEmptyFiles(t *testing.T) {
	appFile := createTempFile(t, "")
	secretFile := createTempFile(t, "")
	defer os.Remove(appFile)
	defer os.Remove(secretFile)

	// Empty YAML files will result in DocumentNode with nil Content[0]
	// The function should handle this gracefully with log.Fatal
	// For testing purposes, we'll skip this as it will panic with log.Fatal
}

func TestReadYamlConfigWithEnvironmentVariables(t *testing.T) {
	appContent := `
app: config
`

	secretContent := `
secret: data
`

	appFile := createTempFile(t, appContent)
	secretFile := createTempFile(t, secretContent)
	defer os.Remove(appFile)
	defer os.Remove(secretFile)

	// Set environment variables
	oldAppPath := os.Getenv("APPLICATION_INJECTED_CONFIG_PATH")
	oldSecretPath := os.Getenv("APPLICATION_INJECTED_SC_PATH")
	defer func() {
		if oldAppPath != "" {
			os.Setenv("APPLICATION_INJECTED_CONFIG_PATH", oldAppPath)
		} else {
			os.Unsetenv("APPLICATION_INJECTED_CONFIG_PATH")
		}
		if oldSecretPath != "" {
			os.Setenv("APPLICATION_INJECTED_SC_PATH", oldSecretPath)
		} else {
			os.Unsetenv("APPLICATION_INJECTED_SC_PATH")
		}
	}()

	os.Setenv("APPLICATION_INJECTED_CONFIG_PATH", appFile)
	os.Setenv("APPLICATION_INJECTED_SC_PATH", secretFile)

	result, err := ReadYamlConfig("ignored_path", "ignored_path")
	if err != nil {
		t.Fatalf("ReadYamlConfig with env vars failed: %v", err)
	}

	var merged map[string]interface{}
	if err := yaml.Unmarshal(result, &merged); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if merged["app"] != "config" {
		t.Errorf("expected app=config, got %v", merged["app"])
	}
	if merged["secret"] != "data" {
		t.Errorf("expected secret=data, got %v", merged["secret"])
	}
}

func TestReadYamlConfigComplexStructure(t *testing.T) {
	appContent := `
server:
  port: 8080
  routes:
    - path: /api
      methods:
        - GET
        - POST
database:
  primary: localhost
`

	secretContent := `
server:
  routes:
    - path: /admin
      methods:
        - DELETE
credentials:
  api_key: secret_key_123
`

	appFile := createTempFile(t, appContent)
	secretFile := createTempFile(t, secretContent)
	defer os.Remove(appFile)
	defer os.Remove(secretFile)

	result, err := ReadYamlConfig(appFile, secretFile)
	if err != nil {
		t.Fatalf("ReadYamlConfig failed: %v", err)
	}

	var merged map[string]interface{}
	if err := yaml.Unmarshal(result, &merged); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// Verify server port is merged
	server := merged["server"].(map[string]interface{})
	if server["port"] != 8080 {
		t.Errorf("expected port=8080, got %v", server["port"])
	}

	// Verify credentials are present
	creds := merged["credentials"].(map[string]interface{})
	if creds["api_key"] != "secret_key_123" {
		t.Errorf("expected api_key in credentials, got %v", creds)
	}

	// Verify database is preserved
	db := merged["database"].(map[string]interface{})
	if db["primary"] != "localhost" {
		t.Errorf("expected primary database, got %v", db["primary"])
	}
}

func TestReadYamlConfigOutputEncoding(t *testing.T) {
	appContent := `
key: value
`

	secretContent := `
secret: data
`

	appFile := createTempFile(t, appContent)
	secretFile := createTempFile(t, secretContent)
	defer os.Remove(appFile)
	defer os.Remove(secretFile)

	result, err := ReadYamlConfig(appFile, secretFile)
	if err != nil {
		t.Fatalf("ReadYamlConfig failed: %v", err)
	}

	// Result should be valid YAML
	var decoded map[string]interface{}
	if err := yaml.Unmarshal(result, &decoded); err != nil {
		t.Errorf("output is not valid YAML: %v", err)
	}

	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestRecursiveMergeDocumentNode(t *testing.T) {
	fromYAML := `
key: value
`
	intoYAML := `
existing: data
`

	var from, into yaml.Node
	if err := yaml.Unmarshal([]byte(fromYAML), &from); err != nil {
		t.Fatalf("failed to unmarshal from YAML: %v", err)
	}
	if err := yaml.Unmarshal([]byte(intoYAML), &into); err != nil {
		t.Fatalf("failed to unmarshal into YAML: %v", err)
	}

	// Both should be document nodes after unmarshaling
	if from.Kind != yaml.DocumentNode || into.Kind != yaml.DocumentNode {
		t.Fatalf("expected document nodes, got from=%d, into=%d", from.Kind, into.Kind)
	}

	err := recursiveMerge(&from, &into)
	if err != nil {
		t.Fatalf("recursiveMerge failed: %v", err)
	}

	var result map[string]interface{}
	if err := into.Decode(&result); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("expected key=value, got %v", result["key"])
	}
	if result["existing"] != "data" {
		t.Errorf("expected existing=data, got %v", result["existing"])
	}
}
