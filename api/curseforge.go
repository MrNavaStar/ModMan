package api

import (
	"errors"
	"fmt"
	"log"
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
}

func GetCurseModData(slug string, version string) (m util.ModData, e error) {
	var project curseProject
	if  _, err := strconv.Atoi(slug); err != nil {
		var curseProjects []curseProject
		_, err1 := client.R().SetResult(&curseProjects).SetHeader("content-type", "application/json").Get("https://addons-ecs.forgesvc.net/api/v2/addon/search?gameId=432&searchfilter=" + slug)
		if err1 != nil {
			log.Fatal(err1)
		}

		for _, p := range curseProjects {
			if p.Slug == slug {
				project = p
			}
		}
	} else {
		_, err := client.R().SetResult(&project).Get("https://addons-ecs.forgesvc.net/api/v2/addon/" + slug)
		if err != nil {
			log.Fatal(err)
		}
	}

	var files []file
	_, err1 := client.R().SetResult(&files).Get("https://addons-ecs.forgesvc.net/api/v2/addon/" + fmt.Sprint(project.Id) + "/files")
	if err1 != nil {
		log.Fatal(err1)
	}

	var file file
	var date time.Time
	for _, f := range files {
		for _, module := range f.Modules {
			if module.Foldername == "fabric.mod.json" {
				for _, v := range f.GameVersion {
					if v == version {
						t, err2 := time.Parse(time.RFC3339, f.GameVersionDateReleased)
						if err2 != nil {
							log.Fatal(err2)
						}

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
		return util.ModData{}, errors.New("failed to get mod data")
	}
	
	var modData util.ModData
	modData.Platform = "curse"
	modData.Id = fmt.Sprint(file.Id)
	modData.Name = project.Name
	modData.Slug = project.Slug
	modData.Url = file.DownloadUrl
	modData.Filename = file.FileName
	return modData, nil
}