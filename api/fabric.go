package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/mrnavastar/modman/util"
	"github.com/mrnavastar/modman/util/fileutils"
	"github.com/pterm/pterm"
)

type Version struct {
	Version string
	Stable bool
	Url string
}

func GetLatestFabricLoaderVersion() (s string, e error) {
	var loaderVersions []Version
	_, err := client.R().SetResult(&loaderVersions).Get("https://meta.fabricmc.net/v2/versions/loader")
	util.Fatal(err)

	for _, loaderVersion := range loaderVersions {
		if loaderVersion.Stable {
			return loaderVersion.Version, nil
		}
	}
	return "", errors.New("failed to find a stable version")
}

func InstallOrUpdateFabricInstaller() {
	state := fileutils.LoadAppState()

	var installerVersions []Version
	_, err1 := client.R().SetResult(&installerVersions).Get("https://meta.fabricmc.net/v2/versions/installer")
	if err1 != nil {
		pterm.Fatal.Println(err1)
	}

	for _, installerVersion := range installerVersions {
		if installerVersion.Stable && state.FabricInstallerVersion != installerVersion.Version {
			fmt.Println("Installing fabric installer v" + installerVersion.Version)
			fileutils.DownloadFile(installerVersion.Url, state.WorkDir + "/installers/fabric-installer.jar")

			state.FabricInstallerVersion = installerVersion.Version
		}
	}
	fileutils.SaveAppState(state)
}

func InstallFabricLoader(state *fileutils.State, gameVersion string, loaderVersion string) {
	if util.Contains(state.FabricLoaderVersions, loaderVersion + "-" + gameVersion) {
		return 
	}
 
	cmd := exec.Command("java", "-jar", state.WorkDir + "/installers/fabric-installer.jar", "client", "-dir", state.DotMinecraft, "-mcversion", gameVersion, "-loader", loaderVersion, "-noprofile", "-snapshot")
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	err := cmd.Run()
	util.Fatal(err)

	state.FabricLoaderVersions = append(state.FabricLoaderVersions, loaderVersion + "-" + gameVersion)
}

func IsVersionSupported(version string) bool {
	var versions []Version
	_, err := client.R().SetResult(&versions).Get("https://meta.fabricmc.net/v2/versions/game")
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