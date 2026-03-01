package yamlparser

import (
	"bytes"
	"log"
	"os"
	"runtime/debug"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Read merges configuration in two files to one file
func ReadYamlConfig(appFilePath string, secretFilePath string) (ret []byte, err error) {
	appFilePathEnv, ok := os.LookupEnv("APPLICATION_INJECTED_CONFIG_PATH")
	if ok {
		appFilePath = appFilePathEnv
	}

	appFile, err := os.Open(appFilePath)
	if err != nil {
		log.Fatal("ymlconfigx - failed to open injected application config file", err, debug.Stack())
		return
	}
	defer appFile.Close()

	secretFilePathEnv, ok := os.LookupEnv("APPLICATION_INJECTED_SC_PATH")
	if ok {
		secretFilePath = secretFilePathEnv
	}

	secretFile, err := os.Open(secretFilePath)
	if err != nil {
		log.Fatal("ymlconfigx - failed to open injected application secret file", err, debug.Stack())
		return
	}
	defer secretFile.Close()

	ba, err := readFile(appFile)
	if err != nil {
		log.Fatal("ymlconfigx - failed to read injected application file", err, debug.Stack())
		return
	}

	bs, err := readFile(secretFile)
	if err != nil {
		log.Fatal("ymlconfigx - failed to read injected secret file", err, debug.Stack())
		return
	}

	var a, s yaml.Node

	if err := yaml.Unmarshal(ba, &a); err != nil {
		log.Fatal("ymlconfigx - failed to unmarshal injected application file", err, debug.Stack())
		return nil, err
	}

	if err := yaml.Unmarshal(bs, &s); err != nil {
		log.Fatal("ymlconfigx - failed to unmarshal injected secret file", err, debug.Stack())
		return nil, err
	}

	if err := recursiveMerge(&a, &s); err != nil {
		log.Fatal("ymlconfigx - failed to merge injected application and secret file", err, debug.Stack())
		return nil, err
	}

	var cb bytes.Buffer

	ne := yaml.NewEncoder(&cb)
	defer ne.Close()

	if err := ne.Encode(&s); err != nil {
		log.Fatal("ymlconfigx - failed to encode output config", err, debug.Stack())
		return nil, err
	}

	ret = cb.Bytes()

	return
}

func readFile(f *os.File) ([]byte, error) {
	// get the file size
	fi, err := f.Stat()
	if err != nil {
		log.Fatal("ymlconfigx - unable to get file size", err, nil)
		return nil, err
	}

	// read the file
	b := make([]byte, fi.Size())
	_, err = f.Read(b)
	if err != nil {
		log.Fatal("ymlconfigx - unable to read file", err, nil)
		return nil, err
	}

	return b, nil
}

func nodesEqual(l, r *yaml.Node) bool {
	if l.Kind == yaml.ScalarNode && r.Kind == yaml.ScalarNode {
		return l.Value == r.Value
	}
	panic("ymlconfigx - equals on non-scalars not implemented!")
}

func recursiveMerge(from, into *yaml.Node) error {
	if from.Kind != into.Kind {
		return errors.New("cannot merge nodes of different kinds")
	}
	switch from.Kind {
	case yaml.MappingNode:
		for i := 0; i < len(from.Content); i += 2 {
			found := false
			for j := 0; j < len(into.Content); j += 2 {
				if nodesEqual(from.Content[i], into.Content[j]) {
					found = true
					if err := recursiveMerge(from.Content[i+1], into.Content[j+1]); err != nil {
						return errors.New("at key " + from.Content[i].Value + ": " + err.Error())
					}
					break
				}
			}
			if !found {
				into.Content = append(into.Content, from.Content[i:i+2]...)
			}
		}
	case yaml.SequenceNode:
		into.Content = append(into.Content, from.Content...)
	case yaml.DocumentNode:
		recursiveMerge(from.Content[0], into.Content[0])
	default:
		return errors.New("ymlconfigx - can only merge mapping and sequence nodes")
	}
	return nil
}
