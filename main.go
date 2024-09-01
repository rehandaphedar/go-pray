package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var configDir string
var cacheDir string

var config map[string]string

var cachePath string
var configPath string

type Timings map[string]string

var salahNames = []string{"Fajr", "Sunrise", "Dhuhr", "Asr", "Maghrib", "Isha"}

func main() {
	initialiseDirectories()

	initialiseViper()
	readViperConfig()

	cache := getCache()

	for true {
		nextSalah, durationToNext := computeDurationToNext(time.Now(), cache)

		performCustomActions(int(durationToNext.Seconds()), nextSalah)
		fmt.Printf("%s in %s\n", nextSalah, formatDuration(durationToNext))

		time.Sleep(time.Second)
	}
}

func initialiseDirectories() {
	configRoot, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Error finding config directory: %v \n", err)
	}
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		log.Fatalf("Error finding cache directory: %v \n", err)
	}

	configDir = filepath.Join(configRoot, "go-pray")
	cacheDir = filepath.Join(cacheRoot, "go-pray")

	err = os.MkdirAll(configDir, os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating config directory: %v", err)
	}

	err = os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating cache directory: %v", err)
	}

	cachePath = filepath.Join(cacheDir, "cache.json")
	configPath = filepath.Join(configDir, "config.yaml")
}

func initialiseViper() {
	viper.SetDefault("city", "New York")
	viper.SetDefault("country", "USA")
	viper.SetDefault("method", "1")
	viper.SetDefault("school", "0")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)
}

func readViperConfig() {
	err := viper.ReadInConfig()
	if err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			viper.WriteConfigAs(configPath)
			viper.ReadInConfig()
		default:
			log.Fatalf("Error reading config file: %v \n", err)
		}
	}

	// Here a temporary `struct` is made, then converted to a `map[string]string` so that it's values can be looped over. A bit inefficient, but it works for now.
	temporaryConfig := &struct {
		City    string `json:"city"`
		Country string `json:"country"`
		Method  string `json:"method"`
		School  string `json:"school"`
	}{}
	viper.Unmarshal(temporaryConfig)

	data, err := json.Marshal(temporaryConfig)
	if err != nil {
		log.Fatalf("Error marshalling config: %v \n", err)
	}

	json.Unmarshal(data, &config)

	viper.WriteConfig()
}

func getCache() map[string]Timings {
	if isConfigChanged() {
		return fetchFreshCache()
	}
	return loadExistingCache()
}

func isConfigChanged() bool {
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return true
	}

	previousConfigPath := filepath.Join(configDir, "previous.yaml")
	if _, err := os.Stat(previousConfigPath); os.IsNotExist(err) {
		viper.WriteConfigAs(previousConfigPath)
		return true
	}

	previousConfigFile, err := os.Open(previousConfigPath)
	if err != nil {
		log.Fatalf("Error opening previous config file: %v", err)
	}

	previousConfigContents, err := io.ReadAll(previousConfigFile)
	if err != nil {
		log.Fatalf("Error reading previous config file: %v", err)
	}

	previousConfig := make(map[string]string)
	yaml.Unmarshal(previousConfigContents, previousConfig)

	eq := reflect.DeepEqual(config, previousConfig)
	viper.WriteConfigAs(previousConfigPath)

	return !eq
}

func fetchFreshCache() map[string]Timings {
	endpointURL, err := url.Parse("http://api.aladhan.com/v1/calendarByCity")
	if err != nil {
		log.Fatalf("Error while parsing URL: %v \n", err)
	}

	params := url.Values{}
	for k, v := range config {
		params.Add(k, v)
	}
	params.Add("annual", "true")
	endpointURL.RawQuery = params.Encode()

	response, err := http.Get(endpointURL.String())
	if err != nil {
		log.Fatalf("Error while fetching URL: %v \n", err)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Error while reading response body: %v \n", err)
	}

	var rawCache map[string]interface{}
	json.Unmarshal(body, &rawCache)
	data := rawCache["data"].(map[string]interface{})
	cache := formatCache(data)

	cacheBytes, err := json.Marshal(cache)
	if err != nil {
		log.Fatalf("error while marshalling cache: %v \n", err)
	}

	os.WriteFile(cachePath, cacheBytes, os.ModePerm)

	return cache
}

func formatCache(data map[string]interface{}) map[string]Timings {
	cache := make(map[string]Timings)

	for _, cacheMonth := range data {
		for _, cacheDayInterface := range cacheMonth.([]interface{}) {
			cacheDay := cacheDayInterface.(map[string]interface{})

			gregorian := cacheDay["date"].(map[string]interface{})["gregorian"].(map[string]interface{})
			date := gregorian["date"].(string)

			cacheDayTimings := cacheDay["timings"].(map[string]interface{})

			timings := make(Timings)

			for _, salahName := range salahNames {
				timings[salahName] = cacheDayTimings[salahName].(string)[0:5]
			}

			cache[date] = timings
		}
	}

	return cache
}

func loadExistingCache() map[string]Timings {
	cacheFile, err := os.Open(cachePath)
	if err != nil {
		log.Fatalf("Error opening cache file: %v", err)
	}

	cacheBytes, err := io.ReadAll(cacheFile)
	if err != nil {
		log.Fatalf("Error reading cache file: %v", err)
	}

	var cache map[string]Timings
	json.Unmarshal(cacheBytes, &cache)

	return cache
}

func computeDurationToNext(current time.Time, cache map[string]Timings) (string, time.Duration) {
	cacheToday := cache[current.Format("02-01-2006")]
	cacheTomorrow := cache[current.Add(time.Hour*24).Format("02-01-2006")]

	for _, salahName := range salahNames {
		salahTimeString := cacheToday[salahName]
		salahTime := parseSalahTimeString(salahTimeString)

		delta := salahTime.Sub(current)
		if delta > 0 {
			return salahName, delta
		}
	}

	salahName := salahNames[0]
	salahTimeString := cacheTomorrow[salahName]
	salahTime := parseSalahTimeString(salahTimeString).Add(time.Hour * 24)

	delta := salahTime.Sub(current)
	return salahName, delta
}

func parseSalahTimeString(salahTimeString string) time.Time {
	salahTimeOnly, err := time.Parse("15:04", salahTimeString)
	if err != nil {
		log.Fatalf("Error parsing time string: %v \n", err)
	}

	salahTimeWithDate := time.Date(
		time.Now().Year(),
		time.Now().Month(),
		time.Now().Day(),
		salahTimeOnly.Hour(),
		salahTimeOnly.Minute(),
		salahTimeOnly.Second(),
		salahTimeOnly.Nanosecond(),
		time.Now().Location())

	return salahTimeWithDate
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	h := d / time.Hour
	d -= h * time.Hour

	m := d / time.Minute
	d -= m * time.Minute

	s := d / time.Second

	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func performCustomActions(durationToNext int, nextSalah string) {
	if durationToNext == 0 {
		beeep.Notify(fmt.Sprintf("%s Prayer Time", nextSalah), "", "")
	}
}
