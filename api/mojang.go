package api

import "log"

type versions struct {
	Latest struct {
		Release string
	}
}

func GetLatestMcVersion() string {
	var versions versions
	_, err := client.R().SetResult(&versions).Get("https://launchermeta.mojang.com/mc/game/version_manifest_v2.json")
	if err != nil {
		log.Fatal(err)
	}
	return versions.Latest.Release
}