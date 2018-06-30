package version

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/ServiceComb/go-chassis/util/fileutil"
	"gopkg.in/yaml.v2"
)

type Version struct {
	Version   string `json:"version" yaml:"version"`
	Commit    string `json:"commit" yaml:"commit"`
	Built     string `json:"built" yaml:"built"`
	GoChassis string `json:"Go-Chassis" yaml:"Go-Chassis"`
}

const (
	VersionFile    = "VERSION"
	DefaultVersion = "latest"
)

var version *Version

func setVersion() {
	v, err := getVersionSet()
	if err != nil {
		log.Printf("Get version failed, err: %s", err)
		version = &Version{}
		return
	}
	version = v
}

func getVersionSet() (*Version, error) {
	workDir, err := fileutil.GetWorkDir()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(workDir, VersionFile)
	content, err := ioutil.ReadFile(p)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		log.Printf("%s not found, mesher version unknown", p)
		return &Version{}, nil
	}
	v := &Version{}
	err = yaml.Unmarshal(content, v)
	if err != nil {
		return nil, &os.PathError{Path: p, Err: err}
	}
	return v, nil
}

func Ver() *Version {
	return version
}

func init() {
	setVersion()
}
