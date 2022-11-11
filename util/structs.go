package util

type Dependency struct {
	VersionId string
	ProjectId string
	Name      string
	Required  bool
}

type ModData struct {
	Platform     string
	Slug         string
	Name         string
	ProjectId    string
	Id           string
	Version      string
	Url          string
	Filename     string
	Dependencies []Dependency
}

type Instance struct {
	Name                string
	Path                string
	Version             string
	Mods                []ModData
	FabricLoaderVersion string
}

type Profile struct {
	Name          string
	Type          string
	Icon          string
	LastVersionId string
	Created       string
	JavaArgs      string
	LastUsed      string
}
