package services

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mrnavastar/modman/api"
	"github.com/mrnavastar/modman/util"
	"github.com/mrnavastar/modman/util/fileutils"
	"github.com/pterm/pterm"
	"golang.org/x/mod/semver"
)

func CreateInstance(name string, version string) error {
	state := fileutils.LoadAppState()

	for _, instance := range state.Instances {
		if strings.EqualFold(instance.Name, name) {
			return errors.New("already instance with that name")
		}
	}

	var instance util.Instance
	instance.Name = name
	instance.Version = version
	instance.Path = state.WorkDir + "/instances/" + name

	flversion, err1 := api.GetLatestFabricLoaderVersion()
	if err1 != nil {
		util.Fatal(err1)
	}
	instance.FabricLoaderVersion = flversion
	state.Instances = append(state.Instances, instance)
	
	api.InstallFabricLoader(&state, version, flversion)

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
	profile.LastVersionId = "fabric-loader-" + instance.FabricLoaderVersion + "-" + version
	profile.JavaArgs = "-Xmx2G -XX:+UnlockExperimentalVMOptions -XX:+UseG1GC -XX:G1NewSizePercent=20 -XX:G1ReservePercent=20 -XX:MaxGCPauseMillis=50 -XX:G1HeapRegionSize=32M -Dfabric.addMods=" + instance.Path
	
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

func isModDownloaded(instance *util.Instance, slug string) bool {
	for _, mod := range instance.Mods {
		if mod.Slug == slug {
			return true
		}
	}
	return false
}

//Must call SaveInstance after using! - this allows for batching mod installs into one file write call
func AddMod(instance *util.Instance, arg string, modData util.ModData, isDep bool, isUpdate bool) error {
	slug := strings.Replace(arg, "c:", "", -1)

	if modData.Id == "" {
		//Check if slug is int
		if  _, err := strconv.Atoi(slug); err == nil || strings.Contains(arg, "c:") {
			m, err1 := api.GetCurseModData(slug, instance.Version)
			if err1 != nil {
				return err1
			}
			modData = m
		} else {
			m, err1 := api.GetModrinthModData(slug, instance.Version)
			if err1 != nil {
				return err1
			}
			modData = m
		}
	}

	if isModDownloaded(instance, modData.Slug) {
		return errors.New("mod already added")
	}

	file := instance.Path + "/" + modData.Filename
	fileutils.DownloadFile(modData.Url, file)
	modJson, err := fileutils.GetModJsonFromJar(file)
	util.Fatal(err)

	modData.Version = modJson.Version
	modData.IsADependency = isDep
	instance.Mods = append(instance.Mods, modData)
	
	if !isUpdate {
		pterm.Success.Println("Installed " + modData.Name)

		//Check if mod requires fabric-api
		if modJson.Depends.Fabric != "" && !isModDownloaded(instance, "fabric-api") {
			err := AddMod(instance, "fabric-api", util.ModData{}, true, false)
			if err != nil && err.Error() != "mod already added" {
				pterm.Error.Println("Failed to download dependency for " + modData.Name + ": Fabric-API")
			}
		}

		for _, project := range modData.Dependencies {
			err := AddMod(instance, project, util.ModData{}, true, false)
			if err != nil && err.Error() == "failed to get mod data" {
				pterm.Error.Println("Failed to download dependency for " + modData.Name + ": " + project)
			}
		}
	} else {
		pterm.Success.Println("Updated " + modData.Name)
	}
	return nil
}

//Must call SaveInstanceData after using! - this allows for batching mod removals into one file write call
func RemoveMod(instance *util.Instance, id string)  {
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

func SetActiveInstance(name string) {
	state := fileutils.LoadAppState()
	state.ActiveInstance = name
	fileutils.SaveAppState(state)
}

func UpdateInstance(state *fileutils.State, name string) {
	var instance util.Instance
	for _, i := range state.Instances {
		if strings.EqualFold(i.Name, name) {
			instance = i
			break
		}
	}
	
	api.InstallOrUpdateFabricInstaller()
	flVersion, err := api.GetLatestFabricLoaderVersion()
	util.Fatal(err)

	//Update fabric loader
	if semver.Compare(instance.FabricLoaderVersion, flVersion) == -1 {
		api.InstallFabricLoader(state, instance.Version, flVersion)
		instance.FabricLoaderVersion = flVersion
	}

	//Update mods
	for _, mod := range instance.Mods {
		var modData util.ModData
		if mod.Platform == "modrinth" {
			m, err := api.GetModrinthModData(modData.Slug, instance.Version)
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
			util.Fatal(AddMod(&instance, "", modData, false, true))
		}
	}	
	util.Fatal(SaveInstance(instance))
}