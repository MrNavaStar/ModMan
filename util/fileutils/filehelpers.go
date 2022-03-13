package fileutils

import (
	"archive/zip"
	"encoding/json"
	"errors"
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

func Setup(dotMinecraft string) {
	workDir := dotMinecraft + "/modman"
    util.Fatal(keyring.Set("modman", "dot_minecraft", dotMinecraft))

	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		util.Fatal(os.MkdirAll(workDir + "/installers", 0700))
	}

	if _, err1 := os.Stat(workDir + "/modman.json"); os.IsNotExist(err1) {
		_, err2 :=os.Create(workDir + "/modman.json")
		util.Fatal(err2)
		util.Fatal(ioutil.WriteFile(workDir + "/modman.json", []byte("{}"), 0644))
	}
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


func DownloadFile(url string, filepath string) {
	resp, err := http.Get(url)
	util.Fatal(err)
	defer resp.Body.Close()

	total, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	util.Fatal(err)

	file, err := os.Create(filepath)
	util.Fatal(err)
	defer file.Close()

	counter := &WriteCounter{}
	counter.Total = int64(total)
	_, err = io.Copy(file, io.TeeReader(resp.Body, counter))
	util.Fatal(err)
}

type ModJson struct {
	Id string
	Version string
	Name string
	Description string
	Depends struct {
		FabricLoader string
		Fabric string
	}
}

func GetModJsonFromJar(filepath string) (modJson ModJson, err error) {
	reader, err := zip.OpenReader(filepath)
	util.Fatal(err)
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name == "fabric.mod.json" {
			f, err := file.Open()
			util.Fatal(err)
			defer f.Close()

			content, err1 := ioutil.ReadAll(f)
			util.Fatal(err1)

			var modJson ModJson
			err2 := json.Unmarshal([]byte(strings.Replace(string(content), "\n", "", -1)), &modJson)
			util.Fatal(err2)
			return modJson, nil
		}
	}
	return ModJson{}, errors.New("not a fabric mod")
}

func AddProfile(profile util.Profile) {
	state := LoadAppState()
	profiles, err1 := ioutil.ReadFile(state.DotMinecraft + "/launcher_profiles.json")
	util.Fatal(err1)

	data, err2 := json.MarshalIndent(profile, "", " ")
	util.Fatal(err2)

	newProfiles, err3 := jsonparser.Set(profiles, data, "profiles", profile.Name)
	util.Fatal(err3)

	util.Fatal(ioutil.WriteFile(state.DotMinecraft + "/launcher_profiles.json", newProfiles, 0644))
}

func RemoveProfile(name string) {
	state := LoadAppState()
	profiles, err1 := ioutil.ReadFile(state.DotMinecraft + "/launcher_profiles.json")
	util.Fatal(err1)

	newProfiles := jsonparser.Delete(profiles, "profiles", name)
	util.Fatal(ioutil.WriteFile(state.DotMinecraft + "/launcher_profiles.json", newProfiles, 0644))
}