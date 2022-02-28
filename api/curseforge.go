package api

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mrnavastar/modman/util"
)

type response struct {
	Data struct {
		Addons []struct {
			Id string
			Name string
			GameVersionLatestFiles []struct {
				GameVersion string 
				ProjectFileId string
			}
		}
	}
}

type curseVersion struct {
	Id string
	DownloadUrl string
	FileName string
	Modules []struct {
		Foldername string
	}
}

func GetCurseModData(slug string, version string) (modData util.ModData, error error) {
	var query string
	if  _, err := strconv.Atoi(slug); err == nil {
		query = "{addons(id: " + slug + ") {name id gameVersionLatestFiles {gameVersion projectFileId}}}"
	} else {
		query = "{addons(slug: \"" + slug + "\") {name id gameVersionLatestFiles {gameVersion projectFileId}}}"
	}

	var data = "{\"query\":\"" + query + "\"}"

	fmt.Println(data)
	
	var response response
	resp, err := client.R().
			SetBody(data).
			//SetResult(&response).
			Post("https://curse.nikky.moe/graphql")

	fmt.Println(err)
	fmt.Println(resp)
	//fmt.Println(data)

	if err == nil {
		if len(response.Data.Addons) > 0 {
			curseProject := response.Data.Addons[0]	

			for _, modVersion := range curseProject.GameVersionLatestFiles {
				if modVersion.GameVersion == version {
					var curseVersion curseVersion
					_, err1 := client.R().SetResult(&curseVersion).Get("https://curse.nikky.moe/api/addon/" + curseProject.Id + "/file/" + modVersion.ProjectFileId)
					
					if err1 == nil {
						for _, module := range curseVersion.Modules {
							if module.Foldername == "fabric.mod.json" {
								var modData util.ModData
								modData.Platform = "curse"
								modData.Id = curseVersion.Id
								modData.Name = strings.Replace(curseProject.Name, " ", "-", -1)
								modData.Url = curseVersion.DownloadUrl
								modData.Filename = curseVersion.FileName
								return modData, nil
							}
						}
					}
				}
			}
		}
	}
	return util.ModData{}, errors.New("failed to get mod data")
}