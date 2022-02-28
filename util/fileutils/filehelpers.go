package fileutils

import (
	"archive/zip"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/mrnavastar/modman/util"
	"github.com/zalando/go-keyring"
)

func Setup(dotMinecraft string) error {
	workDir := dotMinecraft + "/modman"
	err := keyring.Set("modman", "dot_minecraft", dotMinecraft)
    if err != nil {
        return err
    }

	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		os.MkdirAll(workDir + "/installers", 0700)
	}
	if _, err1 := os.Stat(workDir + "/modman.json"); os.IsNotExist(err1) {
		os.Create(workDir + "/modman.json")
		ioutil.WriteFile(workDir + "/modman.json", []byte("{}"), 0644)
	}
	return nil
}

func DownloadFile(url string, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

type ModJson struct {
	Id string
	Version string
	Name string
	Description string
	Authors []string
}

func GetModJsonFromJar(filepath string) (modJson ModJson, err error) {
	reader, err := zip.OpenReader(filepath)
	if err != nil {
		return ModJson{}, err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name == "fabric.mod.json" {
			f, err := file.Open()
			if err != nil {
				return ModJson{}, err
			}
			defer f.Close()

			content, err1 := ioutil.ReadAll(f)
			if err1 != nil {
				return ModJson{}, err1
			}

			var modJson ModJson
			err2 := json.Unmarshal([]byte(strings.Replace(string(content), "\n", "", -1)), &modJson)
			if err2 != nil {
				return ModJson{}, err2
			}
			return modJson, nil
		}
	}
	return ModJson{}, err
}

func AddProfile(profile util.Profile) error {
	state, err := LoadAppState()
	if err != nil {
		return err
	}

	profiles, err1 := ioutil.ReadFile(state.DotMinecraft + "/launcher_profiles.json")
	if err1 != nil {
		return err1
	}

	data, err2 := json.MarshalIndent(profile, "", " ")
	if err2 != nil {
		return err2
	}

	newProfiles, err3 := jsonparser.Set(profiles, data, "profiles", profile.Name)
	if err3 != nil {
		return err3
	}

	return ioutil.WriteFile(state.DotMinecraft + "/launcher_profiles.json", newProfiles, 0644)
}

func RemoveProfile(name string) error {
	state, err := LoadAppState()
	if err != nil {
		return err
	}

	profiles, err1 := ioutil.ReadFile(state.DotMinecraft + "/launcher_profiles.json")
	if err1 != nil {
		return err1
	}

	newProfiles := jsonparser.Delete(profiles, "profiles", name)
	return ioutil.WriteFile(state.DotMinecraft + "/launcher_profiles.json", newProfiles, 0644)
}