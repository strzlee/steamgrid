// Automatically downloads and configures Steam grid images for all games in a
// given Steam installation.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

// Prints an error and quits.
func errorAndExit(err error) {
	fmt.Println(err.Error())
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	os.Exit(0)
}

func startApplication(config *Config) {
	artStyles := map[string][]string{
		// BannerLQ: 460 x 215
		// BannerHQ: 920 x 430
		// CoverLQ: 300 x 450
		// CoverHQ: 600 x 900
		// HeroLQ: 1920 x 620
		// HeroHQ: 3840 x 1240
		// LogoLQ: 640 x 360
		// LogoHQ: 1280 x 720
		// artStyle: ["idExtension", "nameExtension", steamExtension, dimXHQ, dimYHQ, dimXLQ, dimYLQ]
		"Banner": []string{"", ".banner", "header.jpg", "920", "430", "460", "215"},
		"Cover": []string{"p", ".cover", "library_600x900_2x.jpg", "600", "900", "300", "450"},
		"Hero": []string{"_hero", ".hero", "library_hero.jpg" , "3840", "1240", "1920", "620"},
		"Logo": []string{"_logo", ".logo", "logo.png", "1280", "720", "640", "360"},
	}

	steamGridFilter := "?styles=" + config.SteamGridDBArtstyle + "&types=" + config.SteamGridDBType

	fmt.Println("Loading overlays...")
	overlays, err := LoadOverlays(filepath.Join(filepath.Dir(os.Args[0]), "overlays by category"), artStyles)
	if err != nil {
		errorAndExit(err)
	}
	if len(overlays) == 0 {
		fmt.Println("No category overlays found. You can put overlay images in the folder 'overlays by category', where the filename is the game category.\n\nYou can find many user-created overlays at https://www.reddit.com/r/steamgrid/wiki/overlays .\n\nContinuing without overlays...\n")
	} else {
		fmt.Printf("Loaded %v overlays. \n\nYou can find many user-created overlays at https://www.reddit.com/r/steamgrid/wiki/overlays .\n\n", len(overlays))
	}

	fmt.Println("Looking for Steam directory...\nIf SteamGrid doesn´t find the directory automatically, launch it with an argument linking to the Steam directory.")
	installationDir, err := GetSteamInstallation(config.SteamPath)
	if err != nil {
		errorAndExit(err)
	}

	fmt.Println("Loading users...")
	users, err := GetUsers(installationDir)
	if err != nil {
		errorAndExit(err)
	}
	if len(users) == 0 {
		errorAndExit(errors.New("No users found at Steam/userdata. Have you used Steam before in this computer?"))
	}

	nOverlaysApplied := 0
	nDownloaded := 0
	notFounds := map[string][]*Game{
		"Banner": []*Game{},
		"Cover": []*Game{},
		"Hero": []*Game{},
		"Logo": []*Game{},
	}
	steamGridDB := map[string][]*Game{
		"Banner": []*Game{},
		"Cover": []*Game{},
		"Hero": []*Game{},
		"Logo": []*Game{},
	}
	IGDB := map[string][]*Game{
		"Banner": []*Game{},
		"Cover": []*Game{},
		"Hero": []*Game{},
		"Logo": []*Game{},
	}
	searchedGames := map[string][]*Game{
		"Banner": []*Game{},
		"Cover": []*Game{},
		"Hero": []*Game{},
		"Logo": []*Game{},
	}
	failedGames := map[string][]*Game{
		"Banner": []*Game{},
		"Cover": []*Game{},
		"Hero": []*Game{},
		"Logo": []*Game{},
	}
	var errorMessages []string

	for _, user := range users {
		fmt.Println("Loading games for " + user.Name)
		gridDir := filepath.Join(user.Dir, "config", "grid")

		err = os.MkdirAll(filepath.Join(gridDir, "originals"), 0777)
		if err != nil {
			errorAndExit(err)
		}

		games := GetGames(user, config.OnlyNonSteamGames)

		fmt.Println("Loading existing images and backups...")

		i := 0
		for _, game := range games {
			i++

			var name string
			if game.Name == "" {
				game.Name = GetGameName(game.ID)
			}

			if game.Name != "" {
				name = game.Name
			} else {
				name = "unknown game with id " + game.ID
			}
			fmt.Printf("Processing %v (%v/%v)\n", name, i, len(games))

			for artStyle, artStyleExtensions := range artStyles {
				// Clear for multiple runs:
				game.ImageSource = ""
				game.ImageExt = ""
				game.CleanImageBytes = nil
				game.OverlayImageBytes = nil

				overridePath := filepath.Join(filepath.Dir(os.Args[0]), "games")
				LoadExisting(overridePath, gridDir, game, artStyleExtensions)
				// This cleans up unused backups and images for the same game but with different extensions.
				err = RemoveExisting(gridDir, game.ID, artStyleExtensions)
				if err != nil {
					fmt.Println(err.Error())
				}

				///////////////////////
				// Download if missing.
				///////////////////////
				if game.ImageSource == "" {
					from, err := DownloadImage(gridDir, game, artStyle, artStyleExtensions, config.SkipSteam, config.SteamGridDBAPIKey, steamGridFilter, config.IGDBAPIKey, config.SkipGoogle)
					if err != nil && err.Error() == "SteamGridDB authorization token is missing or invalid" {
						// Wrong api key
						config.SteamGridDBAPIKey = ""
						fmt.Println(err.Error())
					} else if err != nil {
						fmt.Println(err.Error())
					}

					if game.ImageSource == "" {
						notFounds[artStyle] = append(notFounds[artStyle], game)
						fmt.Printf("%v not found\n", artStyle)
						// Game has no image, skip it.
						continue
					} else if err == nil {
						nDownloaded++
					}

					switch from {
					case "IGDB":
						IGDB[artStyle] = append(IGDB[artStyle], game)
					case "SteamGridDB":
						steamGridDB[artStyle] = append(steamGridDB[artStyle], game)
					case "search":
						searchedGames[artStyle] = append(searchedGames[artStyle], game)
					}
				}
				fmt.Printf("%v found from %v\n", artStyle, game.ImageSource)

				///////////////////////
				// Apply overlay.
				//
				// Expecting name.artExt.imgExt:
				// Banner: favorites.png
				// Cover: favorites.p.png
				// Hero: favorites.hero.png
				// Logo: favorites.logo.png
				///////////////////////
				err := ApplyOverlay(game, overlays, artStyleExtensions)
				if err != nil {
					print(err.Error(), "\n")
					failedGames[artStyle] = append(failedGames[artStyle], game)
					errorMessages = append(errorMessages, err.Error())
				}
				if game.OverlayImageBytes != nil {
					nOverlaysApplied++
				} else {
					game.OverlayImageBytes = game.CleanImageBytes
				}

				///////////////////////
				// Save result.
				///////////////////////
				err = BackupGame(gridDir, game, artStyleExtensions)
				if err != nil {
					errorAndExit(err)
				}

				imagePath := filepath.Join(gridDir, game.ID + artStyleExtensions[0] + game.ImageExt)
				err = ioutil.WriteFile(imagePath, game.OverlayImageBytes, 0666)

				// Copy with legacy naming for Big Picture mode
				if artStyle == "Banner" {
					id, err := strconv.ParseUint(game.ID, 10, 64)
					if err == nil {
						imagePath := filepath.Join(gridDir, strconv.FormatUint(id<<32|0x02000000, 10) + artStyleExtensions[0] + game.ImageExt)
						err = ioutil.WriteFile(imagePath, game.OverlayImageBytes, 0666)
					}
				}
				if err != nil {
					fmt.Printf("Failed to write image for %v (%v) because: %v\n", game.Name, artStyle, err.Error())
				}
			}
		}
	}

	fmt.Printf("\n\n%v images downloaded and %v overlays applied.\n\n", nDownloaded, nOverlaysApplied)
	if len(searchedGames["Banner"]) + len(searchedGames["Cover"]) + len(searchedGames["Hero"]) + len(searchedGames["Logo"]) >= 1 {
		fmt.Printf("%v images were found with a Google search and may not be accurate:\n", len(searchedGames["Banner"]) + len(searchedGames["Cover"]) + len(searchedGames["Hero"]) + len(searchedGames["Logo"]))
		for artStyle, games := range searchedGames {
			for _, game := range games {
				fmt.Printf("* %v (steam id %v, %v)\n", game.Name, game.ID, artStyle)
			}
		}

		fmt.Printf("\n\n")
	}

	if len(IGDB["Banner"]) + len(IGDB["Cover"]) >= 1 {
		fmt.Printf("%v images were found on IGDB and may not be in full quality or accurate:\n", len(IGDB["Banner"]) + len(IGDB["Cover"]))
		for artStyle, games := range IGDB {
			for _, game := range games {
				fmt.Printf("* %v (steam id %v, %v)\n", game.Name, game.ID, artStyle)
			}
		}

		fmt.Printf("\n\n")
	}

	if len(steamGridDB["Banner"]) + len(steamGridDB["Cover"]) + len(steamGridDB["Hero"]) + len(steamGridDB["Logo"]) >= 1 {
		fmt.Printf("%v images were found on SteamGridDB and may not be in full quality or accurate:\n", len(steamGridDB["Banner"]) + len(steamGridDB["Cover"]) + len(steamGridDB["Hero"]) + len(steamGridDB["Logo"]))
		for artStyle, games := range steamGridDB {
			for _, game := range games {
				fmt.Printf("* %v (steam id %v, %v)\n", game.Name, game.ID, artStyle)
			}
		}

		fmt.Printf("\n\n")
	}

	if len(notFounds["Banner"]) + len(notFounds["Cover"]) + len(notFounds["Hero"]) + len(notFounds["Logo"]) >= 1 {
		fmt.Printf("%v images could not be found anywhere:\n", len(notFounds["Banner"]) + len(notFounds["Cover"]) + len(notFounds["Hero"]) + len(notFounds["Logo"]))
		for artStyle, games := range notFounds {
			for _, game := range games {
				fmt.Printf("- %v (id %v, %v)\n", game.Name, game.ID, artStyle)
			}
		}

		fmt.Printf("\n\n")
	}

	if len(failedGames["Banner"]) + len(failedGames["Cover"]) + len(failedGames["Hero"]) + len(failedGames["Logo"]) >= 1 {
		fmt.Printf("%v images were found but had errors and could not be overlaid:\n", len(failedGames["Banner"]) + len(failedGames["Cover"]) + len(failedGames["Hero"]) + len(failedGames["Logo"]))
		for artStyle, games := range failedGames {
			var i = 0
			for _, game := range games {
				fmt.Printf("- %v (id %v, %v) (%v)\n", game.Name, game.ID, artStyle, errorMessages[i])
				i++
			}
		}

		fmt.Printf("\n\n")
	}

	fmt.Println("Open Steam in grid view to see the results!\n\nPress enter to close.")

	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
