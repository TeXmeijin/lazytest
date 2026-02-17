package config

import "encoding/xml"

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

// parsePHPUnitDirs extracts test directory paths from phpunit.xml content.
func parsePHPUnitDirs(data []byte) []string {
	var parsed phpunitXML
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return nil
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
	return dirs
}
