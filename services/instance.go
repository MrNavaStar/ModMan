package services

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mrnavastar/modman/api"
	"github.com/mrnavastar/modman/util"
	"github.com/mrnavastar/modman/util/fileutils"
	"golang.org/x/mod/semver"
)

func CreateInstance(name string, version string) error {
	state, err := fileutils.LoadAppState()
	if err != nil {
		return err
	}

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
	
	err2 := api.InstallFabricLoader(&state, version, flversion)
	if err2 != nil {
		return err2
	}

	if _, err := os.Stat(instance.Path); os.IsNotExist(err) {
		os.MkdirAll(instance.Path, 0700)
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
	
	err3 := fileutils.AddProfile(profile)
	if err3 != nil {
		return err3
	}

	err4 := fileutils.SaveAppState(state)
	if err4 != nil {
		return err4
	}
	return nil
}

func DeleteInstance(name string) error {
	state, err := fileutils.LoadAppState()
	if err != nil {
		return err
	}

	list := state.Instances
	for i, instance := range list {
		if strings.EqualFold(instance.Name, name) { 
			err1 := fileutils.RemoveProfile(name)
			if err1 != nil {
				return err1
			}

			err2 := os.RemoveAll(instance.Path)
			if err2 != nil {
				return err2
			}

			err3 := SetActiveInstance("")
			if err3 != nil {
				return err3
			}

			//Remove Item
			list[i] = list[len(list)-1]
    		state.Instances = list[:len(list)-1]
			break
		}
	}
	return fileutils.SaveAppState(state)
}

func GetInstance(name string) (i util.Instance, e error) {
	state, err := fileutils.LoadAppState()
	if err != nil {
		return util.Instance{}, err
	}

	for _, instance := range state.Instances {
		if strings.EqualFold(instance.Name, name) {
			return instance, nil
		}
	}
	return util.Instance{}, errors.New("failed to find instance")
}

func SaveInstance(instance util.Instance) error {
	state, err := fileutils.LoadAppState()
	if err != nil {
		return err
	}

	for i, in := range state.Instances {
		if instance.Name == in.Name {
			state.Instances[i] = instance
			return fileutils.SaveAppState(state)
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
	modJson, err2 := fileutils.GetModJsonFromJar(file)
	if err2 != nil {
		return err2
	}

	modData.Version = modJson.Version
	instance.Mods = append(instance.Mods, modData)
	return nil
}

//Must call SaveInstanceData after using! - this allows for batching mod removals into one file write call
func RemoveMod(instance *util.Instance, id string) error {
	mods := instance.Mods
	for i, mod := range mods {
		if mod.Id == id {
			err := os.Remove(instance.Path + "/" + mod.Filename)
			if err != nil {
				return err
			}

			//Remove item
			mods[i] = mods[len(mods)-1]
    		instance.Mods = mods[:len(mods)-1]
			fmt.Println("Uninstalled " + mod.Name)
			return nil
		}
	}
	return errors.New("no mod found")
}

func SetActiveInstance(name string) error {
	state, err := fileutils.LoadAppState()
	if err != nil {
		return err
	}

	state.ActiveInstance = name
	return fileutils.SaveAppState(state)
}

func UpdateInstance(name string) error {
	state, err := fileutils.LoadAppState()
	if err != nil {
		return err
	}

	var instance util.Instance
	for _, i := range state.Instances {
		if strings.EqualFold(i.Name, name) {
			instance = i
			break
		}
	}

	flVersion, err1 := api.GetLatestFabricLoaderVersion()
	if err1 != nil {
		return err1
	}

	//Update fabric installer
	err2 := api.InstallOrUpdateFabricInstaller()
	if err2 != nil {
		return err2
	}

	//Update fabric loader
	if semver.Compare(instance.FabricLoaderVersion, flVersion) == -1 {
		err3 := api.InstallFabricLoader(&state, instance.Version, flVersion)
		if err3 != nil {
			return err3
		}
		instance.FabricLoaderVersion = flVersion
	}

	//Update mods
	for _, mod := range instance.Mods {
		var modData util.ModData
		if mod.Platform == "modrinth" {
			modData, err4 := api.GetModrinthModData(modData.Slug, instance.Version)
			if err4 != nil {
				return err4
			}

			if mod.Id != modData.Id {
				err5 := RemoveMod(&instance, mod.Id)
				if err5 != nil {
					return err5
				}

				err6 := AddMod(&instance, "", modData)
				if err6 != nil {
					return err6
				}
			}
		}
	}	
	return fileutils.SaveAppState(state)
}

func MigrateInstanceToVersion(name string, version string) error {
	err := CreateInstance(name + "_Migrated", version)
	if err != nil {
		return err
	}

	oldInstance, err1 := GetInstance(name)
	if err1 != nil {
		return err1
	}

	newInstance, err2 := GetInstance(name + "_Migrated")
	if err2 != nil {
		return err2
	}

	for _, mod := range oldInstance.Mods {
		err2 := AddMod(&newInstance, mod.Slug, util.ModData{})
		if err2 != nil {
			if err2.Error() == "failed to get mod data" {
				fmt.Println(mod.Name + " does not have a version for " + version)
			} else {
				return err2
			}
		}
	}
	return SaveInstance(newInstance)
}