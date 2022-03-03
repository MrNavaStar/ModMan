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

	var instance util.Instance
	instance.Name = name
	instance.Version = version
	instance.Path = state.WorkDir + "/instances/" + name

	flversion, err1 := api.GetLatestFabricLoaderVersion()
	if err1 != nil {
		return err1
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
	
	util.Fatal(fileutils.AddProfile(profile))
	fileutils.SaveAppState(state)
	return nil
}

func DeleteInstance(name string) {
	state := fileutils.LoadAppState()

	list := state.Instances
	for i, instance := range list {
		if strings.EqualFold(instance.Name, name) { 
			util.Fatal(fileutils.RemoveProfile(name))
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

//Must call SaveInstance after using! - this allows for batching mod installs into one file write call
func AddMod(instance *util.Instance, arg string, modData util.ModData) error {
	slug := strings.Replace(arg, "c=", "", -1)

	if modData.Id == "" {
		//Check if slug is int
		if  _, err := strconv.Atoi(slug); err == nil || strings.Contains(arg, "c=") {
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

	//Check if mod has been added
	for _, mod := range instance.Mods {
		if modData.Name == mod.Name {
			return errors.New("mod already added")
		}
	}

	file := instance.Path + "/" + modData.Filename
	fileutils.DownloadFile(modData.Url, file)
	//putils.DownloadFileWithDefaultProgressbar(modData.Filename, instance.Path, modData.Url, 0700)
	modJson, err2 := fileutils.GetModJsonFromJar(file)
	util.Fatal(err2)

	modData.Version = modJson.Version
	instance.Mods = append(instance.Mods, modData)
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
	
	flVersion, err1 := api.GetLatestFabricLoaderVersion()
	util.Fatal(err1)
	api.InstallOrUpdateFabricInstaller()

	//Update fabric loader
	if semver.Compare(instance.FabricLoaderVersion, flVersion) == -1 {
		api.InstallFabricLoader(state, instance.Version, flVersion)
		instance.FabricLoaderVersion = flVersion
	}

	//Update mods
	for _, mod := range instance.Mods {
		var modData util.ModData
		if mod.Platform == "modrinth" {
			modData, err4 := api.GetModrinthModData(modData.Slug, instance.Version)
			util.Fatal(err4)

			if mod.Id != modData.Id {
				RemoveMod(&instance, mod.Id)
				util.Fatal(AddMod(&instance, "", modData))
			}
		}
	}	
	util.Fatal(SaveInstance(instance))
}