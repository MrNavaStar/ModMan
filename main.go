package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/common-nighthawk/go-figure"
	"github.com/mrnavastar/modman/api"
	"github.com/mrnavastar/modman/services"
	"github.com/mrnavastar/modman/util"
	"github.com/mrnavastar/modman/util/fileutils"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

type failedMod struct {
	UserIn string
	Slug string
}

func main() {
	app := &cli.App{
		Name: "ModMan",
		Usage: "Manage your mods with ease",
		Commands: []*cli.Command {
			{
				Name: "init",
				Usage: "Setup modman on your system",
				Action: func(c *cli.Context) error {
					pterm.DefaultCenter.Println(pterm.FgLightCyan.Sprint(figure.NewFigure("ModMan", "speed", true)))

					pterm.DefaultCenter.Print(pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgGreen)).WithMargin(10).Sprint("V1.0 - Created By MrNavaStar"))
					pterm.DefaultCenter.WithCenterEachLineSeparately().Println("Welcome!\nHelp contribute to this project over at:\nGit: https://github.com/MrNavaStar/ModMan\nIssues: https://github.com/MrNavaStar/ModMan/issues")
					
					reader := bufio.NewReader(os.Stdin)
					pterm.Info.Println("Enter the path to your .minecraft folder:")
					pterm.FgDarkGray.Print(">>> ")
					workingDir, _ := reader.ReadString('\n')
					workingDir = strings.Replace(workingDir, "\n", "", -1)
					
					fileutils.Setup(workingDir)
					err := api.InstallOrUpdateFabricInstaller()
					if err != nil {
						return err
					}
					pterm.Success.Println("Setup complete")
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

					if len(state.Instances) == 0 {
						return nil
					}

					fmt.Println()
					var instances [][]string
					instances = append(instances, []string{" ", "Name", "Version", "Mods"})
					for _, instance := range state.Instances {
						var prefix string
						if state.ActiveInstance == instance.Name {
							prefix = ">"
						}

						instances = append(instances, []string{prefix, instance.Name, instance.Version, fmt.Sprint(len(instance.Mods))})
					}
					pterm.DefaultTable.WithHasHeader().WithData(pterm.TableData(instances)).Render()
					fmt.Println()
					return nil
				},
			},
			{
				Name: "make",
				Usage: "Create a  new instance",
				Action: func(c *cli.Context) error {
					name := c.Args().Get(0)
					version := c.Args().Get(1)
					_, err := services.GetInstance(name)
					if err == nil {
						pterm.Error.Println("Instance with that name already exists")
						return nil
					}

					if err.Error() != "failed to find instance" {
						return err
					} 
					
					if version == "" {
						version = api.GetLatestMcVersion()
					}

					pterm.Info.Println("Creating " + name)
					err1 := services.CreateInstance(name, version)
					if err1 != nil {
						return err1
					}

					pterm.Success.Println("Created " + name)
					return services.SetActiveInstance(name)
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

					pterm.Info.Println("Now modifying " + instance.Name)
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
							pterm.Error.Println("Failed to find an instance with that name")
							return nil
						}
						return err
					}

					reader := bufio.NewReader(os.Stdin)
					pterm.Info.Print("Are you sure? [y/N]: ")
					input, _ := reader.ReadString('\n')
					input = strings.Replace(input, "\n", "", -1)

					if input == "y" || input == "Y" {
						err1 := services.DeleteInstance(args.Get(0))
						if err1 != nil {
							return err1
						}
						pterm.Success.Println("Removed " + args.Get(0))
					} else {
						pterm.Warning.Println("Action canceled")
					}
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
					
					var retrymods []failedMod
					mods := args.Slice()
					prefix := ""
					for i := 0; i < len(mods); i++ {
						mod := mods[i]

						if mod == "-c" {
							prefix = "c="
							continue
						}

						err2 := services.AddMod(&instance, prefix + mod, util.ModData{})
						if err2 != nil {
							if err2.Error() == "mod already added" {
								pterm.Error.Println(mod + " has already been added")
								continue
							}

							if err2.Error() == "failed to get mod data" {
								slug, err3 := api.SearchModrinth(mod)
								if err3 != nil && err3.Error() == "no mod found" {
									fmt.Println("Could not find mod under " + mod)
									continue
								}

								var failedMod failedMod
								failedMod.UserIn = mod
								failedMod.Slug = slug
								retrymods = append(retrymods, failedMod)
								continue
							}
						}
						pterm.Success.Println("Installed " + mod)
					}

					for _, mod := range retrymods {
						pterm.Error.Println("Failed to find mod under " + mod.UserIn)
						reader := bufio.NewReader(os.Stdin)
						pterm.Info.Print("Would you like to try under " + mod.Slug + "? [Y/n]: ")
						input, _ := reader.ReadString('\n')
						input = strings.Replace(input, "\n", "", -1)

						if input == "Y" || input == "y" || input == "" {
							err3 := services.AddMod(&instance, mod.Slug, util.ModData{})
							if err3 != nil && err3.Error() == "mod already added"{
								pterm.Error.Println(mod.Slug + " has already been added")
								continue
							}
							pterm.Success.Println("Installed " + mod.Slug)
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
							if strings.EqualFold(modData.Name, mod) || strings.EqualFold(modData.Slug, mod) {
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

					if len(instance.Mods) == 0 { 
						return nil
					}
					
					fmt.Println()
					var mods [][]string
					mods = append(mods, []string{"Name", "Version", "Filename"})
					for _, mod := range instance.Mods {
						mods = append(mods, []string{mod.Name, mod.Version, mod.Filename})
					}
					pterm.DefaultTable.WithHasHeader().WithData(pterm.TableData(mods)).Render()
					fmt.Println()
					return nil
				},
			},
			{
				Name: "update",
				Usage: "update an instance",
				Action: func(c *cli.Context) error {
					state, err := fileutils.LoadAppState()
					if err != nil {
						return err
					}

					pterm.Info.Println("Updating " + state.ActiveInstance)
					services.UpdateInstance(state.ActiveInstance)
					pterm.Success.Println("Done.")
					return nil
				},
			},
			{
				Name: "migrate",
				Usage: "migrate an instance to a newer game version",
				Action: func(c *cli.Context) error {
					state, err := fileutils.LoadAppState()
					if err != nil {
						return err
					} 
					
					pterm.Info.Println("Migrating " + state.ActiveInstance + " to " + c.Args().Get(0))
					err1 := services.MigrateInstanceToVersion(state.ActiveInstance, c.Args().Get(0))
					if err1 != nil {
						return err1
					}
					pterm.Success.Println("Migration Complete")
					return services.SetActiveInstance(state.ActiveInstance + "_Migrated")
				},
			},
		},
	}
	
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}