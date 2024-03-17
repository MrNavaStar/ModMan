package api

import (
	"io/ioutil"
	"os"

	"github.com/mrnavastar/modman/util"
	"github.com/mrnavastar/modman/util/fileutils"
	"github.com/pterm/pterm"
)

func GetLatestQuiltLoaderVersion() string {
	var loaderVersions []Version
	_, err := client.R().SetResult(&loaderVersions).Get("https://meta.quiltmc.org/v3/versions/loader")
	util.Fatal(err)

	return loaderVersions[0].Version
}

func DownloadQuiltJson(state *fileutils.State, gameVersion string, loaderVersion string) {
	response, err := client.R().Get("https://meta.quiltmc.org/v3/versions/loader/" + gameVersion + "/" + loaderVersion + "/profile/json")
	util.Fatal(err)

	profileName := "quilt-loader-" + loaderVersion + "-" + gameVersion

	dir := state.DotMinecraft + "/versions/" + profileName
	if _, err1 := os.Stat(dir + "/" + profileName + ".json"); os.IsNotExist(err1) {
		util.Fatal(os.MkdirAll(dir, 0700))
	} else {
		return
	}

	err2 := ioutil.WriteFile(dir+"/"+profileName+".json", response.Body(), 0644)
	util.Fatal(err2)
}

func IsQuiltVersionSupported(version string) bool {
	var versions []Version
	_, err := client.R().SetResult(&versions).Get("https://meta.quiltmc.org/v3/versions/game")
	if err != nil {
		pterm.Fatal.Println(err)
	}

	for _, v := range versions {
		if v.Version == version {
			return true
		}
	}
	return false
}
