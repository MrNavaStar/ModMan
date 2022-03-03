package fileutils

import (
	"archive/zip"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
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

type WriteCounter struct {
	Total int64
	Size int64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Size += int64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {

}


func DownloadFile(url string, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	total, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	counter := &WriteCounter{}
	counter.Total = int64(total)
	_, err = io.Copy(file, io.TeeReader(resp.Body, counter))
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