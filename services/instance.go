package services

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mrnavastar/modman/api"
	"github.com/mrnavastar/modman/util"
	"github.com/mrnavastar/modman/util/fileutils"
	"github.com/pterm/pterm"
	"golang.org/x/mod/semver"
)

func CreateInstance(name string, loader string, version string) error {
	state := fileutils.LoadAppState()

	for _, instance := range state.Instances {
		if strings.EqualFold(instance.Name, name) {
			return errors.New("already instance with that name")
		}
	}

	var instance util.Instance
	instance.Name = name
	instance.Loader = loader
	instance.Version = version
	instance.Path = state.WorkDir + "/instances/" + name

	if _, err := os.Stat(instance.Path); os.IsNotExist(err) {
		util.Fatal(os.MkdirAll(instance.Path, 0700))
	}

	//Create data for launcher_profiles.json
	time := time.Now().Format(time.RFC3339)
	var profile util.Profile
	profile.Name = name
	profile.Type = "custom"
	profile.Icon = "Crafting_Table"
	profile.Created = time
	profile.LastUsed = time
	profile.JavaArgs = "-Xmx2G -XX:+UnlockExperimentalVMOptions -XX:+UseG1GC -XX:G1NewSizePercent=20 -XX:G1ReservePercent=20 -XX:MaxGCPauseMillis=50 -XX:G1HeapRegionSize=32M"

	if loader == "fabric" {
		lversion, err1 := api.GetLatestFabricLoaderVersion()
		util.Fatal(err1)
		instance.LoaderVersion = lversion

		api.DownloadFabricJson(&state, version, lversion)
		profile.JavaArgs += " -Dfabric.addMods=" + instance.Path
	} else if loader == "quilt" {
		lversion := api.GetLatestQuiltLoaderVersion()
		instance.LoaderVersion = lversion

		api.DownloadQuiltJson(&state, version, lversion)
		profile.JavaArgs += " -Dloader.modsDir=" + instance.Path
	}

	state.Instances = append(state.Instances, instance)
	profile.LastVersionId = loader + "-loader-" + instance.LoaderVersion + "-" + version

	fileutils.AddProfile(profile)
	fileutils.SaveAppState(state)
	return nil
}

func DeleteInstance(name string) {
	state := fileutils.LoadAppState()

	list := state.Instances
	for i, instance := range list {
		if strings.EqualFold(instance.Name, name) {
			fileutils.RemoveProfile(name)
			util.Fatal(os.RemoveAll(instance.Path))
			SetActiveInstance("")

			//Remove Item
			list[i] = list[len(list)-1]
			state.Instances = list[:len(list)-1]
			break
		}
	}

	SetActiveInstance("")
	fileutils.SaveAppState(state)
}

func SetActiveInstance(name string) {
	state := fileutils.LoadAppState()
	state.ActiveInstance = name
	fileutils.SaveAppState(state)
}

func GetInstance(name string) (i util.Instance, e error) {
	state := fileutils.LoadAppState()

	for _, instance := range state.Instances {
		if strings.EqualFold(instance.Name, name) {
			return instance, nil
		}
	}
	return util.Instance{}, errors.New("failed to find instance")
}

func SaveInstance(instance util.Instance) error {
	state := fileutils.LoadAppState()

	for i, in := range state.Instances {
		if instance.Name == in.Name {
			state.Instances[i] = instance
			fileutils.SaveAppState(state)
			return nil
		}
	}
	return errors.New("failed to find instance")
}

func isModDownloaded(instance *util.Instance, modData util.ModData) bool {
	for _, mod := range instance.Mods {
		if mod.ProjectId == modData.ProjectId {
			return true
		}
	}
	return false
}

func GetModsRelyOn(instance *util.Instance, slug string) []string {
	var mod util.ModData
	for _, m := range instance.Mods {
		if m.Slug == slug {
			mod = m
		}
	}

	var mods []string
	for _, m := range instance.Mods {
		for _, dep := range m.Dependencies {
			if mod.ProjectId == dep.ProjectId {
				mods = append(mods, m.Name)
			}
		}
	}
	return mods
}

// AddMod Must call SaveInstance after using! - this allows for batching mod installations into one file write call
func AddMod(instance *util.Instance, arg string, modData util.ModData, isUpdate bool) error {
	slug := strings.Replace(arg, "c:", "", -1)

	if modData.Id == "" {
		//Check if slug is int
		if _, err := strconv.Atoi(slug); err == nil || strings.Contains(arg, "c:") {
			m, err1 := api.GetCurseModData(slug, instance.Version)
			if err1 != nil {
				return err1
			}
			modData = m
		} else {
			m, err1 := api.GetModrinthModData(slug, instance.Loader, instance.Version)
			if err1 != nil {
				return err1
			}
			modData = m
		}
	}

	if isModDownloaded(instance, modData) {
		return errors.New("mod already added")
	}

	file := instance.Path + "/" + modData.Filename
	fileutils.DownloadFile(modData.Url, file)
	modJson, err := fileutils.GetModJsonFromJar(file)
	util.Fatal(err)

	modData.Version = modJson.Version
	instance.Mods = append(instance.Mods, modData)

	if !isUpdate {
		pterm.Success.Println("Installed " + modData.Name)

		//Check if mod requires fabric-api
		//if modJson.Depends.Fabric != "" && !isModDownloaded(instance, "fabric-api") {
		//	err := AddMod(instance, "fabric-api", util.ModData{}, false)
		//	if err != nil && err.Error() != "mod already added" {
		//		pterm.Error.Println("Failed to download dependency for " + modData.Name + ": Fabric-API")
		//	}
		//}

		for _, dep := range modData.Dependencies {
			if dep.Required {
				err := AddMod(instance, dep.ProjectId, util.ModData{}, false)
				if err != nil && err.Error() == "failed to get mod data" {
					pterm.Error.Println("Failed to download dependency for " + modData.Name + ": " + dep.Name)
				}
			}
		}
	} else {
		pterm.Success.Println("Updated " + modData.Name)
	}
	return nil
}

// RemoveMod Must call SaveInstanceData after using! - this allows for batching mod removals into one file write call
func RemoveMod(instance *util.Instance, id string) {
	mods := instance.Mods
	for i, mod := range mods {
		if mod.Id == id {
			util.Fatal(os.Remove(instance.Path + "/" + mod.Filename))

			//Remove item
			mods[i] = mods[len(mods)-1]
			instance.Mods = mods[:len(mods)-1]
			pterm.Success.Println("Uninstalled " + mod.Name)
		}
	}
}

func UpdateInstance(state *fileutils.State, name string) {
	instance, err := GetInstance(name)
	util.Fatal(err)

	if instance.Loader == "fabric" {
		lversion, err1 := api.GetLatestFabricLoaderVersion()
		util.Fatal(err1)

		if semver.Compare(instance.LoaderVersion, lversion) == -1 {
			api.DownloadFabricJson(state, instance.Version, lversion)
			instance.Loader = lversion
		}
	} else if instance.Loader == "quilt" {
		lversion := api.GetLatestQuiltLoaderVersion()

		if semver.Compare(instance.LoaderVersion, lversion) == -1 {
			api.DownloadQuiltJson(state, instance.Version, lversion)
			instance.Loader = lversion
		}
	}

	//Update mods
	for _, mod := range instance.Mods {
		var modData util.ModData
		if mod.Platform == "modrinth" {
			m, err := api.GetModrinthModData(modData.Slug, instance.Loader, instance.Version)
			if err != nil {
				continue
			}
			modData = m
		}

		if mod.Platform == "curse" {
			m, err := api.GetCurseModData(modData.Slug, instance.Version)
			if err != nil {
				continue
			}
			modData = m
		}

		if mod.Id != modData.Id {
			RemoveMod(&instance, mod.Id)
			util.Fatal(AddMod(&instance, "", modData, true))
		}
	}
	util.Fatal(SaveInstance(instance))
}

func ExportInstance(instance util.Instance) {
	state := fileutils.LoadAppState()
	instance.Path = ""

	file, err1 := json.MarshalIndent(instance, "", " ")
	util.Fatal(err1)

	if _, err := os.Stat(state.WorkDir + "/exports/"); os.IsNotExist(err) {
		util.Fatal(os.MkdirAll(state.WorkDir+"/exports/", 0700))
	}

	err2 := ioutil.WriteFile(state.WorkDir+"/exports/"+instance.Name+".json", file, 0644)
	util.Fatal(err2)
}

func ImportInstance(file string) string {
	data, err := ioutil.ReadFile(file)
	util.Fatal(err)

	var instanceData util.Instance
	err2 := json.Unmarshal(data, &instanceData)
	util.Fatal(err2)

	CreateInstance(instanceData.Name, instanceData.Loader, instanceData.Version)
	instance, err2 := GetInstance(instanceData.Name)
	util.Fatal(err2)

	for _, mod := range instanceData.Mods {
		mod.Dependencies = nil
		AddMod(&instance, "", mod, false)
	}

	util.Fatal(SaveInstance(instance))
	return instance.Name
}

func ImportMods(instance *util.Instance, folder string) {
	var mods []string
	filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		util.Fatal(err)

		if _, err = fileutils.GetModJsonFromJar(path); err == nil {
			mods = append(mods, path)
		}

		return nil
	})

	util.Fatal(SaveInstance(*instance))
}
