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

func SaveAppState(state State) error {
	dotMinecraft, err := keyring.Get("modman", "dot_minecraft")
    if err != nil {
        return err
    }

	file, err1 := json.MarshalIndent(state, "", " ")
	if err1 != nil {
		return err1
	}

	return ioutil.WriteFile(dotMinecraft + "/modman/modman.json", file, 0644)
}

func LoadAppState() (s State, e error) {
	dotMinecraft, err := keyring.Get("modman", "dot_minecraft")
    if err != nil {
        return State{}, err
    }

	data, err1 := ioutil.ReadFile(dotMinecraft + "/modman/modman.json")
	if err1 != nil {
		return State{}, err1
	}

	var state State
	err2 := json.Unmarshal(data, &state)
	if err2 != nil {
		return State{}, err2
	}

	state.DotMinecraft = dotMinecraft
	state.WorkDir = dotMinecraft + "/modman"
	return state, nil
}