package config

import (
	"encoding/xml"
	"os"
	"path/filepath"
)

type phpunitXML struct {
	XMLName    xml.Name        `xml:"phpunit"`
	TestSuites phpunitSuitesXML `xml:"testsuites"`
}

type phpunitSuitesXML struct {
	Suites []phpunitSuiteXML `xml:"testsuite"`
}

type phpunitSuiteXML struct {
	Name        string   `xml:"name,attr"`
	Directories []string `xml:"directory"`
}

// DetectPHPUnit looks for phpunit.xml or phpunit.xml.dist in the given directory
// and extracts test directories from it.
func DetectPHPUnit(dir string) (Config, error) {
	candidates := []string{
		filepath.Join(dir, "phpunit.xml"),
		filepath.Join(dir, "phpunit.xml.dist"),
	}

	var data []byte
	var err error
	for _, path := range candidates {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
	if err != nil {
		// No phpunit.xml found, return defaults
		cfg := Config{}
		cfg.applyDefaults()
		return cfg, nil
	}

	var parsed phpunitXML
	if err := xml.Unmarshal(data, &parsed); err != nil {
		// XML parse error, return defaults
		cfg := Config{}
		cfg.applyDefaults()
		return cfg, nil
	}

	var dirs []string
	for _, suite := range parsed.TestSuites.Suites {
		for _, d := range suite.Directories {
			if d != "" {
				// Ensure trailing slash
				if d[len(d)-1] != '/' {
					d += "/"
				}
				dirs = append(dirs, d)
			}
		}
	}

	cfg := Config{
		TestDirs: dirs,
	}
	cfg.applyDefaults()
	return cfg, nil
}
