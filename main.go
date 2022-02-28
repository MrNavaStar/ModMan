package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/text"
	"github.com/mrnavastar/modman/api"
	"github.com/mrnavastar/modman/services"
	"github.com/mrnavastar/modman/util"
	"github.com/mrnavastar/modman/util/fileutils"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name: "ModMan",
		Usage: "Manage your mods with ease",
		Commands: []*cli.Command {
			{
				Name: "init",
				Usage: "Setup modman on your system",
				Action: func(c *cli.Context) error {
					fileutils.Setup(c.Args().Get(0))
					err := api.InstallOrUpdateFabricInstaller()
					if err != nil {
						return err
					}
					fmt.Println("Done.")
					return nil
				},
			},
			{
				Name: "ls",
				Aliases: []string{"list"},
				Usage:   "List all instances",
				Action:  func(c *cli.Context) error {
					state, err := fileutils.LoadAppState()
					if err != nil {
						return err
					}

					lname := 0
					lversion := 0
					for _, instance := range state.Instances {
						if len(instance.Name) > lname {
							lname = len(instance.Name)
						}
						if len(instance.Version) > lversion {
							lversion = len(instance.Version)
						}
					}

					for _, instance := range state.Instances {
						fmt.Println()
						fmt.Print(text.AlignDefault.Apply("NAME:", lname +2) + text.AlignDefault.Apply("VERSION:", lversion))
						fmt.Println()
						fmt.Println(text.AlignDefault.Apply(text.Bold.Sprintf(instance.Name), lname +2) + text.AlignDefault.Apply(instance.Version, lversion))
						fmt.Println()
					}
					return nil
				},
			},
			{
				Name: "make",
				Usage: "Create a  new instance",
				Action: func(c *cli.Context) error {
					args := c.Args()
					_, err := services.GetInstance(args.Get(0))
					if err == nil {
						fmt.Println("Instance with that name already exists")
						return nil
					}

					if err.Error() != "failed to find instance" {
						return err
					} 

					fmt.Println("Creating " + args.Get(0))
					err1 := services.CreateInstance(args.Get(0), args.Get(1))
					if err1 != nil {
						return err1
					}

					fmt.Println("Done.")
					return services.SetActiveInstance(args.Get(0))
				},
			},
			{
				Name: "mod",
				Usage: "Modify an instance",
				Action: func(c *cli.Context) error {
					instance, err := services.GetInstance(c.Args().Get(0))
					if err != nil {
						return err
					}

					fmt.Println("Now modifying " + instance.Name)
					return services.SetActiveInstance(instance.Name)
				},
			},
			{
				Name: "rm",
				Aliases: []string{"remove"},
				Usage: "Remove an instance",
				Action: func(c *cli.Context) error {
					args := c.Args()
					_, err := services.GetInstance(args.Get(0))
					if err != nil {
						if err.Error() == "failed to find instance" {
							fmt.Println("Failed to find an instance with that name")
							return nil
						}
						return err
					}

					err1 := services.DeleteInstance(args.Get(0))
					if err1 != nil {
						return err1
					}
					fmt.Println("Removed " + args.Get(0))
					return nil
				},
			},
			{
				Name: "install",
				Usage: "Install mods",
				Action: func(c *cli.Context) error {
					args := c.Args()
					state, err := fileutils.LoadAppState()
					if err != nil {
						return err
					}

					instance, err1 := services.GetInstance(state.ActiveInstance)
					if err1 != nil {
						return err
					} 

					for _, mod := range args.Slice() {
						err2 := services.AddMod(&instance, mod, util.ModData{})
						if err2 != nil {
							if err2.Error() == "mod already added" {
								fmt.Println(mod + " has already been added")
								continue
							}

							if err2.Error() == "failed to get mod data" {
								slug, err3 := api.SearchModrinth(mod)
								if err3 != nil && err3.Error() == "no mod found" {
									fmt.Println("Could not find mod under " + mod)
									continue
								}

								err4 := services.AddMod(&instance, slug, util.ModData{})
								if err4 != nil && err4.Error() == "mod already added" {
									fmt.Println(slug + " has already been added")
									continue
								}
							}
						}	
					}
					return services.SaveInstance(instance)
				},
			},
			{
				Name: "uninstall",
				Usage: "Uninstall mods",
				Action: func(c *cli.Context) error {
					args := c.Args()
					state, err := fileutils.LoadAppState()
					if err != nil {
						return err
					}

					instance, err1 := services.GetInstance(state.ActiveInstance)
					if err1 != nil {
						return err
					}

					for _, mod := range args.Slice() {
						for _, modData := range instance.Mods {
							if strings.EqualFold(modData.Name, mod) {
								err2 := services.RemoveMod(&instance, modData.Id)
								if err2 != nil {
									if err2.Error() == "no mod found" {
										continue
									} else {
										return err2
									}
								}
							}
						}
					}
					return services.SaveInstance(instance)
				},
			},
			{
				Name: "lsmod",
				Usage: "list mods installed on the active instance",
				Action: func(c *cli.Context) error {
					state, err := fileutils.LoadAppState()
					if err != nil {
						return err
					}

					instance, err1 := services.GetInstance(state.ActiveInstance)
					if err1 != nil {
						return err
					}

					lname := 0
					lfname := 0
					lversion := 0
					for _, mod := range instance.Mods {
						if len(mod.Name) > lname {
							lname = len(mod.Name)
						}
						if len(mod.Filename) > lfname {
							lfname = len(mod.Filename)
						}
						if len(mod.Version) > lversion {
							lversion = len(mod.Version)
						}
					}
					
					fmt.Println()
					fmt.Println(text.AlignDefault.Apply("NAME:", lname + 2) + text.AlignDefault.Apply("VERSION:", lversion + 2) + text.AlignDefault.Apply("FILENAME:", lfname))
					for _, mod := range instance.Mods {
						fmt.Println(text.AlignDefault.Apply(text.Bold.Sprint(mod.Name), lname + 2) + text.AlignDefault.Apply(text.Underline.Sprint(mod.Version), lversion + 2) + text.AlignDefault.Apply(mod.Filename, lfname))
					}
					fmt.Println()
					return nil
				},
			},
		},
	}
	
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}