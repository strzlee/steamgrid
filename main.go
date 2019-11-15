package main

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"os"
	"net/http"
	"time"

	"fyne.io/fyne/app"
	"fyne.io/fyne/widget"
)

//Config Steamgrid
type Config struct {
	SteamPath           string
	SteamGridDBAPIKey   string
	SteamGridDBArtstyle string
	SteamGridDBType     string
	IGDBAPIKey          string
	OnlyNonSteamGames   bool
	SkipSteam           bool
	SkipGoogle          bool
}

const configFileName = "steamgrid-config.json"

func writeConfigFile(c *Config) error {
	file, err := os.Create(configFileName)      
	if err == nil {
		defer file.Close()
		data, err := json.MarshalIndent(c, "", "  ")
		if err == nil {
				_, err = file.Write(data)
		}
	}
	return err
}

func readConfigFile() *Config {
	var c *Config = new(Config)
    file, err := ioutil.ReadFile(configFileName)
    if err == nil {
        json.Unmarshal(file, c)
    }
	return c
}

func main() {
	http.DefaultTransport.(*http.Transport).ResponseHeaderTimeout = time.Second * 10

	os.Setenv("FYNE_THEME", "light")
	a := app.New()
	a.SetIcon(nil)

	conf := readConfigFile()
	steamPathInput := widget.NewEntry()
	steamPathInput.SetText(conf.SteamPath)
	steamGridDbKeyInput := widget.NewEntry()
	steamGridDbKeyInput.SetText(conf.SteamGridDBAPIKey)
	IGDBKeyInput := widget.NewEntry()
	IGDBKeyInput.SetText(conf.IGDBAPIKey)
	steamGridDbArtstyleInput := widget.NewEntry()
	steamGridDbArtstyleInput.SetText(conf.SteamGridDBArtstyle)
	steamGridDbTypeInput := widget.NewEntry()
	steamGridDbTypeInput.SetText(conf.SteamGridDBType)
	onlyNonSteamCheck := widget.NewCheck("Only Non-Steam-Games", nil)
	onlyNonSteamCheck.SetChecked(conf.OnlyNonSteamGames)
	skipSteamCheck := widget.NewCheck("Skip downloads from Steam servers", nil)
	skipSteamCheck.SetChecked(conf.SkipSteam)
	skipGoogleCheck := widget.NewCheck("Skip search and downloads from Google", nil)
	skipGoogleCheck.SetChecked(conf.SkipGoogle)
	IGDBUrl, _ := url.Parse("https://api.igdb.com/signup")
  steamGridDbURL, _ := url.Parse("https://www.steamgriddb.com/profile/preferences")
  statusLabel := widget.NewLabel("")

	w := a.NewWindow("Steamgrid")

	w.SetContent(widget.NewVBox(
		widget.NewGroup("IGDB",
			widget.NewLabel("IGDB API Key"),
			widget.NewHyperlink("Get API-Key here: https://api.igdb.com/signup", IGDBUrl),
			IGDBKeyInput,
		),
		widget.NewGroup("SteamGridDB",
			widget.NewLabel("SteamGridDB API Key"),
			widget.NewHyperlink("Get API-Key here: https://www.steamgriddb.com/profile/preferences", steamGridDbURL),
			steamGridDbKeyInput,
			widget.NewLabel("SteamGridDB Artstyle (alternate, blurred, white_logo, material, no_logo)"),
			steamGridDbArtstyleInput,
			widget.NewLabel("SteamGridDB Type (static, animated)"),
			steamGridDbTypeInput,
		),
		widget.NewGroup("Settings",
			widget.NewLabel("Steam-Path"),
			steamPathInput,
			onlyNonSteamCheck,
			skipSteamCheck,
			skipGoogleCheck,
		),
		widget.NewButton("Save & Start", func() {
			conf.SteamPath = steamPathInput.Text
			conf.SteamGridDBAPIKey = steamGridDbKeyInput.Text
			conf.SteamGridDBArtstyle = steamGridDbArtstyleInput.Text
			conf.SteamGridDBType = steamGridDbTypeInput.Text
			conf.IGDBAPIKey = IGDBKeyInput.Text
			conf.OnlyNonSteamGames = onlyNonSteamCheck.Checked
			conf.SkipSteam = skipSteamCheck.Checked
			conf.SkipGoogle = skipGoogleCheck.Checked
            err := writeConfigFile(conf)
            if err != nil {
                statusLabel.SetText("Config saved. Steamgrid running...")
            }
			startApplication(conf)
		}),
		widget.NewButton("Exit", func() {
			a.Quit()
    }),
    widget.NewVBox(
    	statusLabel,
    ),
	))

	w.ShowAndRun()
}
