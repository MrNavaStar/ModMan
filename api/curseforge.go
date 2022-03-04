package api

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/mrnavastar/modman/util"
)

type curseProject struct {
	Id int
	Name string
	Slug string
}

type file struct {
	Id int
	GameVersionDateReleased string
	DownloadUrl string
	FileName string
	GameVersion []string
	Modules []struct {
		Foldername string
	}
	Dependencies []struct {
		AddonId int
	}
}

var CURSE_API_BASE = "https://addons-ecs.forgesvc.net/api/v2"

func GetCurseModData(slug string, version string) (m util.ModData, e error) {
	var project curseProject
	if  _, err := strconv.Atoi(slug); err != nil {
		var curseProjects []curseProject
		_, err1 := client.R().SetResult(&curseProjects).SetHeader("content-type", "application/json").Get(CURSE_API_BASE + "/addon/search?gameId=432&searchfilter=" + slug)
		util.Fatal(err1)

		for _, p := range curseProjects {
			if p.Slug == slug {
				project = p
			}
		}
	} else {
		_, err1 := client.R().SetResult(&project).Get(CURSE_API_BASE + "/addon/" + slug)
		util.Fatal(err1)
	}

	var files []file
	resp, err1 := client.R().SetResult(&files).Get(CURSE_API_BASE + "/addon/" + fmt.Sprint(project.Id) + "/files")
	util.Fatal(err1)

	if resp.StatusCode() != 200 {
		return util.ModData{}, errors.New("invalid slug")
	}

	var file file
	var date time.Time
	for _, f := range files {
		for _, module := range f.Modules {
			if module.Foldername == "fabric.mod.json" {
				for _, v := range f.GameVersion {
					if v == version {
						t, err2 := time.Parse(time.RFC3339, f.GameVersionDateReleased)
						util.Fatal(err2)

						if date.Before(t) {
							file = f
							date = t
						}
					} 
				}
			}
		}
	}
	
	if file.DownloadUrl == "" {
		return util.ModData{}, errors.New("failed to find matching version")
	}
	
	var modData util.ModData
	modData.Platform = "curse"
	modData.ProjectId = fmt.Sprint(project.Id)
	modData.Id = fmt.Sprint(file.Id)
	modData.Name = project.Name
	modData.Slug = project.Slug
	modData.Url = file.DownloadUrl
	modData.Filename = file.FileName
	
	for _, mod := range file.Dependencies {
		modData.Dependencies = append(modData.Dependencies, fmt.Sprint(mod.AddonId))
	}
	return modData, nil
}