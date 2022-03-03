package fileutils

import (
	"encoding/json"
	"io/ioutil"

	"github.com/mrnavastar/modman/util"
	"github.com/zalando/go-keyring"
)

type State struct {
	DotMinecraft string
	WorkDir string
	FabricInstallerVersion string
	FabricLoaderVersions []string
	ActiveInstance string
	Instances []util.Instance
}

func SaveAppState(state State) {
	dotMinecraft, err := keyring.Get("modman", "dot_minecraft")
    util.Fatal(err)

	file, err1 := json.MarshalIndent(state, "", " ")
	util.Fatal(err1)

	err2 := ioutil.WriteFile(dotMinecraft + "/modman/modman.json", file, 0644)
	util.Fatal(err2)
}

func LoadAppState() State {
	dotMinecraft, err := keyring.Get("modman", "dot_minecraft")
    util.Fatal(err)

	data, err1 := ioutil.ReadFile(dotMinecraft + "/modman/modman.json")
	util.Fatal(err1)

	var state State
	err2 := json.Unmarshal(data, &state)
	util.Fatal(err2)

	state.DotMinecraft = dotMinecraft
	state.WorkDir = dotMinecraft + "/modman"
	return state
}