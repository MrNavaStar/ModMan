package api

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/mrnavastar/modman/util"
)

type modrinthProject struct {
	Title string
}

type modrinthVersion struct {
	Id string
	Game_versions []string
	Loaders []string
	Files []struct {
		Url string
		Filename string
	}
}

func GetModrinthModData(slug string, version string) (m util.ModData, e error) {
	var project modrinthProject
	var versions []modrinthVersion
	_, err := client.R().SetResult(&project).Get("https://api.modrinth.com/v2/project/" + slug)
	if err != nil {
		return util.ModData{}, err
	}

	_, err1 := client.R().SetResult(&versions).Get("https://api.modrinth.com/v2/project/" + slug + "/version")
	if err1 != nil {
		return util.ModData{}, err1
	}
	
	for _, modVersion := range versions {
		if util.Contains(modVersion.Loaders, "fabric") && util.Contains(modVersion.Game_versions, version) {
			var modData util.ModData
			modData.Platform = "modrinth"
			modData.Slug = slug
			modData.Id = modVersion.Id
			modData.Name = strings.Replace(project.Title, " ", "-", -1)
			modData.Url = modVersion.Files[0].Url
			modData.Filename = modVersion.Files[0].Filename
			return modData, nil
		}
	}
	return util.ModData{}, errors.New("failed to get mod data")
}

type searchResult struct {
	Hits []struct {
		Slug string
		Categories []string
	}
}

func SearchModrinth(query string) (s string, e error) {
	resp, err := client.NewRequest().Get("https://api.modrinth.com/v2/search?query=" + query)
	if err != nil {
		return "", err
	}

	var search searchResult
	json.Unmarshal([]byte(resp.String()), &search)
	if len(search.Hits) > 0 && util.Contains(search.Hits[0].Categories, "fabric") {
		return search.Hits[0].Slug, nil
	}
	return "", errors.New("no mod found")
}