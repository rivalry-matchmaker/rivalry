//go:build mage
// +build mage

package main

import (
	"bytes"
	_ "embed"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"

	mapset "github.com/deckarep/golang-set/v2"

	"github.com/magefile/mage/sh"
)

//go:embed templates/generate.template
var generateTemplate string

type MockFiles struct {
	File       string
	Interfaces string
}

type MockPackages struct {
	Package string
	Files   []*MockFiles
}

func (m *MockPackages) AddInterface(f, i string) {
	found := false
	for _, fi := range m.Files {
		if fi.File == f {
			fi.Interfaces += "," + i
			found = true
			break
		}
	}
	if !found {
		m.Files = append(m.Files, &MockFiles{
			File: f, Interfaces: i,
		})
	}
}

func Generate() error {
	if err := generateMocks(); err != nil {
		return err
	}
	return nil
}

func generateMocks() error {
	proto := findIName("*_grpc.pb.go")
	result, err := grepFiles("type .* interface {", proto)
	if err != nil {
		return err
	}
	re := regexp.MustCompile(`^(.*):type (.*) interface \{$`)

	interfaceMap := make(map[string]*MockPackages)
	addInterface := func(f, i string) {
		d, fx := path.Split(f)
		if _, ok := interfaceMap[d]; !ok {
			dParts := strings.Split(d, "/")
			interfaceMap[d] = &MockPackages{
				Package: dParts[len(dParts)-2],
				Files:   make([]*MockFiles, 0),
			}
		}
		interfaceMap[d].AddInterface(fx, i)
	}

	for _, res := range result {
		reOut := re.FindAllStringSubmatch(res, 1)
		addInterface(reOut[0][1], reOut[0][2])
	}

	t := template.Must(template.New("generate").Parse(generateTemplate))
	files := mapset.NewSet[string]()
	for d, mf := range interfaceMap {
		buf := bytes.NewBuffer([]byte{})
		t.Execute(buf, mf)
		f := path.Join(d, "generate.go")
		if err := writeToFile(f, buf.Bytes()); err != nil {
			return err
		}
		files.Add(f)
	}

	for _, f := range files.ToSlice() {
		if err := sh.Run("go", "generate", f); err != nil {
			return err
		}
	}
	return nil
}

func findIName(pattern string) []string {
	if out, err := sh.Output("find", ".", "-iname", pattern); err != nil {
		return nil
	} else {
		return strings.Split(out, "\n")
	}
}

func grepFiles(pattern string, files []string) ([]string, error) {
	var grepArgs []string
	grepArgs = append(grepArgs, pattern)
	grepArgs = append(grepArgs, files...)
	out, err := sh.Output("grep", grepArgs...)
	if err != nil {
		return nil, err
	}
	return strings.Split(out, "\n"), nil
}

func writeToFile(file string, data []byte) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}
