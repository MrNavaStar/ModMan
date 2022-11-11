package api

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/mrnavastar/modman/util"
)

var MODRINTH_API_BASE = "https://api.modrinth.com/v2"

type modrinthProject struct {
	Title string
	Id    string
}

type modrinthVersion struct {
	Id            string
	Game_versions []string
	Loaders       []string
	Files         []struct {
		Url      string
		Filename string
	}
	Dependencies []struct {
		Version_id      string
		Project_id      string
		File_name       string
		Dependency_type string
	}
}

func GetModrinthModData(slug string, version string) (m util.ModData, e error) {
	var project modrinthProject
	var versions []modrinthVersion

	resp, err := client.R().SetResult(&project).Get(MODRINTH_API_BASE + "/project/" + slug)
	util.Fatal(err)

	resp1, err1 := client.R().SetResult(&versions).Get(MODRINTH_API_BASE + "/project/" + slug + "/version")
	util.Fatal(err1)

	if resp.StatusCode() != 200 && resp1.StatusCode() != 200 {
		return util.ModData{}, errors.New("invalid slug")
	}

	for _, modVersion := range versions {
		if util.Contains(modVersion.Loaders, "fabric") && util.Contains(modVersion.Game_versions, version) {
			var modData = util.ModData{
				Platform:  "modrinth",
				Slug:      slug,
				ProjectId: project.Id,
				Id:        modVersion.Id,
				Name:      strings.Replace(project.Title, " ", "-", -1),
				Url:       modVersion.Files[0].Url,
				Filename:  modVersion.Files[0].Filename,
			}

			for _, mod := range modVersion.Dependencies {
				var dep = util.Dependency{
					VersionId: mod.Version_id,
					ProjectId: mod.Project_id,
					Name:      mod.File_name,
					Required:  mod.Dependency_type == "required",
				}
				modData.Dependencies = append(modData.Dependencies, dep)
			}

			return modData, nil
		}
	}
	return util.ModData{}, errors.New("failed to find matching version")
}

type searchResult struct {
	Hits []struct {
		Slug       string
		Categories []string
	}
}

func SearchModrinth(query string) (s string, e error) {
	resp, err := client.NewRequest().Get(MODRINTH_API_BASE + "/search?query=" + query)
	util.Fatal(err)

	var search searchResult
	json.Unmarshal([]byte(resp.String()), &search)
	if len(search.Hits) > 0 && util.Contains(search.Hits[0].Categories, "fabric") {
		return search.Hits[0].Slug, nil
	}
	return "", errors.New("no mod found")
}
