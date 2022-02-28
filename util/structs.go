package util

type ModData struct {
	Platform string
	Slug string
	Name string
	Id string
	Version string
	Url string
	Filename string
}

type Instance struct {
	Name string
	Path string
	Version string
	Mods []ModData
	FabricLoaderVersion string
}

type Profile struct {
	Name string
	Type string
	Icon string
	LastVersionId string
	Created string
	JavaArgs string
	LastUsed string
}